package mongo

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrNoDocuments = mongo.ErrNoDocuments
)

const (
	mongoOpTimeout = 10 * time.Second
)

type MongoSaveOp struct {
	Filter bson.M
	Value  []byte // Value必须是bson序列化好的二进制数据
}

// 保存至MongoDB
// GroupKey为保证并发时的时序
type MongoSaveSingle struct {
	GroupKey string
	CollName string
	Op       *MongoSaveOp
}

// 保存至MongoDB
// GroupKey为保证并发时的时序
type MongoSaveMulti struct {
	GroupKey string
	CollName string
	Ops      []*MongoSaveOp
}

// 从MongoDB加载数据
// GroupKey为保证并发时的时序
// 返回[]bson.Raw
type MongoLoadMulti struct {
	GroupKey string
	CollName string
	Filter   bson.M
}

// 从MongoDB加载数据
// GroupKey为保证并发时的时序
// 返回bson.Raw
type MongoLoadSingle struct {
	GroupKey string
	CollName string
	Filter   bson.M
}
