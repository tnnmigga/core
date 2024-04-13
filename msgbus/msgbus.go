package msgbus

import (
	"errors"
	"fmt"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/util"

	"reflect"
)

var (
	ErrRPCTimeout = errors.New("rpc timeout")
)

func init() {
	conf.RegInitFn(func() {
		// rpcMaxWaitTime = time.Duration(conf.Int64("rpc-wait-time", 10)) * time.Second
	})
}

var (
	recvers map[reflect.Type][]IRecver
	// rpcMaxWaitTime time.Duration
)

func init() {
	recvers = map[reflect.Type][]IRecver{}
}

type IRecver interface {
	Name() idef.ModName
	Assign(any)
}

// 跨进程投递消息
func Cast(msg any, opts ...castOpt) {
	// 跨协程传递消息默认深拷贝防止并发修改
	msg = deepcopy.Copy(msg)
	// 如果不指定serverID则默认投递到本地
	serverID := findCastOpt[uint32](opts, idef.ConstKeyServerID, conf.ServerID)
	if serverID == conf.ServerID {
		castLocal(msg, opts...)
		return
	}
	// 检查是否不使用stream
	if nonuse := findCastOpt[bool](opts, idef.ConstKeyNonuseStream, false); nonuse {
		// 不使用stream
		castLocal(&idef.CastPackage{
			ServerID: serverID,
			Body:     msg,
		}, opts...)
		return
	}
	// 默认使用stream
	castLocal(&idef.StreamCastPackage{
		ServerID: serverID,
		Body:     msg,
		Header:   castHeader(opts),
	}, opts...)
}

// 投递到本地其他协程
// 跨进程投递靠本地link模块转发
func castLocal(msg any, opts ...castOpt) {
	recvs, ok := recvers[reflect.TypeOf(msg)]
	if !ok {
		zlog.Errorf("message cast recv not fuound %v", util.TypeName(msg))
		return
	}
	modName := findCastOpt[idef.ModName](opts, idef.ConstKeyOneOfMods, "")
	for _, recv := range recvs {
		if modName != "" && modName != recv.Name() {
			continue
		}
		recv.Assign(msg)
	}
}

// 广播到一个serverType类别下的所有进程
func Broadcast(serverType string, msg any) {
	pkg := &idef.BroadcastPackage{
		ServerType: serverType,
		Body:       deepcopy.Copy(msg),
	}
	castLocal(pkg)
}

// 随机等概率投递到一个serverType类别下的某个进程
func Randomcast(serverType string, msg any) {
	pkg := &idef.RandomCastPackage{
		ServerType: serverType,
		Body:       deepcopy.Copy(msg),
	}
	castLocal(pkg)
}

// RPC 跨协程/进程调用
// caller: 为调用者模块 也是回调函数的执行者
// target: 目标参数 可以通过msgbus.ServerID()指定某个特定的进程或通过msgbus.ServerType()在某类进程中随机一个
// 调用本地使用msgbus.Local()或msgbus.ServerID(conf.ServerID)
// req: 请求参数
// cb: 回调函数 由调用方模块线程执行
func RPC[T any](caller idef.IModule, target castOpt, req any, cb func(resp T, err error)) {
	// 跨协程传递消息默认深拷贝防止并发修改
	req = deepcopy.Copy(req)
	if target.key == idef.ConstKeyServerID && target.value.(uint32) == conf.ServerID {
		localCall(caller, req, warpCb(cb))
		return
	}
	rpcCtx := &idef.RPCContext{
		Caller: caller,
		Req:    req,
		Resp:   util.New[T](),
		Cb:     warpCb(cb),
	}
	if target.key == idef.ConstKeyServerID {
		rpcCtx.ServerID = target.value.(uint32)
	} else if target.key == idef.ConstKeyServerType {
		rpcCtx.ServerType = target.value.(string)
	} else {
		zlog.Errorf("rpc target type error %v", target.value)
		return
	}
	castLocal(rpcCtx)
}

func localCall(m idef.IModule, req any, cb func(resp any, err error)) {
	recvs, ok := recvers[reflect.TypeOf(req)]
	if !ok {
		zlog.Errorf("recvs not fuound %v", util.TypeName(req))
		return
	}
	core.Go(func() {
		callReq := &idef.RPCRequest{
			Req:  req,
			Resp: make(chan any, 1),
			Err:  make(chan error, 1),
		}
		callResp := &idef.RPCResponse{
			Module: m,
			Req:    req,
			Cb:     cb,
		}
		recvs[0].Assign(callReq)
		timer := time.NewTimer(conf.MaxRPCWaitTime)
		defer timer.Stop()
		select {
		case <-timer.C:
			callResp.Err = ErrRPCTimeout
		case callResp.Resp = <-callReq.Resp:
		case callResp.Err = <-callReq.Err:
		}
		m.Assign(callResp)
	})
}

func warpCb[T any](cb func(T, error)) func(any, error) {
	return func(pkg any, err error) {
		if err != nil {
			var empty T
			cb(empty, err)
			return
		}
		resp, ok := pkg.(T)
		if !ok {
			zlog.Errorf("rpc resp type error, %#v %#v", *new(T), pkg)
		}
		cb(resp, err)
	}
}

func castHeader(opts []castOpt) map[string]string {
	handler := map[string]string{}
	for _, opt := range opts {
		switch opt.key {
		case idef.ConstKeyExpires:
			handler[opt.key] = fmt.Sprint(opt.value)
		}
	}
	return handler
}
