package mongo

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/conc"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/mods/basic"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const MaxConcurrency = 0xFF

type module struct {
	*basic.Module
	semaphore conc.Semaphore // 控制并发数
	mongoCli  *mongo.Client  // mongo
	mongoURI  string
}

func New(name idef.ModName, uri string) idef.IModule {
	m := &module{
		Module:    basic.New(name, basic.DefaultMQLen),
		semaphore: conc.NewSemaphore(MaxConcurrency),
		mongoURI:  uri,
	}
	m.registerHandler()
	m.Before(idef.ServerStateRun, m.beforeRun)
	m.After(idef.ServerStateStop, m.afterStop)
	return m
}

func (m *module) beforeRun() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m.mongoCli, err = mongo.Connect(ctx, options.Client().ApplyURI(m.mongoURI))
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
