package mongo

import (
	"context"
	"time"

	"github.com/tnnmigga/nett/conc"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/msgbus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (m *module) registerHandler() {
	msgbus.RegisterHandler(m, m.onMongoSave)
	msgbus.RegisterRPC(m, m.onMongoLoadMulti)
	msgbus.RegisterRPC(m, m.onMongoLoadSingle)
}

func (m *module) onMongoSave(req *MongoSave) {
	ms := make([]mongo.WriteModel, 0, len(req.Ops))
	for _, op := range req.Ops {
		m := mongo.NewReplaceOneModel().SetFilter(op.Filter).SetReplacement(op.Value).SetUpsert(true)
		ms = append(ms, m)
	}
	conc.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		_, err := m.database.Collection(req.CollName).BulkWrite(ctx, ms)
		cancel()
		if err != nil {
			zlog.Errorf("mongo save error %v", err)
		}
	})
}

func (m *module) onMongoLoadSingle(req *MongoLoadSingle, resolve func(any), reject func(error)) {
	conc.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		res := m.database.Collection(req.CollName).FindOne(ctx, req.Filter)
		cancel()
		raw, err := res.Raw()
		if res.Err() != nil {
			reject(err)
		} else {
			resolve(raw)
		}
	})
}

func (m *module) onMongoLoadMulti(req *MongoLoadMulti, resolve func(any), reject func(error)) {
	conc.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		cur, _ := m.database.Collection(req.CollName).Find(ctx, req.Filter)
		raws := []bson.Raw{}
		for cur.Next(ctx) {
			raws = append(raws, cur.Current)
		}
		err := cur.Err()
		if err != nil {
			reject(err)
		} else {
			resolve(raws)
		}
	})
}
