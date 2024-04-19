package redis

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/mods/basic"

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
	m.After(idef.ServerStateStop, m.afterStop)
	return m
}

func (m *module) afterInit() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := m.cli.Ping(ctx).Result()
	if err != nil {
		return err
	}
	m.initHandler()
	return nil
}

func (m *module) afterStop() error {
	return m.cli.Close()
}
