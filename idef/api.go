package idef

import (
	"reflect"
)

type IModule interface {
	Name() ModName
	Assign(any)
	MQ() chan any
	Run()
	Stop()
	RegisterHandler(mType reflect.Type, handler any)
	Before(state ServerState, hook func() error)
	After(state ServerState, hook func() error)
	Hook(state ServerState, stage int) []func() error
	// 异步回调的方式执行函数
	// 启动一个新的goruntine执行同步阻塞的代码
	// 执行完将结果返到模块线程往后执行
	// 匿名函数捕获的变量需要防范并发读写问题
	Async(f func() (any, error), cb func(any, error))
}
