package main

import (
	"github.com/WuKongIM/GoPDK/pdk"
)

func main() {

	pdk.StartServer(New, Version, Priority)
}

var Version = "0.2"
var Priority = 1

type Config struct {
	Message string
}

type Hello struct {
	Config Config // 此配置将会显示在UI上，让用户可以配置
	s      *pdk.Server
}

func New(s *pdk.Server) interface{} {
	return &Hello{
		s: s,
	}
}

func (s Hello) Route(c *pdk.Route) {
}

// 消息持久化前（适合敏感词过滤插件）
func (s Hello) PersistBefore(c *pdk.Context) {

}

// 消息持久化后（适合消息搜索插件）
func (s Hello) PersistAfter(c *pdk.Context) {

}

// 回复消息（适合AI插件）
func (s Hello) Reply(c *pdk.Context) {

}
