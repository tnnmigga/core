package msgbus

import (
	"errors"
	"fmt"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/core"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/util"
	"github.com/tnnmigga/nett/zlog"

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

func Broadcast(serverType string, msg any) {
	pkg := &idef.BroadcastPackage{
		ServerType: serverType,
		Body:       deepcopy.Copy(msg),
	}
	castLocal(pkg)
}

func RPC[T any](m idef.IModule, serverID uint32, req any, cb func(resp T, err error)) {
	if serverID == conf.ServerID {
		localCall(m, req, warpCb(cb))
		return
	}
	rpcCtx := &idef.RPCContext{
		Caller:   m,
		ServerID: serverID,
		Req:      req,
		Resp:     util.New[T](),
		Cb:       warpCb(cb),
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
			zlog.Panicf("rpc resp type error, need %s, cur %s", util.TypeName(new(T)), util.TypeName(pkg))
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
