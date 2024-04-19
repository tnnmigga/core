package nett

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/conc"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/cluster"
	"github.com/tnnmigga/nett/infra/process"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/modules/link"
	"github.com/tnnmigga/nett/utils"
)

func init() {
	fname := process.Argv.Str("configs.jsonc", "-c")
	conf.LoadFromJSON(utils.ReadFile(fname))
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
	err := cluster.InitNode()
	if err != nil {
		zlog.Errorf("cluster.InitNode error %v", err)
		os.Exit(1)
	}
	s.after(idef.ServerStateInit, s.abort)
}

func (s *Server) onRun() {
	s.before(idef.ServerStateRun, s.exit)
	zlog.Warn("server try to run")
	for _, m := range s.modules {
		s.runModule(s.wg, m)
	}
	zlog.Warn("server running")
	s.after(idef.ServerStateRun, s.exit)
}

func (s *Server) onStop() {
	s.before(idef.ServerStateStop, s.record)
	zlog.Warn("server try to stop")
	s.waitMsgHandling(time.Minute)
	conc.WaitGoDone(time.Minute)
	for i := len(s.modules) - 1; i >= 0; i-- {
		m := s.modules[i]
		utils.ExecAndRecover(m.Stop)
	}
	s.wg.Wait()
	zlog.Warn("server stoped")
	s.after(idef.ServerStateStop, s.record)
}

func (s *Server) onExit() {
	defer cluster.DeadNode()
	s.before(idef.ServerStateExit, s.record)
	zlog.Warn("server exit")
}

func (s *Server) Shutdown() {
	defer s.onExit()
	defer s.onStop()
}

func (s *Server) runModule(wg *sync.WaitGroup, m idef.IModule) {
	wg.Add(1)
	go func() {
		defer utils.RecoverPanic()
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

// 不走退出流程直接退出进程
func (s *Server) abort(m idef.IModule, err error) {
	zlog.Errorf("module %s, on %s, error: %v", m.Name(), utils.Caller(3), err)
	os.Exit(1)
}

// 正常走退出流程退出
func (s *Server) exit(m idef.IModule, err error) {
	zlog.Fatalf("module %s, on %s, error: %v", m.Name(), utils.Caller(3), err)
}

// 仅记录错误
func (s *Server) record(m idef.IModule, err error) {
	zlog.Errorf("module %s, on %s, error: %v", m.Name(), utils.Caller(3), err)
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
