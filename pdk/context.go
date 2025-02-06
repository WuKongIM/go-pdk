package pdk

import (
	"encoding/json"

	"github.com/WuKongIM/GoPDK/pdk/pluginproto"
)

type Context struct {
	// 数据包
	Packet interface{}
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
