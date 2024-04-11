package idef

type CastPackage struct {
	ServerID uint32
	Body     any
}

type StreamCastPackage struct {
	ServerID uint32
	Body     any
	Header   map[string]string
}

type BroadcastPackage struct {
	ServerType string
	Body       any
}

type RandomCastPackage struct {
	ServerType string
	Body       any
}

// 发起RPC请求
type RPCRequest struct {
	Req  any
	Resp chan any
	Err  chan error
}

// RPC请求完成
type RPCResponse struct {
	Module IModule
	Req    any
	Resp   any
	Err    error
	Cb     func(resp any, err error)
}

// RPC上下文 跨进程调用时用到
type RPCContext struct {
	Caller     IModule
	ServerType string
	ServerID   uint32
	Req        any
	Resp       any
	Cb         func(resp any, err error)
}
