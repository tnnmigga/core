package mongo

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/basic"
	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type module struct {
	*basic.Module
	semaphore *core.Semaphore // 控制并发数
	mongoCli  *mongo.Client   // mongo
}

func New(name string) idef.IModule {
	m := &module{
		Module:    basic.New(name, basic.DefaultMQLen),
		semaphore: core.NewSemaphore(conf.Int("mongo.max_concurrency", 0xFF)),
	}
	m.registerHandler()
	m.Before(idef.ServerStateRun, m.beforeRun)
	m.After(idef.ServerStateStop, m.afterStop)
	return m
}

func (m *module) beforeRun() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m.mongoCli, err = mongo.Connect(ctx, options.Client().ApplyURI(conf.String("mongo.url", "mongodb://localhost")))
	if err != nil {
		return err
	}
	if err := m.mongoCli.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}
	return nil
}

func (m *module) afterStop() (err error) {
	m.mongoCli.Disconnect(context.Background())
	return nil
}
