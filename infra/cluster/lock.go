package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/utils"
	etcd "go.etcd.io/etcd/client/v3"
)

var (
	ErrLockIsLocked = errors.New("etcd lock locked")
	ErrLockFaild    = errors.New("etcd lock faild")
	ErrLockTimeout  = errors.New("etcd lock timeout")
)

const (
	lockPrefix = "/cluster/locks"
)

type waitLockManager struct {
	waitQueue map[string][]*globalLock
	mtx       sync.Mutex
	watcher   etcd.WatchChan
	ctx       context.Context
}

func newWaitLockManager(ctx context.Context, cli *etcd.Client) *waitLockManager {
	watcher := cli.Watch(ctx, fmt.Sprintf("%s/", lockPrefix), etcd.WithPrefix())
	manager := &waitLockManager{
		waitQueue: make(map[string][]*globalLock),
		watcher:   watcher,
		ctx:       ctx,
	}
	go manager.watch()
	return &waitLockManager{
		waitQueue: make(map[string][]*globalLock),
	}
}

func (m *waitLockManager) watch() {
	defer utils.RecoverPanic()
	for {
		select {
		case <-m.ctx.Done():
			return
		case resp := <-m.watcher:
			for _, ev := range resp.Events {
				if ev.Type != etcd.EventTypeDelete {
					continue
				}
				lockKey := string(ev.Kv.Key)
				m.mtx.Lock()
				locks := m.waitQueue[lockKey]
				if len(locks) == 0 {
					m.mtx.Unlock()
					continue
				}
				locks[0].TryLock()
				if locks[0].Locked() {
					locks = locks[1:]
					m.waitQueue[lockKey] = locks
				}
				m.mtx.Unlock()
			}
		}
	}
}

func (m *waitLockManager) subscribe(gl *globalLock) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	locks := m.waitQueue[gl.lockKey]
	for _, lock := range locks {
		if lock == gl {
			zlog.Errorf("lock is already in wait queue %s", gl.lockKey)
			return
		}

	}
	m.waitQueue[gl.lockKey] = append(m.waitQueue[gl.lockKey], gl)
}

func etcdLockKey(name string) string {
	return fmt.Sprintf("%s/%s", lockPrefix, name)
}

type globalLock struct {
	locked    atomic.Bool
	lockKey   string
	cancelCtx utils.IContextWithCancel
	awake     chan struct{}
}

func NewLock(name string) *globalLock {
	return &globalLock{
		lockKey:   etcdLockKey(name),
		cancelCtx: utils.ContextWithCancel(context.Background()),
	}
}

func (l *globalLock) TryLock() bool {
	if l.locked.Load() {
		panic(ErrLockIsLocked)
	}
	cli := clusterNode.etcdClient
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	lease, err := clusterNode.etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		zlog.Errorf("etcd grant error: %v", err)
		return false
	}
	txn := cli.Txn(ctx).If(etcd.Compare(etcd.Version(l.lockKey), "=", 0)).
		Then(etcd.OpPut(l.lockKey, "", etcd.WithLease(lease.ID)))
	resp, err := txn.Commit()
	if err != nil {
		return false
	}
	if !resp.Succeeded {
		_, err := cli.Revoke(ctx, lease.ID)
		if err != nil {
			zlog.Errorf("etcd revoke error: %v", err)
		}
		return false
	}
	_, err = cli.KeepAlive(l.cancelCtx, lease.ID)
	if err != nil {
		zlog.Errorf("etcd keepalive error: %v", err)
		return false
	}
	l.locked.Store(true)
	if l.awake != nil {
		close(l.awake)
	}
	return true
}

func (l *globalLock) Wait(timeout time.Duration) error {
	if l.TryLock() {
		return nil
	}
	l.awake = make(chan struct{})
	clusterNode.waitLocks.subscribe(l)
	timer := time.NewTimer(timeout)
	select {
	case <-l.awake:
		return nil
	case <-timer.C:
		return ErrLockTimeout
	}
}

func (l *globalLock) Locked() bool {
	return l.locked.Load()
}

func (l *globalLock) While(retries int, interval ...time.Duration) error {
	if l.locked.Load() {
		panic(ErrLockIsLocked)
	}
	if len(interval) == 0 {
		interval = []time.Duration{128 * time.Millisecond}
	}
	for i := 0; i < retries; i++ {
		if l.TryLock() {
			return nil
		}
		time.Sleep(interval[0])
	}
	return ErrLockTimeout
}

type globalLockedCb[T any] struct {
	f     func(error, ...T)
	cbctx []T
}

func LockAndDo[T any](m idef.IModule, name string, f func(error, ...T), timeout time.Duration, cbctx ...T) {
	m.Async(func() (any, error) {
		l := NewLock(name)
		cb := globalLockedCb[T]{
			f:     f,
			cbctx: cbctx,
		}
		if l.TryLock() {
			return cb, nil
		}
		err := l.Wait(timeout)
		return cb, err
	}, func(r any, err error) {
		cb := r.(globalLockedCb[T])
		cb.f(err, cbctx...)
	})
}

func (l *globalLock) Release() {
	if !l.locked.Load() {
		zlog.Errorf("lock is not locked %s", l.lockKey)
		return
	}
	l.cancelCtx.Cancel()
	clusterNode.etcdClient.Delete(context.Background(), l.lockKey)
}
