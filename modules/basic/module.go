package basic

import (
	"fmt"
	"reflect"

	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/msgbus"
	"github.com/tnnmigga/nett/util"
)

const (
	DefaultMQLen = 100000
)

type Module struct {
	name      idef.ModName
	mq        chan any
	handlers  map[reflect.Type]any
	hooks     [idef.ServerStateExit + 1][2][]func() error
	closeSign chan struct{}
}

func New(name idef.ModName, mqLen int32) *Module {
	m := &Module{
		name:      name,
		mq:        make(chan any, mqLen),
		handlers:  map[reflect.Type]any{},
		closeSign: make(chan struct{}, 1),
	}
	msgbus.RegisterHandler(m, m.onRPCRequest)
	msgbus.RegisterHandler(m, m.onRPCResponse)
	msgbus.RegisterHandler(m, m.onAsyncContext)
	return m
}

func (m *Module) Name() idef.ModName {
	return m.name
}

func (m *Module) MQ() chan any {
	return m.mq
}

func (m *Module) Assign(msg any) {
	select {
	case m.mq <- msg:
	default:
		zlog.Errorf("modele %s mq full, lose %s", m.name, util.String(msg))
	}
}

func (m *Module) RegisterHandler(mType reflect.Type, handler any) {
	_, ok := m.handlers[mType]
	if ok {
		// 一个module内一个msg只能被注册一次, 但不同模块可以分别注册监听同一个消息
		zlog.Fatal(fmt.Errorf("RegisterHandler multiple registration %v", mType))
	}
	m.handlers[mType] = handler
}

func (m *Module) Hook(state idef.ServerState, stage int) []func() error {
	return m.hooks[state][stage]
}

func (m *Module) Before(state idef.ServerState, hook func() error) {
	if state <= idef.ServerStateInit {
		zlog.Fatal("module after close hook not support")
	}
	m.hooks[state][0] = append(m.hooks[state][0], hook)
}

func (m *Module) After(state idef.ServerState, hook func() error) {
	if state >= idef.ServerStateExit {
		zlog.Fatal("module after close hook not support")
	}
	m.hooks[state][1] = append(m.hooks[state][1], hook)
}

func (m *Module) Run() {
	defer func() {
		zlog.Infof("%v has stoped", m.Name())
		m.closeSign <- struct{}{}
	}()
	for msg := range m.mq {
		m.cb(msg)
	}
}

func (m *Module) Stop() {
	zlog.Infof("try stop %s", m.name)
	close(m.mq)
	<-m.closeSign
}

func (m *Module) cb(msg any) {
	defer util.RecoverPanic()
	msgType := reflect.TypeOf(msg)
	h, ok := m.handlers[msgType]
	if !ok {
		zlog.Errorf("handler not exist %v", msgType)
		return
	}
	fn, ok := h.(func(any))
	if !ok {
		zlog.Errorf("%s %s cb type error", m.name, util.TypeName(msg))
	}
	fn(msg)
}

type asyncContext struct {
	res any
	err error
	cb  func(any, error)
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
