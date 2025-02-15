package main

import (
	"fmt"
	"net/http"

	"github.com/WuKongIM/go-pdk/pdk"
	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
)

var Version = "0.0.1"   // 插件版本
var Priority = int32(1) // 插件优先级

func main() {

	err := pdk.RunServer(New, "wk.plugin.hello", pdk.WithVersion(Version), pdk.WithPriority(Priority))
	if err != nil {
		panic(err)
	}
}

type Hello struct {
}

func New() interface{} {
	return &Hello{}
}

// 插件初始化
func (s Hello) Setup() {
	fmt.Println("plugin setup...")
}

// 注册http路由
func (s Hello) Route(c *pdk.Route) {
	// http://127.0.0.1:5001/plugins/wk.plugin.hello/hello
	c.GET("/hello", s.sayHello)
}

// 消息发送前（适合敏感词过滤之类的插件）(同步调用)
func (s Hello) Send(c *pdk.Context) {

	sendPacket := c.Packet.(*pluginproto.SendPacket)
	sendPacket.Payload = []byte("{\"content\":\"hello\",\"type\":1}")
}

// 消息持久化后（适合消息搜索类插件）（默认异步调用）
func (s Hello) PersistAfter(c *pdk.Context) {
	fmt.Println("PersistAfter:", c.Packet)
}

// 回复消息（适合AI类插件）（默认异步调用）
func (s Hello) Reply(c *pdk.Context) {

}

// 插件停止
func (s Hello) Stop() {
	fmt.Println("plugin stop...")
}

func (s Hello) sayHello(c *pdk.HttpContext) {
	name := c.GetQuery("name")

	c.JSON(http.StatusOK, map[string]interface{}{
		"say": fmt.Sprintf("hello %s", name),
	})
}
