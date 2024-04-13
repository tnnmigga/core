package basic

import (
	"fmt"
	"reflect"

	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/util"
)

func (m *Module) onRPCRequest(req *idef.RPCRequest) {
	msgType := reflect.TypeOf(req.Req)
	h, ok := m.handlers[msgType]
	if !ok {
		req.Err <- fmt.Errorf("rpc handler not found %v", msgType)
		return
	}
	fn, ok := h.(func(any, func(any), func(error)))
	if !ok {
		zlog.Errorf("%s %s rpc type error", m.name, util.TypeName(req))
	}
	fn(req.Req, func(v any) {
		req.Resp <- v
	}, func(err error) {
		req.Err <- err
	})
}

func (m *Module) onRPCResponse(req *idef.RPCResponse) {
	req.Cb(req.Resp, req.Err)
}

func (m *Module) onAsyncContext(req *asyncContext) {
	req.cb(req.res, req.err)
}
