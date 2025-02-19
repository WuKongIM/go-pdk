package pdk

import (
	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
	"github.com/WuKongIM/wkrpc/client"
	"github.com/WuKongIM/wkrpc/proto"
	"go.uber.org/zap"
)

func (s *Server) routes() {
	s.rpcClient.Route("/plugin/send", s.send)
	s.rpcClient.Route("/plugin/persist_after", s.persistAfter)
	s.rpcClient.Route("/plugin/route", s.route)
}

// 收到消息
func (s *Server) onMessage() {
	s.rpcClient.OnMessage(func(msg *proto.Message) {
		switch msg.MsgType {
		case uint32(PluginMethodTypePersistAfter):
			messages := &pluginproto.MessageBatch{}
			err := messages.Unmarshal(msg.Content)
			if err != nil {
				s.Error("unmarshal message batch error", zap.Error(err))
				return
			}
			s.handlePersistAfter(messages)
		}
	})
}

func (s *Server) send(c *client.Context) {
	sendPacket := &pluginproto.SendPacket{}
	err := sendPacket.Unmarshal(c.Body())
	if err != nil {
		s.Error("unmarshal send packet error", zap.Error(err))
		c.WriteErr(err)
		return
	}

	ctx := newContext(s, sendPacket, nil)
	s.plugin.send(ctx)

	resultData, err := sendPacket.Marshal()
	if err != nil {
		s.Error("marshal send packet error", zap.Error(err))
		c.WriteErr(err)
		return
	}
	c.Write(resultData)
}

func (s *Server) persistAfter(c *client.Context) {
	messages := &pluginproto.MessageBatch{}
	err := messages.Unmarshal(c.Body())
	if err != nil {
		s.Error("unmarshal message batch error", zap.Error(err))
		c.WriteErr(err)
		return
	}

	s.handlePersistAfter(messages)
	c.WriteOk()
}

func (s *Server) handlePersistAfter(messageBatch *pluginproto.MessageBatch) {
	ctx := newContext(s, nil, messageBatch.Messages)
	s.plugin.persistAfter(ctx)
}

func (s *Server) route(c *client.Context) {

	req := &pluginproto.HttpRequest{}
	err := req.Unmarshal(c.Body())
	if err != nil {
		s.Error("unmarshal http request error", zap.Error(err))
		c.WriteErr(err)
		return
	}

	ctx := &HttpContext{
		Request: req,
		Response: &pluginproto.HttpResponse{
			Headers: map[string]string{},
		},
	}

	// route
	s.plugin.route(ctx)

	// response
	data, err := ctx.Response.Marshal()
	if err != nil {
		s.Error("marshal http response error", zap.Error(err))
		c.WriteErr(err)
		return
	}
	c.Write(data)
}
