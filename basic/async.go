package basic

import (
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/util"
)

type asyncContext struct {
	res any
	err error
	cb  func(any, error)
}

// 异步回调的方式执行函数
// 启动一个新的goruntine执行同步阻塞的代码
// 执行完将结果返到模块线程往后执行
// 匿名函数捕获的变量需要防范并发读写问题
func Async[T any](m idef.IModule, f func() (T, error), cb func(T, error)) {
	m.Async(func() (any, error) {
		res, err := f()
		return res, err
	}, func(a any, err error) {
		cb(a.(T), err)
	})
}

// 异步回调的方式执行函数
// 启动一个新的goruntine执行同步阻塞的代码
// 执行完将结果返到模块线程往后执行
// 匿名函数捕获的变量需要防范并发读写问题
func (m *Module) Async(f func() (any, error), cb func(any, error)) {
	core.Go(func() {
		defer util.RecoverPanic()
		res, err := f()
		m.Assign(&asyncContext{
			res: res,
			err: err,
			cb:  cb,
		})
	})
}
