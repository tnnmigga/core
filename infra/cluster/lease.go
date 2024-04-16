package cluster

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/tnnmigga/nett/infra/zlog"
	etcd "go.etcd.io/etcd/client/v3"
)

type Lease struct {
	LeaseID etcd.LeaseID
	TTL     int64
	ticker  *time.Ticker
	m       sync.Mutex
}

func NewLease(TTL int64) (*Lease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	el, err := etcdCli.Grant(ctx, TTL)
	if err != nil {
		return nil, err
	}
	ticker := time.NewTicker(time.Duration(TTL) * time.Second / 2)
	go func() {
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
			_, err := etcdCli.KeepAliveOnce(ctx, el.ID)
			cancel()
			if err == nil {
				continue
			}
			zlog.Warnf("keep alive lease failed: %v", err)
			ticker.Stop()
			ticker = nil
			return
		}
	}()
	lease := &Lease{
		LeaseID: el.ID,
		TTL:     TTL,
		ticker:  ticker,
	}
	runtime.SetFinalizer(lease, func(l *Lease) {
		l.Revoke()
	})
	return lease, nil
}

func (l *Lease) Revoke() {
	l.m.Lock()
	defer l.m.Unlock()
	if l.ticker != nil {
		l.ticker.Stop()
		l.ticker = nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	_, err := etcdCli.Revoke(ctx, l.LeaseID)
	if err != nil {
		zlog.Warnf("revoke lease failed: %v", err)
	}
}

func (l *Lease) KeepAlive() {
	go func() {
		defer zlog.Debugf("lease keep alive routine exit")
		for range l.ticker.C {
			err := l.KeepAliveOnce()
			if err != nil {
				if err != context.Canceled && err != context.DeadlineExceeded {
					time.Sleep(128 * time.Millisecond)
					err = l.KeepAliveOnce()
				}
			}
			if err == nil {
				continue
			}
			zlog.Warnf("keep alive lease failed: %v", err)
			l.m.Lock()
			defer l.m.Unlock()
			l.ticker.Stop()
			l.ticker = nil
			return
		}
	}()
}

func (l *Lease) KeepAliveOnce() error {
	l.m.Lock()
	defer l.m.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	_, err := etcdCli.KeepAliveOnce(ctx, l.LeaseID)
	return err
}