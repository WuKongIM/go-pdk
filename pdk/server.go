package pdk

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/WuKongIM/GoPDK/pdk/pluginproto"
	"github.com/WuKongIM/wklog"
	"github.com/WuKongIM/wkrpc/client"
)

func RunServer(constructor func() interface{}, no string, opt ...Option) error {
	if constructor == nil {
		return fmt.Errorf("constructor is nil")
	}

	// 创建选项
	opts := newOptions()
	for _, o := range opt {
		o(opts)
	}
	opts.No = no
	// 创建rpc客户端
	rpcClient := newRpcClient(no)
	err := rpcClient.Start()
	if err != nil {
		return err
	}

	// 创建插件
	plugin := newPlugin(opts, constructor, rpcClient)

	// 创建服务
	s := newServer(rpcClient, plugin, opts)
	err = s.run()
	if err != nil {
		return err
	}
	rpcClient.Stop()
	return nil
}

func newRpcClient(pluginNo string) *client.Client {
	socketPath, err := getSocketPath()
	if err != nil {
		panic(err)
	}

	return client.New(fmt.Sprintf("unix://%s", socketPath), client.WithUid(pluginNo))
}

type Server struct {
	rpcClient  *client.Client
	pluginInfo *pluginproto.PluginInfo
	plugin     *plugin
	opts       *Options
	wklog.Log
}

func newServer(rpcClient *client.Client, plugin *plugin, opts *Options) *Server {

	return &Server{
		opts:      opts,
		rpcClient: rpcClient,
		plugin:    plugin,
		Log:       wklog.NewWKLog(fmt.Sprintf("Server[%s]", opts.No)),
	}
}

func (s *Server) run() error {

	s.routes()

	s.plugin.start()

	// 创建一个接收信号的channel
	sigChan := make(chan os.Signal, 1)
	// 通知该channel接收SIGTERM信号（例如，kill命令发送的信号）和SIGINT（Ctrl+C）信号
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	// 阻塞直到接收到信号
	<-sigChan

	s.stop()

	return nil
}

func (s *Server) stop() {

	s.plugin.stop()
}

func getSocketPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	pth := path.Join(homeDir, ".wukong", "run", "wukongim.sock")
	return pth, nil
}
