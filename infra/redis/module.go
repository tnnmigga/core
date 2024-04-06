package redis

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/basic"
	"github.com/tnnmigga/nett/idef"

	"github.com/go-redis/redis/v8"
)

type module struct {
	*basic.Module
	cli *redis.Client
}

func New(name idef.ModName, addr, username, password string) idef.IModule {
	m := &module{
		Module: basic.New(name, basic.DefaultMQLen),
		cli: redis.NewClient(&redis.Options{
			Addr:     addr,
			Username: username,
			Password: password,
		}),
	}
	m.After(idef.ServerStateInit, m.afterInit)
	return m
}

func (m *module) afterInit() error {
	m.initHandler()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := m.cli.Ping(ctx).Result()
	return err
}
