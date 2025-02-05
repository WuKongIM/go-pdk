package main

import (
	"fmt"

	"github.com/WuKongIM/GoPDK/pdk"
	"github.com/WuKongIM/GoPDK/pdk/pluginproto"
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

func (s Hello) Route(c *pdk.Route) {
}

// 消息发送前（适合敏感词过滤之类的插件）(同步调用)
func (s Hello) Send(c *pdk.Context) {

	sendPacket := c.Packet.(*pluginproto.SendPacket)
	sendPacket.Payload = []byte("{\"content\":\"hello\",\"type\":1}")
}

// 消息持久化后（适合消息搜索类插件）（默认异步调用）
func (s Hello) PersistAfter(c *pdk.Context) {
	fmt.Println("PersistAfter--->", c.Packet)
}

// 回复消息（适合AI类插件）（默认异步调用）
func (s Hello) Reply(c *pdk.Context) {

}
