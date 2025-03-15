package pdk

import (
	"encoding/json"
	"fmt"
	"syscall"

	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
	"github.com/WuKongIM/wkrpc/client"
	"github.com/WuKongIM/wkrpc/proto"
	"go.uber.org/zap"
)

func (s *Server) routes() {
	s.rpcClient.Route("/plugin/send", s.send)                        // 发送消息请求
	s.rpcClient.Route("/plugin/persist_after", s.persistAfter)       // 存储消息后请求
	s.rpcClient.Route("/plugin/route", s.route)                      // 路由请求
	s.rpcClient.Route("/plugin/reply", s.reply)                      // 回复请求
	s.rpcClient.Route("/plugin/config_update", s.handleConfigUpdate) // WuKongIM请求更新插件配置
	s.rpcClient.Route("/stop", s.handleStop)                         // WuKongIM请求停止插件
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
		case uint32(PluginMethodTypeReply):
			recvPacket := &pluginproto.RecvPacket{}
			err := recvPacket.Unmarshal(msg.Content)
			if err != nil {
				s.Error("unmarshal recv packet error", zap.Error(err))
				return
			}
			s.handleReply(recvPacket)
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

	ctx := newSendContext(s, sendPacket)
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

func (s *Server) reply(c *client.Context) {
	recvPacket := &pluginproto.RecvPacket{}
	err := recvPacket.Unmarshal(c.Body())
	if err != nil {
		s.Error("unmarshal recv packet error", zap.Error(err))
		c.WriteErr(err)
		return
	}
	s.handleReply(recvPacket)
}

func (s *Server) handlePersistAfter(messageBatch *pluginproto.MessageBatch) {
	ctx := newMessageContext(s, messageBatch.Messages)
	s.plugin.persistAfter(ctx)
}

func (s *Server) handleReply(recvPacket *pluginproto.RecvPacket) {
	ctx := newRecvContext(s, recvPacket)
	s.plugin.reply(ctx)
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

func (s *Server) handleStop(c *client.Context) {

	fmt.Println("plugin stop...")

	select {
	case s.sigChan <- syscall.SIGTERM:
	default:
	}
	c.WriteOk()
}

func (s *Server) handleConfigUpdate(c *client.Context) {

	var config map[string]interface{}
	err := json.Unmarshal(c.Body(), &config)
	if err != nil {
		s.Error("unmarshal config error", zap.Error(err))
		c.WriteErr(err)
		return
	}
	s.plugin.configUpdate(config)
	c.WriteOk()
}
