package pdk

import (
	"encoding/json"
	"net/http"

	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
)

type Context struct {
	// 发送包
	SendPacket *pluginproto.SendPacket
	// 消息包
	Messages []*pluginproto.Message
	s        *Server
}

func newContext(s *Server, sendPacket *pluginproto.SendPacket, messages []*pluginproto.Message) *Context {
	return &Context{
		s:          s,
		SendPacket: sendPacket,
		Messages:   messages,
	}
}

// 打开流
func (c *Context) OpenStream(streamInfo *pluginproto.Stream) (*Stream, error) {

	resp, err := c.s.RequestStreamOpen(streamInfo)
	if err != nil {
		return nil, err
	}

	stream := newStream(resp.StreamNo, streamInfo, c.s)

	return stream, nil
}

// 回复消息
func (c *Context) Reply(payload []byte) error {

	return nil
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
