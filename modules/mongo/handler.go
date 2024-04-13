package mongo

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/msgbus"
	"github.com/tnnmigga/nett/zlog"

	"go.mongodb.org/mongo-driver/mongo"
)

func (m *module) registerHandler() {
	msgbus.RegisterHandler(m, m.onMongoSave)
	msgbus.RegisterRPC(m, m.onMongoLoad)
}

func (m *module) onMongoSave(req *MongoSave) {
	ms := make([]mongo.WriteModel, 0, len(req.Ops))
	for _, op := range req.Ops {
		m := mongo.NewReplaceOneModel().SetFilter(op.Filter).SetReplacement(op.Value).SetUpsert(true)
		ms = append(ms, m)
	}
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		_, err := m.mongoCli.Database(req.DBName).Collection(req.CollName).BulkWrite(ctx, ms)
		cancel()
		if err != nil {
			zlog.Errorf("mongo save error %v", err)
		}
	})
}

func (m *module) onMongoLoad(req *MongoLoad, resolve func(any), reject func(error)) {
	core.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		cur, _ := m.mongoCli.Database(req.DBName).Collection(req.CollName).Find(context.Background(), req.Filter)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		err := cur.All(ctx, &req.Data)
		cancel()
		if err != nil {
			reject(err)
		} else {
			resolve(req.Data)
		}
	})
}