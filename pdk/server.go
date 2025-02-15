package pdk

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
	"github.com/WuKongIM/wklog"
	"github.com/WuKongIM/wkrpc/client"
)

var S *Server

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
	// 设置全局对象
	S = s
	// 运行服务
	err = s.run()
	if err != nil {
		return err
	}
	// 停止
	rpcClient.Stop()
	s.stop()

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
	rpcClient *client.Client
	plugin    *plugin
	opts      *Options
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

// Request 向服务端发送请求
func (s *Server) Request(requestPath string, data []byte) ([]byte, error) {
	return s.plugin.request(requestPath, data)
}

// GetChannelMessages 获取频道消息
func (s *Server) GetChannelMessages(req *pluginproto.ChannelMessageBatchReq) (*pluginproto.ChannelMessageBatchResp, error) {
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}
	respData, err := s.Request("/channel/messages", data)
	if err != nil {
		return nil, err
	}
	resp := &pluginproto.ChannelMessageBatchResp{}
	err = resp.Unmarshal(respData)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ForwardHttp 转发插件http请求
func (s *Server) ForwardHttp(req *pluginproto.ForwardHttpReq) (*pluginproto.HttpResponse, error) {
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	if req.PluginNo == "" {
		req.PluginNo = s.opts.No
	}
	respData, err := s.Request("/plugin/httpForward", data)
	if err != nil {
		return nil, err
	}
	resp := &pluginproto.HttpResponse{}
	err = resp.Unmarshal(respData)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ConversationChannels 查询用户最近会话的频道
func (s *Server) ConversationChannels(uid string) (*pluginproto.ConversationChannelResp, error) {
	req := &pluginproto.ConversationChannelReq{
		Uid: uid,
	}
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}
	respData, err := s.Request("/conversation/channels", data)
	if err != nil {
		return nil, err
	}
	resp := &pluginproto.ConversationChannelResp{}
	err = resp.Unmarshal(respData)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ClusterChannelBelongNode 获取频道所属节点
func (s *Server) ClusterChannelBelongNode(req *pluginproto.ClusterChannelBelongNodeReq) (*pluginproto.ClusterChannelBelongNodeBatchResp, error) {
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}
	respData, err := s.Request("/cluster/channels/belongNode", data)
	if err != nil {
		return nil, err
	}
	resp := &pluginproto.ClusterChannelBelongNodeBatchResp{}
	err = resp.Unmarshal(respData)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// SandboxDir 插件沙箱目录 （插件数据可以保存到此目录下）
func (s *Server) SandboxDir() string {
	return s.plugin.sandbox
}

func (s *Server) run() error {

	s.routes()
	s.onMessage()

	s.plugin.start()

	// 创建一个接收信号的channel
	sigChan := make(chan os.Signal, 1)
	// 通知该channel接收SIGTERM信号（例如，kill命令发送的信号）和SIGINT（Ctrl+C）信号
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	// 阻塞直到接收到信号
	<-sigChan

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
