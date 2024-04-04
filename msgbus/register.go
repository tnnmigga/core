package msgbus

import (
	"fmt"
	"reflect"

	"github.com/tnnmigga/nett/codec"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/zlog"
)

func RegisterHandler[T any](m idef.IModule, fn func(T)) {
	var tmp T
	mType := reflect.TypeOf(tmp)
	codec.Register[T]()
	registerRecver(mType, m)
	m.RegisterHandler(mType, func(data any) {
		msg := data.(T)
		fn(msg)
	})
}

func RegisterRPC[T any](m idef.IModule, fn func(msg T, resolve func(any), reject func(error))) {
	var tmp T
	mType := reflect.TypeOf(tmp)
	codec.Register[T]()
	registerRecver(mType, m)
	m.RegisterHandler(mType, func(data any, res func(any), rej func(error)) {
		msg := data.(T)
		fn(msg, res, rej)
	})
}

// 注册消息接收者
func registerRecver(mType reflect.Type, recver IRecver) {
	if ms, has := recvers[mType]; has {
		for _, m := range ms {
			if m.Name() == recver.Name() {
				zlog.Fatal(fmt.Errorf("message duplicate registration %v %v", recver.Name(), mType.Elem().Name()))
			}
		}
	}
	recvers[mType] = append(recvers[mType], recver)
}
