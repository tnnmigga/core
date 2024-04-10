package basic

import (
	"fmt"
	"reflect"

	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/util"
	"github.com/tnnmigga/nett/zlog"
)

func (m *Module) onRPCRequest(msg *idef.RPCRequest) {
	msgType := reflect.TypeOf(msg.Req)
	h, ok := m.handlers[msgType]
	if !ok {
		msg.Err <- fmt.Errorf("rpc handler not found %v", msgType)
		return
	}
	fn, ok := h.(func(any, func(any), func(error)))
	if !ok {
		zlog.Errorf("%s %s rpc type error", m.name, util.TypeName(msg))
	}
	fn(msg.Req, func(v any) {
		msg.Resp <- v
	}, func(err error) {
		msg.Err <- err
	})
}

func (m *Module) onRPCResponse(req *idef.RPCResponse) {
	req.Cb(req.Resp, req.Err)
}

func (m *Module) onAsyncContext(ctx *asyncContext) {
	ctx.cb(ctx.res, ctx.err)
}
