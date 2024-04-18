package cluster

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/infra/sys"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/utils"

	etcd "go.etcd.io/etcd/client/v3"
)

var (
	ErrNodeIsExists = errors.New("node already exists")
)

const (
	leaseTTL   = 10
	opTimeout  = 10 * time.Second
	nodePrefix = "/cluster/nodes"
)

var clusterNode *Node

type Node struct {
	etcdClient *etcd.Client
	leaseID    etcd.LeaseID
	cancelCtx  utils.IContextWithCancel
	waitLocks  waitLockManager
}

func (n *Node) KeepAlive() {
	go func() {
		defer func() {
			utils.RecoverPanic()
			if r := recover(); r != nil {
				zlog.Errorf("%v: %s", r, debug.Stack())
				sys.Abort()
			}
		}()
		ticker := time.NewTicker(leaseTTL * time.Second / 2)
		defer ticker.Stop()
		for {
			select {
			case <-n.cancelCtx.Done():
				return
			case <-ticker.C:
				zlog.Debugf("etcd keep alive %d", n.leaseID)
				ctx, cancel := context.WithTimeout(n.cancelCtx, opTimeout/2)
				_, err := n.etcdClient.KeepAliveOnce(ctx, n.leaseID)
				cancel()
				// 若etcd异常则退出
				if err != nil && !n.cancelCtx.Canceled() {
					zlog.Errorf("etcd keep alive error: %v", err)
					sys.Abort()
					return
				}
			}
		}
	}()
}

func etcdNodeKey() string {
	return fmt.Sprintf("%s/%d", nodePrefix, conf.ServerID)
}

func InitNode() error {
	cli, err := etcd.New(etcd.Config{
		Endpoints:   conf.Array("etcd.endpoints", []string{"http://localhost:2379"}),
		DialTimeout: opTimeout,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	lease, err := cli.Grant(ctx, leaseTTL)
	if err != nil {
		return err
	}
	putTxn := cli.Txn(ctx)
	putTxn.If(etcd.Compare(etcd.Version(etcdNodeKey()), "=", 0)).
		Then(etcd.OpPut(etcdNodeKey(), "", etcd.WithLease(lease.ID)))
	putRes, err := putTxn.Commit()
	if err != nil {
		return nil
	}
	if !putRes.Succeeded {
		return ErrNodeIsExists
	}
	clusterNode = &Node{
		cancelCtx:  utils.ContextWithCancel(context.Background()),
		leaseID:    lease.ID,
		etcdClient: cli,
	}
	clusterNode.KeepAlive()
	return nil
}

func DeadNode() {
	clusterNode.cancelCtx.Cancel()
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	_, err := clusterNode.etcdClient.Delete(ctx, etcdNodeKey())
	if err != nil {
		zlog.Errorf("etcd delete node error: %v", err)
	}
	clusterNode.etcdClient.Close()
}
