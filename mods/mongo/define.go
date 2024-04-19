package mongo

import (
	"go.mongodb.org/mongo-driver/bson"
)

type MongoSaveOp struct {
	Filter bson.M
	Value  []byte // Value必须是bson序列化好的二进制数据
}

// 保存至MongoDB
// GroupKey为保证并发时的时序
type MongoSave struct {
	GroupKey string
	DBName   string
	CollName string
	Ops      []*MongoSaveOp
}

// 从MongoDB加载数据
// GroupKey为保证并发时的时序
type MongoLoad struct {
	GroupKey string
	DBName   string
	CollName string
	Filter   bson.M
	Data     any
}
