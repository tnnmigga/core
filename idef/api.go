package idef

import (
	"context"
	"reflect"
)

type IModule interface {
	Name() ModName
	// 将一个消息指派给这个模块处理
	Assign(any)
	// 消息缓冲chan
	MQ() chan any
	// 开始消息处理
	Run()
	// 在处理完当前消息之后停止消息处理
	Stop()
	// 注册消息处理函数
	RegisterHandler(mType reflect.Type, handler any)
	// 注册钩子函数(切换到state之前调用)
	Before(state ServerState, hook func() error)
	// 注册钩子函数(切换到state之后调用)
	After(state ServerState, hook func() error)
	// 返还钩子函数列表
	Hook(state ServerState, stage int) []func() error
	// 异步回调的方式执行函数
	// 启动一个新的goruntine执行同步阻塞的代码
	// 执行完将结果返到模块线程往后执行
	// 匿名函数捕获的变量需要防范并发读写问题
	Async(f func() (any, error), cb func(any, error))
}

type ContextWithCancel interface {
	context.Context
	Cancel()
}