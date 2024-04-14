package cluster

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/infra/zlog"
	etcd "go.etcd.io/etcd/client/v3"
)

var (
	ErrNodeIsExists = errors.New("node already exists")
)

const (
	nodeTTL = 5 * time.Second
)

func etcdNodeKey() string {
	return fmt.Sprintf("nett/nodes/%d", conf.ServerID)
}

var (
	etcdCli *etcd.Client
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
	keepAlive, err := cli.KeepAlive(context.Background(), lease.ID)
	if err != nil {
		return err
	}
	go func() {
		for resp := range keepAlive {
			zlog.Debugf("etcd keep alive success %v", resp.ID)
		}
	}()
	etcdCli = cli
	return nil
}

func DeadNode() {
	etcdCli.Delete(context.Background(), etcdNodeKey())
	etcdCli.Close()
}
