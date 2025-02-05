package pdk

import (
	"fmt"

	"github.com/WuKongIM/GoPDK/pdk/pluginproto"
	"github.com/WuKongIM/wkrpc/client"
	"github.com/WuKongIM/wkrpc/proto"
	"go.uber.org/zap"
)

func (s *Server) routes() {
	s.rpcClient.OnMessage(func(msg *proto.Message) {
		fmt.Println("msg---->", msg.MsgType, len(msg.Content))
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
	s.rpcClient.Route("/plugin/send", s.send)
}

func (s *Server) send(c *client.Context) {
	sendPacket := &pluginproto.SendPacket{}
	err := sendPacket.Unmarshal(c.Body())
	if err != nil {
		s.Error("unmarshal send packet error", zap.Error(err))
		c.WriteErr(err)
		return
	}

	s.plugin.send(&Context{
		Packet: sendPacket,
	})

	resultData, err := sendPacket.Marshal()
	if err != nil {
		s.Error("marshal send packet error", zap.Error(err))
		c.WriteErr(err)
		return
	}
	c.Write(resultData)
}

func (s *Server) persistAfter(c *client.Context) {

}

func (s *Server) handlePersistAfter(messages *pluginproto.MessageBatch) {
	s.plugin.persistAfter(&Context{
		Packet: messages,
	})
}
