package idef

// 普通投递消息(若对方不在线则丢弃)
type CastPackage struct {
	ServerID uint32
	Body     any
}

// 通过流投递消息(持续到消息被消费)
type StreamCastPackage struct {
	ServerID uint32
	Body     any
	Header   map[string]string
}

// 广播给某一类进程
type BroadcastPackage struct {
	ServerType string
	Body       any
}

// 随机投递到某一类进程中的一个上
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
