package mongo

import (
	"context"

	"github.com/tnnmigga/nett/conc"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/msgbus"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (m *module) registerHandler() {
	msgbus.RegisterHandler(m, m.onMongoSaveSingle)
	msgbus.RegisterHandler(m, m.onMongoSaveMulti)
	msgbus.RegisterRPC(m, m.onMongoLoadMulti)
	msgbus.RegisterRPC(m, m.onMongoLoadSingle)
}

func (m *module) onMongoSaveSingle(req *MongoSaveSingle) {
	conc.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
		defer cancel()
		_, err := m.database.Collection(req.CollName).ReplaceOne(ctx, req.Op.Filter, req.Op.Value, options.Replace().SetUpsert(true))
		if err != nil {
			zlog.Errorf("mongo save single error %v", err)
		}
	})
}

func (m *module) onMongoSaveMulti(req *MongoSaveMulti) {
	ms := make([]mongo.WriteModel, 0, len(req.Ops))
	for _, op := range req.Ops {
		m := mongo.NewReplaceOneModel().SetFilter(op.Filter).SetReplacement(op.Value).SetUpsert(true)
		ms = append(ms, m)
	}
	conc.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
		defer cancel()
		_, err := m.database.Collection(req.CollName).BulkWrite(ctx, ms)
		if err != nil {
			zlog.Errorf("mongo save multi error %v", err)
		}
	})
}

func (m *module) onMongoLoadSingle(req *MongoLoadSingle, resolve func(any), reject func(error)) {
	conc.GoWithGroup(req.GroupKey, func() {
		m.semaphore.P()
		defer m.semaphore.V()
		ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
		defer cancel()
		res := m.database.Collection(req.CollName).FindOne(ctx, req.Filter)
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
		ctx, cancel := context.WithTimeout(context.Background(), mongoOpTimeout)
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
