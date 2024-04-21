package cluster

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Lease struct {
	ctx     utils.IContextWithCancel
	leaseID clientv3.LeaseID
}

func NewLease() (*Lease, error) {
	ctx := utils.ContextWithCancel(context.Background())
	resp, err := etcd.Grant(ctx, leaseTTL)
	if err != nil {
		return nil, err
	}
	lease := &Lease{
		ctx:     ctx,
		leaseID: resp.ID,
	}
	lease.keepAlive()
	return lease, nil
}

func (l *Lease) Revoke() {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	_, err := etcd.Revoke(ctx, l.leaseID)
	if err != nil {
		zlog.Errorf("etcd lease revoke error %v", err)
	}
	l.ctx.Cancel()
}

func (n *Lease) keepAlive() {
	go func() {
		defer func() {
			utils.RecoverPanic()
			if r := recover(); r != nil {
				zlog.Errorf("%v: %s", r, debug.Stack())
			}
		}()
		ticker := time.NewTicker(leaseTTL * time.Second / 2)
		defer ticker.Stop()
		for {
			select {
			case <-n.ctx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(n.ctx, opTimeout/2)
				_, err := etcd.KeepAliveOnce(ctx, n.leaseID)
				cancel()
				if err != nil && !n.ctx.Canceled() {
					zlog.Errorf("etcd keep alive error: %v", err)
					return
				}
			}
		}
	}()
}
