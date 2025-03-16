package pdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
	"github.com/WuKongIM/wklog"
	"github.com/WuKongIM/wkrpc/client"
	"github.com/WuKongIM/wkrpc/proto"
	"go.uber.org/zap"
)

type PluginMethod string

const (
	PluginSend         PluginMethod = "Send"
	PluginPersistAfter PluginMethod = "PersistAfter"
	PluginReceive      PluginMethod = "Receive"
	PluginRoute        PluginMethod = "Route"
	PluginConfigUpdate PluginMethod = "ConfigUpdate"
)

func (p PluginMethod) String() string {
	return string(p)
}

type PluginMethodType uint32

const (
	PluginMethodTypeSend         PluginMethodType = 1
	PluginMethodTypePersistAfter PluginMethodType = 2
	PluginMethodTypeReceive      PluginMethodType = 3
	PluginMethodTypeRoute        PluginMethodType = 4
	PluginMethodTypeConfigUpdate PluginMethodType = 5
)

func (p PluginMethod) Type() PluginMethodType {
	switch p {
	case PluginSend:
		return PluginMethodTypeSend
	case PluginPersistAfter:
		return PluginMethodTypePersistAfter
	case PluginReceive:
		return PluginMethodTypeReceive
	case PluginRoute:
		return PluginMethodTypeRoute
	case PluginConfigUpdate:
		return PluginMethodTypeConfigUpdate
	}
	return 0
}

type plugin struct {
	constructor         func() interface{}
	opts                *Options
	rpcClient           *client.Client
	methods             []string
	handlers            map[string]func(*Context)
	routeHandler        func(*Route)
	stopHandler         func()
	setupHandler        func()
	configUpdateHandler func()
	sandbox             string // 沙箱目录
	r                   *Route // http 路由
	wklog.Log
	setupOnce    sync.Once
	serverNodeId uint64                      // 服务节点id
	cfgTemplate  *pluginproto.ConfigTemplate // 插件配置模版
	configType   reflect.Type                // 配置对象的类型
	instance     interface{}
}

func newPlugin(opts *Options, constructor func() interface{}, rpcClient *client.Client) *plugin {

	instance := constructor()
	t := reflect.TypeOf(instance)

	configType := getPluginConfigTemplateType(t)

	// 获取路由处理函数
	routeHandler := getRouteHandler(instance)
	var r *Route
	if routeHandler != nil {
		r = newRoute()
		routeHandler(r)
	}

	// 获取停止处理函数
	stopHandler := getStopHandler(instance)

	// setup handler
	setupHandler := getSetupHandler(instance)

	// config update handler
	configUpdateHandler := getConfigUpdateHandler(instance)

	pg := &plugin{
		constructor:         constructor,
		opts:                opts,
		rpcClient:           rpcClient,
		methods:             getHandlerNames(t),
		handlers:            getHandlers(instance),
		routeHandler:        routeHandler,
		stopHandler:         stopHandler,
		setupHandler:        setupHandler,
		r:                   r,
		Log:                 wklog.NewWKLog(fmt.Sprintf("Plugin[%s]", opts.No)),
		cfgTemplate:         getPluginConfigTemplate(configType),
		configType:          configType,
		configUpdateHandler: configUpdateHandler,
		instance:            instance,
	}
	if strings.TrimSpace(opts.Sandbox) != "" {
		pg.sandbox = opts.Sandbox
		pg.initLogger()
	}
	return pg
}

func (p *plugin) start() {
	if p.rpcClient.IsAuthed() {
		_ = p.requestStart()

		p.setupOnce.Do(func() {
			p.initLogger()
			if p.setupHandler != nil {
				p.setupHandler()
			}
		})
	}

	p.rpcClient.OnConnectChanged(func(status client.ConnStatus) {
		if status == client.Authed {
			_ = p.requestStart()
			p.setupOnce.Do(func() {
				p.initLogger()
				if p.setupHandler != nil {
					p.setupHandler()
				}
			})
		}
	})

}

func (p *plugin) stop() {
	if p.stopHandler != nil {
		p.stopHandler()
	}
}

func (p *plugin) initLogger() {
	// 初始化日志目录
	opts := wklog.NewOptions()
	opts.LogDir = path.Join(p.sandbox, "logs")
	wklog.Configure(opts)
}

func (p *plugin) send(ctx *Context) {
	handler := p.handlers[PluginSend.String()]
	if handler != nil {
		handler(ctx)
	}
}

