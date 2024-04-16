package cluster

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	etcd "go.etcd.io/etcd/client/v3"
)

func etcdLockKey(name string) string {
	return fmt.Sprintf("nett/lock/%s", name)
}

type globalLock struct {
	lockKey string
	flag    atomic.Bool
	rw      sync.RWMutex
	cancel  func()
	leaseID etcd.LeaseID
}

func NewLock(name string) *globalLock {
	return &globalLock{
		lockKey: etcdLockKey(name),
	}
}

func (l *globalLock) Try() bool {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	if l.leaseID == 0 {
		lease, err := etcdCli.Grant(ctx, leaseTTL)
		
	}
	etcdCli.Revoke()
	txn := etcdCli.Txn(ctx).If(etcd.Compare(etcd.Version(etcdNodeKey()), "=", 0)).Then(etcd.OpPut(l.lockKey, "", etcd.WithLease(l.leaseID)))
	l.rw.Lock()
}

func (l *globalLock) Wait(timeout ...time.Duration) bool {
	l.rw.Lock()
}

func (l *globalLock) While(retries ...int) bool {
	l.rw.Unlock()
}

func (l *globalLock) LockAndDo(f func(error), timeout ...time.Duration) bool {
	if l.cancel != nil {
		l.cancel()
	}
}

func (l *globalLock) Release() {
	l.rw.Unlock()
}
