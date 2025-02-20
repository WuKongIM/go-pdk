package main

import (
	"encoding/json"
	"fmt"

	"github.com/WuKongIM/go-pdk/pdk"
	"github.com/WuKongIM/wklog"
	"go.uber.org/zap"
)

var PluginNo = "wk.plugin.robot" // 插件编号
var Version = "0.0.1"            // 插件版本
var Priority = int32(1)          // 插件优先级

func main() {
	err := pdk.RunServer(New, PluginNo, pdk.WithVersion(Version), pdk.WithPriority(Priority))
	if err != nil {
		panic(err)
	}
}

type Robot struct {
	wklog.Log
}

func New() interface{} {
	return &Robot{
		Log: wklog.NewWKLog("robot"),
	}
}

// 实现插件的回复消息方法
func (r Robot) Reply(c *pdk.Context) {

	fmt.Println("plugin reply...", c.RecvPacket)
	//打开流
	stream, err := c.OpenStream()
	if err != nil {
		r.Error("open stream error:", zap.Error(err))
		return
	}

	data, _ := json.Marshal(map[string]interface{}{
		"type":    1,
		"content": "hello",
	})

	stream.Write(data)

	stream.Close()
}
