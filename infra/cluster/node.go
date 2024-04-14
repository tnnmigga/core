package cluster

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/infra/sys"
	"github.com/tnnmigga/nett/infra/zlog"
	etcd "go.etcd.io/etcd/client/v3"
)

var (
	ErrNodeIsExists = errors.New("node already exists")
)

const (
	nodeTTL = 10 * time.Second
)

func etcdNodeKey() string {
	return fmt.Sprintf("nett/nodes/%d", conf.ServerID)
}

var (
	etcdCli *etcd.Client
	ticker  *time.Ticker
)

func InitNode() error {
	cli, err := etcd.New(etcd.Config{
		Endpoints:   conf.Array("etcd.endpoints", []string{"http://localhost:2379"}),
		DialTimeout: nodeTTL,
	})
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), nodeTTL)
	defer cancel()
	lease, err := cli.Grant(ctx, 10)
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
	ticker = time.NewTicker(nodeTTL / 2)
	go func() {
		defer zlog.Infof("etcd keep alive goroutine exit")
		for range ticker.C {
			zlog.Debugf("etcd keep alive %d", lease.ID)
			ctx, cancel := context.WithTimeout(context.Background(), nodeTTL/2)
			_, err := etcdCli.KeepAliveOnce(ctx, lease.ID)
			cancel()
			// 若etcd异常则退出
			if err != nil {
				zlog.Errorf("etcd keep alive error: %v", err)
				etcdCli.Close()
				etcdCli = nil
				sys.Abort()
				return
			}
		}
	}()
	etcdCli = cli
	return nil
}

func DeadNode() {
	ticker.Stop()
	if etcdCli == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), nodeTTL)
	defer cancel()
	_, err := etcdCli.Delete(ctx, etcdNodeKey())
	if err != nil {
		zlog.Errorf("etcd delete node error: %v", err)
	}
	etcdCli.Close()
}
