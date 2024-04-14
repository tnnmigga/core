package nett

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/sys/argv"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/modules/link"
	"github.com/tnnmigga/nett/util"
)

func init() {
	fname := argv.Str("configs.jsonc", "-c", "--config")
	conf.LoadFromJSON(util.ReadFile(fname))
	zlog.Init()
}

type Server struct {
	modules []idef.IModule
	wg      *sync.WaitGroup
}

func NewServer(modules ...idef.IModule) *Server {
	server := &Server{
		modules: make([]idef.IModule, 0, len(modules)+1),
		wg:      &sync.WaitGroup{},
	}
	server.modules = append(server.modules, link.New()) // nats最后停止
	server.modules = append(server.modules, modules...)
	server.onInit()
	server.onRun()
	return server
}

func (s *Server) onInit() {
	zlog.Warnf("server initialization")
	s.after(idef.ServerStateInit, s.abort)
}

func (s *Server) onRun() {
	s.before(idef.ServerStateRun, s.abort)
	zlog.Warn("server try to run")
	for _, m := range s.modules {
		s.runModule(s.wg, m)
	}
	zlog.Warn("server running")
	s.after(idef.ServerStateRun, s.abort)
}

func (s *Server) onStop() {
	s.before(idef.ServerStateStop, s.noabort)
	zlog.Warn("server try to stop")
	s.waitMsgHandling(time.Minute)
	core.WaitGoDone(time.Minute)
	for i := len(s.modules) - 1; i >= 0; i-- {
		m := s.modules[i]
		util.ExecAndRecover(m.Stop)
	}
	s.wg.Wait()
	zlog.Warn("server stoped")
	s.after(idef.ServerStateStop, s.noabort)
}

func (s *Server) onExit() {
	s.before(idef.ServerStateExit, s.noabort)
	zlog.Warn("server exit")
	os.Exit(0)
}

func (s *Server) Shutdown() {
	defer s.onExit()
	defer s.onStop()
}

func (s *Server) runModule(wg *sync.WaitGroup, m idef.IModule) {
	wg.Add(1)
	go func() {
		defer util.RecoverPanic()
		defer wg.Done()
		m.Run()
	}()
}

func (s *Server) waitMsgHandling(maxWaitTime time.Duration) {
	// 每100ms检查一次模块消息是否处理完
	maxCheckCount := maxWaitTime / time.Millisecond / 100
	for ; maxCheckCount > 0; maxCheckCount-- {
		time.Sleep(100 * time.Millisecond)
		isEmpty := true
		for _, m := range s.modules {
			if len(m.MQ()) != 0 {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			return
		}
	}
	zlog.Errorf("wait msg handing timeout")
}

func (s *Server) abort(m idef.IModule, err error) {
	zlog.Fatalf("module %s, on %s, error: %v", m.Name(), util.Caller(3), err)
}

func (s *Server) noabort(m idef.IModule, err error) {
	zlog.Errorf("module %s, on %s, error: %v", m.Name(), util.Caller(3), err)
}

func (s *Server) before(state idef.ServerState, onError ...func(idef.IModule, error)) {
	for _, m := range s.modules {
		hook := m.Hook(state, 0)
		for _, h := range hook {
			if err := wrapHook(h)(); err != nil {
				zlog.Errorf("server before %#v error, module %s, error %v", state, m.Name(), err)
				for _, f := range onError {
					f(m, err)
				}
			}
		}
	}
}

func (s *Server) after(state idef.ServerState, onError ...func(idef.IModule, error)) {
	for _, m := range s.modules {
		hook := m.Hook(state, 1)
		for _, h := range hook {
			if err := wrapHook(h)(); err != nil {
				zlog.Errorf("server before %#v error, module %s, error %v", state, m.Name(), err)
				for _, f := range onError {
					f(m, err)
				}
			}
		}
	}
}

// 添加panic处理
func wrapHook(h func() error) func() error {
	return func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v: %s", r, debug.Stack())
			}
		}()
		return h()
	}
}