func (p *plugin) receive(ctx *Context) {
	handler := p.handlers[PluginReceive.String()]
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

func (p *plugin) configUpdate(cfg map[string]interface{}) {
	if p.configType == nil {
		return
	}

	// 将map配置填充到Config结构体中
	config := reflect.New(p.configType).Interface()
	err := fillConfig(cfg, config)
	if err != nil {
		p.Error("fill config error", zap.Error(err))
		return
	}

	// 设置Config
	v := reflect.ValueOf(p.instance).Elem()
	cfgValue := v.FieldByName("Config")
	if !cfgValue.IsValid() {
		p.Warn("config field not found")
		return
	}
	cfgValue.Set(reflect.ValueOf(config).Elem())

	// 通知插件
	if p.configUpdateHandler != nil {
		p.configUpdateHandler()
	}

}

// fillConfig 将 cfg map 类型的数据填充到 config 结构体中
func fillConfig(cfg map[string]interface{}, config interface{}) error {
	v := reflect.ValueOf(config).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}

		if value, ok := cfg[jsonTag]; ok {
			fieldValue := v.Field(i)
			if fieldValue.CanSet() {
				val := reflect.ValueOf(value)
				if val.Type().ConvertibleTo(fieldValue.Type()) {
					fieldValue.Set(val.Convert(fieldValue.Type()))
				} else {
					return fmt.Errorf("cannot convert %v to %v", val.Type(), fieldValue.Type())
				}
			}
		}
	}
	return nil
}

func (p *plugin) requestStart() error {
	pluginInfo := p.getPluginInfo()
	data, err := pluginInfo.Marshal()
	if err != nil {
		panic(err)
	}
	resultData, err := p.request("/plugin/start", data)
	if err != nil {
		return err
	}

	resp := &pluginproto.StartupResp{}
	err = resp.Unmarshal(resultData)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("startup failed: %s", resp.ErrMsg)
	}
	p.sandbox = resp.SandboxDir
	p.serverNodeId = resp.NodeId
	if len(resp.Config) > 0 {
		var config map[string]interface{}
		err = json.Unmarshal(resp.Config, &config)
		if err != nil {
			p.Warn("unmarshal config error", zap.Error(err))
		} else {
			p.configUpdate(config)
		}
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
		ConfigTemplate:   p.cfgTemplate,
	}

}

func getPluginConfigTemplate(configType reflect.Type) *pluginproto.ConfigTemplate {
	if configType == nil {
		return &pluginproto.ConfigTemplate{}
	}
	return &pluginproto.ConfigTemplate{
		Fields: getFields(configType),
	}
}

func getPluginConfigTemplateType(t reflect.Type) reflect.Type {
	switch t.Kind() {
	case reflect.Ptr:
		return getPluginConfigTemplateType(t.Elem())
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.Name == "Config" {
				return field.Type
			}
		}
	}
	return nil

}

func getFields(t reflect.Type) []*pluginproto.Field {
	fields := []*pluginproto.Field{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// ignore unexported fields
		if len(field.PkgPath) != 0 {
			continue
		}

		name := field.Tag.Get("json")
		label := field.Tag.Get("label")
		fieldType := getFieldType(field.Type)
		if fieldType == "" {
			continue
		}

		if label == "" {
			label = name
		}

		fields = append(fields, &pluginproto.Field{
			Name:  name,
			Label: label,
			Type:  fieldType,
		})
	}
	return fields
}

func getFieldType(t reflect.Type) string {
	if t == reflect.TypeOf(SecretKey("")) {
		return "secret"
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "number"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "bool"
	}
	return ""
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
	PluginReceive.String(),
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
	if h, ok := instance.(receive); ok {
		handlers[PluginReceive.String()] = h.Receive
	}
	return handlers
}

func getRouteHandler(instance interface{}) func(*Route) {
	if h, ok := instance.(route); ok {
		return h.Route
	}
	return nil
}

func getStopHandler(instance interface{}) func() {
	if h, ok := instance.(stop); ok {
		return h.Stop
	}
	return nil
}

func getSetupHandler(instance interface{}) func() {
	if h, ok := instance.(setup); ok {
		return h.Setup
	}
	return nil
}

func getConfigUpdateHandler(instance interface{}) func() {
	if h, ok := instance.(configUpdate); ok {
		return h.ConfigUpdate
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

	receive interface {
		Receive(*Context)
	}

	route interface {
		Route(*Route)
	}

	stop interface {
		Stop()
	}

	setup interface {
		Setup()
	}

	configUpdate interface {
		ConfigUpdate()
	}
)
