package pdk

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/WuKongIM/GoPDK/pdk/pluginproto"
	"github.com/WuKongIM/wklog"
	"github.com/WuKongIM/wkrpc/client"
	"github.com/WuKongIM/wkrpc/proto"
)

type PluginMethod string

const (
	PluginSend         PluginMethod = "Send"
	PluginPersistAfter PluginMethod = "PersistAfter"
	PluginReply        PluginMethod = "Reply"
	PluginRoute        PluginMethod = "Route"
)

func (p PluginMethod) String() string {
	return string(p)
}

type PluginMethodType uint32

const (
	PluginMethodTypeSend         PluginMethodType = 1
	PluginMethodTypePersistAfter PluginMethodType = 2
	PluginMethodTypeReply        PluginMethodType = 3
	PluginMethodTypeRoute        PluginMethodType = 4
)

func (p PluginMethod) Type() PluginMethodType {
	switch p {
	case PluginSend:
		return PluginMethodTypeSend
	case PluginPersistAfter:
		return PluginMethodTypePersistAfter
	case PluginReply:
		return PluginMethodTypeReply
	case PluginRoute:
		return PluginMethodTypeRoute
	}
	return 0
}

type plugin struct {
	handlerType  reflect.Type
	constructor  func() interface{}
	opts         *Options
	rpcClient    *client.Client
	methods      []string
	handlers     map[string]func(*Context)
	routeHandler func(*Route)
	r            *Route // http 路由
	wklog.Log
}

func newPlugin(opts *Options, constructor func() interface{}, rpcClient *client.Client) *plugin {

	instance := constructor()
	t := reflect.TypeOf(constructor())

	routeHandler := getRouteHandler(instance)

	var r *Route
	if routeHandler != nil {
		r = newRoute()
		routeHandler(r)
	}

	return &plugin{
		handlerType:  t,
		constructor:  constructor,
		opts:         opts,
		rpcClient:    rpcClient,
		methods:      getHandlerNames(t),
		handlers:     getHandlers(instance),
		routeHandler: routeHandler,
		r:            r,
		Log:          wklog.NewWKLog(fmt.Sprintf("Plugin[%s]", opts.No)),
	}
}

func (p *plugin) start() {
	if p.rpcClient.IsAuthed() {
		_ = p.requestStart()
	}

	p.rpcClient.OnConnectChanged(func(status client.ConnStatus) {
		if status == client.Authed {
			_ = p.requestStart()
		}
	})

}

func (p *plugin) stop() {

}

func (p *plugin) send(ctx *Context) {
	handler := p.handlers[PluginSend.String()]
	if handler != nil {
		handler(ctx)
	}
}

func (p *plugin) persistAfter(ctx *Context) {
	handler := p.handlers[PluginPersistAfter.String()]
	if handler != nil {
		handler(ctx)
	}
}

func (p *plugin) route(ctx *HttpContext) {
	if p.r == nil {
		p.Warn("route handler not found")
		return
	}
	p.r.handle(ctx)
}

func (p *plugin) requestStart() error {
	pluginInfo := p.getPluginInfo()
	data, err := pluginInfo.Marshal()
	if err != nil {
		panic(err)
	}
	_, err = p.request("/plugin/start", data)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) request(ph string, data []byte) ([]byte, error) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	resp, err := p.rpcClient.RequestWithContext(timeoutCtx, ph, data)
	if err != nil {
		return nil, err
	}
	if resp.Status != proto.StatusOK {
		return nil, fmt.Errorf("status: %d, message: %s", resp.Status, string(resp.Body))
	}
	return resp.Body, nil
}

func (p *plugin) getPluginInfo() *pluginproto.PluginInfo {
	name, err := getName()
	if err != nil {
		panic(err)
	}
	return &pluginproto.PluginInfo{
		No:               p.opts.No,
		Name:             name,
		PersistAfterSync: p.opts.PersistAfterSync,
		ReplySync:        p.opts.ReplySync,
		Version:          p.opts.Version,
		Priority:         p.opts.Priority,
		Methods:          p.methods,
	}

}

// 获取插件名称
func getName() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	name := path.Base(execPath)
	return name, nil
}

var methodNames = [...]string{
	PluginSend.String(),
	PluginPersistAfter.String(),
	PluginReply.String(),
	PluginRoute.String(),
}

func getHandlerNames(t reflect.Type) []string {
	handlers := []string{}
	for _, name := range methodNames {
		_, hasIt := t.MethodByName(name)
		if hasIt {
			handlers = append(handlers, name)
		}
	}
	return handlers
}

func getHandlers(instance interface{}) map[string]func(*Context) {
	handlers := map[string]func(*Context){}

	if h, ok := instance.(send); ok {
		handlers[PluginSend.String()] = h.Send
	}
	if h, ok := instance.(persistAfter); ok {
		handlers[PluginPersistAfter.String()] = h.PersistAfter
	}
	if h, ok := instance.(reply); ok {
		handlers[PluginReply.String()] = h.Reply
	}
	return handlers
}

func getRouteHandler(instance interface{}) func(*Route) {
	if h, ok := instance.(route); ok {
		return h.Route
	}
	return nil
}

type (
	send interface {
		Send(*Context)
	}
	persistAfter interface {
		PersistAfter(*Context)
	}

	reply interface {
		Reply(*Context)
	}

	route interface {
		Route(*Route)
	}
)
