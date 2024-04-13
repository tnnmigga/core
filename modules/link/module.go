package link

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/tnnmigga/nett/codec"
	"github.com/tnnmigga/nett/conf"
	"github.com/tnnmigga/nett/idef"
	"github.com/tnnmigga/nett/infra/zlog"
	"github.com/tnnmigga/nett/modules/basic"
	"github.com/tnnmigga/nett/msgbus"
	"github.com/tnnmigga/nett/util"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	castStreamName = "stream-cast"
)

type module struct {
	*basic.Module
	conn    *nats.Conn
	js      jetstream.JetStream
	stream  jetstream.Stream
	cons    jetstream.Consumer
	consCtx jetstream.ConsumeContext
	subs    [5]*nats.Subscription
}

func New() idef.IModule {
	m := &module{
		Module: basic.New(idef.ModLink, conf.Int32("nats.mq-len", basic.DefaultMQLen)),
	}
	codec.Register[*RPCResult]()
	m.initHandler()
	m.After(idef.ServerStateInit, m.afterInit)
	m.After(idef.ServerStateRun, m.afterRun)
	m.Before(idef.ServerStateStop, m.beforeStop)
	m.After(idef.ServerStateStop, m.afterStop)
	return m
}

func (m *module) afterInit() error {
	conn, err := nats.Connect(
		conf.String("nats.url", nats.DefaultURL),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			zlog.Errorf("nats retry connect")
		}),
	)
	if err != nil {
		return err
	}
	m.conn = conn
	m.js, err = jetstream.New(m.conn)
	if err != nil {
		return err
	}
	m.stream, err = m.js.Stream(context.Background(), castStreamName)
	if err != nil {
		return err
	}
	return nil
}

func (m *module) afterRun() (err error) {
	m.cons, err = m.stream.CreateOrUpdateConsumer(context.Background(), jetstream.ConsumerConfig{
		Durable:       fmt.Sprintf("%s-%d", conf.ServerType, conf.ServerID),
		FilterSubject: streamCastSubject(conf.ServerID),
	})
	if err != nil {
		return err
	}
	m.consCtx, err = m.cons.Consume(m.streamMsgHandler)
	if err != nil {
		return err
	}
	m.subs[0], err = m.conn.Subscribe(castSubject(conf.ServerID), m.msgHandler)
	if err != nil {
		return err
	}
	m.subs[1], err = m.conn.Subscribe(broadcastSubject(conf.ServerType), m.msgHandler)
	if err != nil {
		return err
	}
	m.subs[2], err = m.conn.QueueSubscribe(randomCastSubject(conf.ServerType), conf.ServerType, m.msgHandler)
	if err != nil {
		return err
	}
	m.subs[3], err = m.conn.Subscribe(rpcSubject(conf.ServerID), m.rpcHandler)
	if err != nil {
		return err
	}
	m.subs[4], err = m.conn.QueueSubscribe(randomRpcSubject(conf.ServerType), conf.ServerType, m.rpcHandler)
	if err != nil {
		return err
	}
	return nil
}

func castSubject(serverID uint32) string {
	return fmt.Sprintf("cast.%d", serverID)
}

func streamCastSubject(serverID uint32) string {
	return fmt.Sprintf("stream.cast.%d", serverID)
}

func broadcastSubject(serverType string) string {
	return fmt.Sprintf("broadcast.%s", serverType)
}

func randomCastSubject(serverType string) string {
	return fmt.Sprintf("randomcast.%s", serverType)
}

func rpcSubject(serverID uint32) string {
	return fmt.Sprintf("rpc.%d", serverID)
}

func randomRpcSubject(serverType string) string {
	return fmt.Sprintf("randomrpc.%s", serverType)
}

func (m *module) beforeStop() error {
	m.consCtx.Stop()
	for _, sub := range m.subs {
		sub.Drain()
	}
	return nil
}

func (m *module) afterStop() error {
	<-m.js.PublishAsyncComplete()
	m.conn.Close()
	return nil
}

func (m *module) streamMsgHandler(msg jetstream.Msg) {
	defer util.RecoverPanic()
	msg.Ack()
	if expires := msg.Headers().Get(idef.ConstKeyExpires); expires != "" {
		// 检测部分不重要但有一定时效性的消息是否超时
		// 比如往客户端推送的实时消息
		// 超时后直接丢弃
		n, err := strconv.Atoi(expires)
		if err == nil && util.NowNs() > time.Duration(n) {
			zlog.Debugf("message expired")
			return
		}
	}
	pkg, err := codec.Decode(msg.Data())
	if err != nil {
		zlog.Errorf("nats streamRecv decode msg error: %v", err)
		return
	}
	msgbus.Cast(pkg)
}

func (m *module) msgHandler(msg *nats.Msg) {
	defer util.RecoverPanic()
	pkg, err := codec.Decode(msg.Data)
	if err != nil {
		zlog.Errorf("nats recv decode msg error: %v", err)
		return
	}
	msgbus.Cast(pkg)
}

func (m *module) rpcHandler(msg *nats.Msg) {
	defer util.RecoverPanic()
	req, err := codec.Decode(msg.Data)
	rpcResp := &RPCResult{}
	if err != nil {
		rpcResp.Err = fmt.Sprintf("req decode msg error: %v", err)
		m.conn.Publish(msg.Reply, codec.Encode(rpcResp))
		return
	}
	msgbus.RPC(m, msgbus.Local(), req, func(resp any, err error) {
		if err != nil {
			rpcResp.Err = err.Error()
		} else {
			rpcResp.Data = codec.Marshal(resp)
		}
		m.conn.Publish(msg.Reply, codec.Encode(rpcResp))
	})
}
