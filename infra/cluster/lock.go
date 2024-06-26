package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tnnmigga/core/idef"
	"github.com/tnnmigga/core/infra/zlog"
	"github.com/tnnmigga/core/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	ErrLockIsLocked  = errors.New("etcd lock locked")
	ErrLockEtcdError = errors.New("etcd lock etcd error")
	ErrLockTimeout   = errors.New("etcd lock timeout")
)

const (
	lockPrefix = "/cluster/locks"
)

type waitLockManager struct {
	waitQueue map[string][]*Lock
	mtx       sync.Mutex
	watcher   clientv3.WatchChan
	ctx       context.Context
}

func newWaitLockManager(ctx context.Context) *waitLockManager {
	watcher := etcd.Watch(ctx, fmt.Sprintf("%s/", lockPrefix), clientv3.WithPrefix())
	manager := &waitLockManager{
		waitQueue: make(map[string][]*Lock),
		watcher:   watcher,
		ctx:       ctx,
	}
	go manager.watch()
	return &waitLockManager{
		waitQueue: make(map[string][]*Lock),
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
				if ev.Type != clientv3.EventTypeDelete {
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

func (m *waitLockManager) subscribe(gl *Lock) {
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

type Lock struct {
	lease   *Lease
	locked  atomic.Bool
	lockKey string
	awake   chan struct{}
}

func NewLock(name string) *Lock {
	return &Lock{
		lockKey: etcdLockKey(name),
	}
}

func (l *Lock) TryLock() error {
	if l.locked.Load() {
		return ErrLockIsLocked
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	if l.lease == nil {
		lease, err := NewLease()
		if err != nil {
			zlog.Errorf("etcd error %v", err)
			return ErrLockEtcdError
		}
		l.lease = lease
	}
	txn := etcd.Txn(ctx).If(clientv3.Compare(clientv3.Version(l.lockKey), "=", 0)).
		Then(clientv3.OpPut(l.lockKey, "", clientv3.WithLease(l.lease.leaseID)))
	resp, err := txn.Commit()
	if err != nil {
		l.lease.Revoke()
		zlog.Errorf("etcd txn commit error %v", err)
		return ErrLockEtcdError
	}
	if !resp.Succeeded {
		l.lease.Revoke()
		return ErrLockIsLocked
	}
	l.locked.Store(true)
	if l.awake != nil {
		close(l.awake)
	}
	return nil
}

func (l *Lock) Wait(timeout time.Duration) error {
	if l.TryLock() == nil {
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

func (l *Lock) Locked() bool {
	return l.locked.Load()
}

func (l *Lock) While(retries int, interval ...time.Duration) error {
	if l.locked.Load() {
		panic(ErrLockIsLocked)
	}
	if len(interval) == 0 {
		interval = []time.Duration{128 * time.Millisecond}
	}
	for i := 0; i < retries; i++ {
		if l.TryLock() == nil {
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
		if l.TryLock() == nil {
			return cb, nil
		}
		err := l.Wait(timeout)
		return cb, err
	}, func(r any, err error) {
		cb := r.(globalLockedCb[T])
		cb.f(err, cbctx...)
	})
}

func (l *Lock) Release() {
	if !l.locked.Load() {
		zlog.Errorf("lock is not locked %s", l.lockKey)
		return
	}
	if l.lease != nil {
		l.lease.Revoke()
	}
	etcd.Delete(context.Background(), l.lockKey)
}
