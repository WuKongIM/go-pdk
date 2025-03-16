package pdk

import (
	"encoding/json"
	"errors"
	"net/http"

	wkproto "github.com/WuKongIM/WuKongIMGoProto"
	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
	"go.uber.org/zap"
)

type Context struct {
	// 发送包
	SendPacket *pluginproto.SendPacket
	// 消息包
	Messages []*pluginproto.Message
	// 接收包
	RecvPacket *pluginproto.RecvPacket
	s          *Server
}

func newMessageContext(s *Server, messages []*pluginproto.Message) *Context {
	return &Context{
		s:        s,
		Messages: messages,
	}
}

func newSendContext(s *Server, sendPacket *pluginproto.SendPacket) *Context {
	return &Context{
		s:          s,
		SendPacket: sendPacket,
	}
}

func newRecvContext(s *Server, recvPacket *pluginproto.RecvPacket) *Context {
	return &Context{
		s:          s,
		RecvPacket: recvPacket,
	}
}

// 打开流
func (c *Context) OpenStream(opt ...StreamOption) (*Stream, error) {

	if c.RecvPacket == nil {
		return nil, errors.New("RecvPacket is nil")
	}

	channelId := c.RecvPacket.ChannelId
	if c.RecvPacket.ChannelType == uint32(wkproto.ChannelTypePerson) {
		channelId = c.RecvPacket.FromUid
	}
	streamInfo := &pluginproto.Stream{
		FromUid:     c.RecvPacket.ToUid,
		ChannelId:   channelId,
		ChannelType: c.RecvPacket.ChannelType,
	}
	for _, o := range opt {
		o(streamInfo)
	}
	resp, err := c.s.RequestStreamOpen(streamInfo)
	if err != nil {
		return nil, err
	}

	stream := newStream(resp.StreamNo, streamInfo, c.s)

	return stream, nil
}

// 回复消息
func (c *Context) Reply(payload []byte, opt ...ReplyOption) {
	if c.RecvPacket == nil {
		return
	}

	opts := &ReplyOptions{}
	for _, o := range opt {
		o(opts)
	}

	channelId := c.RecvPacket.ChannelId
	channelType := c.RecvPacket.ChannelType
	if c.RecvPacket.ChannelType == uint32(wkproto.ChannelTypePerson) {
		channelId = c.RecvPacket.FromUid
	}

	_, err := c.s.RequestSend(&pluginproto.SendReq{
		Header:      opts.Header,
		ClientMsgNo: opts.ClientMsgNo,
		FromUid:     c.RecvPacket.ToUid,
		ChannelId:   channelId,
		ChannelType: channelType,
		Payload:     payload,
	})
	if err != nil {
		c.s.Error("Reply error", zap.Error(err))
	}
}

type HttpContext struct {
	Request  *pluginproto.HttpRequest
	Response *pluginproto.HttpResponse
}

func (h *HttpContext) GetQuery(key string) string {
	if h.Request.Query == nil {
		return ""
	}
	return h.Request.Query[key]
}

func (h *HttpContext) BindJSON(v interface{}) error {

	if len(h.Request.Body) == 0 {
		return nil
	}
	return json.Unmarshal(h.Request.Body, v)
}

func (h *HttpContext) GetHeader(key string) string {
	if h.Request.Headers == nil {
		return ""
	}
	return h.Request.Headers[key]
}

func (h *HttpContext) JSON(code int, v interface{}) {
	data, _ := json.Marshal(v)
	h.Response.Body = data
	h.Response.Status = int32(code)
	h.Response.Headers["Content-Type"] = "application/json"
}

func (h *HttpContext) ResponseError(err error) {
	data, _ := json.Marshal(map[string]interface{}{
		"msg":    err.Error(),
		"status": http.StatusBadGateway,
	})
	h.Response.Status = http.StatusBadGateway
	h.Response.Body = data
	h.Response.Headers["Content-Type"] = "application/json"
}
