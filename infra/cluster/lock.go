package cluster

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/utils"
	etcd "go.etcd.io/etcd/client/v3"
)

var (
	ErrLockIsLocked = errors.New("etcd lock locked")
	ErrLockFaild    = errors.New("etcd lock faild")
)

func etcdLockKey(name string) string {
	return fmt.Sprintf("nett/lock/%s", name)
}

type globalLock struct {
	locked    atomic.Bool
	lockKey   string
	cancelCtx utils.IContextWithCancel
	leaseID   etcd.LeaseID
}

func NewLock(name string) *globalLock {
	return &globalLock{
		lockKey:   etcdLockKey(name),
		cancelCtx: utils.ContextWithCancel(context.Background()),
	}
}

func (l *globalLock) tryLock() error {
	if l.locked.Load() {
		return ErrLockIsLocked
	}
	cli := clusterNode.etcdClient
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	lease, err := clusterNode.etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		zlog.Errorf("etcd grant error: %v", err)
		return err
	}
	txn := cli.Txn(ctx).If(etcd.Compare(etcd.Version(l.lockKey), "=", 0)).
		Then(etcd.OpPut(l.lockKey, "", etcd.WithLease(lease.ID)))
	resp, err := txn.Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		_, err := cli.Revoke(ctx, lease.ID)
		if err != nil {
			zlog.Errorf("etcd revoke error: %v", err)
		}
		return err
	}
	_, err = cli.KeepAlive(l.cancelCtx, lease.ID)
	if err != nil {
		zlog.Errorf("etcd keepalive error: %v", err)
		return err
	}
	return nil
}

func (l *globalLock) Wait(timeout ...time.Duration) error {
	return nil
}

func (l *globalLock) Locked() bool {
	return l.locked.Load()
}

func (l *globalLock) While(retries ...int) error {
	return nil
}

func (l *globalLock) LockAndDo(f func(error), timeout ...time.Duration) error {
	return nil
}

func (l *globalLock) Release() {
	l.cancelCtx.Cancel()
	clusterNode.etcdClient.Delete(context.Background(), l.lockKey)
}
