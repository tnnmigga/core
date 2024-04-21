package cluster

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/infra/process"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/utils"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	ErrNodeIsExists = errors.New("node already exists")
)

var etcd *clientv3.Client

func init() {

}

const (
	leaseTTL   = 10
	opTimeout  = 10 * time.Second
	nodePrefix = "/cluster/nodes"
)

var clusterNode *Node

type Node struct {
	waitLocks *waitLockManager
	leaseID   clientv3.LeaseID
	cancelCtx utils.IContextWithCancel
}

func Init() error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Array("etcd.endpoints", []string{"http://localhost:2379"}),
		DialTimeout: opTimeout,
	})
	if err != nil {
		panic(err)
	}
	etcd = cli
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	lease, err := etcd.Grant(ctx, leaseTTL)
	if err != nil {
		return err
	}
	putTxn := etcd.Txn(ctx)
	putTxn.If(clientv3.Compare(clientv3.Version(etcdNodeKey()), "=", 0)).
		Then(clientv3.OpPut(etcdNodeKey(), "", clientv3.WithLease(lease.ID)))
	putRes, err := putTxn.Commit()
	if err != nil {
		return err
	}
	if !putRes.Succeeded {
		return ErrNodeIsExists
	}
	clusterNode = &Node{
		cancelCtx: utils.ContextWithCancel(context.Background()),
		leaseID:   lease.ID,
		waitLocks: newWaitLockManager(ctx),
	}
	clusterNode.KeepAlive()
	return nil
}

func etcdNodeKey() string {
	return fmt.Sprintf("%s/%d", nodePrefix, conf.ServerID)
}

func (n *Node) KeepAlive() {
	go func() {
		defer func() {
			utils.RecoverPanic()
			if r := recover(); r != nil {
				zlog.Errorf("%v: %s", r, debug.Stack())
				process.Exit()
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
				_, err := etcd.KeepAliveOnce(ctx, n.leaseID)
				cancel()
				// 若etcd异常则退出
				if err != nil && !n.cancelCtx.Canceled() {
					zlog.Errorf("etcd keep alive error: %v", err)
					process.Exit()
					return
				}
			}
		}
	}()
}

func Dead() {
	clusterNode.cancelCtx.Cancel()
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()
	_, err := etcd.Delete(ctx, etcdNodeKey())
	if err != nil {
		zlog.Errorf("etcd delete node error: %v", err)
	}
	etcd.Close()
}
