package pdk

type Options struct {
	No               string // 插件唯一编号
	PersistAfterSync bool   // PersistAfter方法是否同步调用
	ReplySync        bool   // Reply方法是否同步调用
	Version          string
	Priority         int32
}

func newOptions() *Options {
	return &Options{
		Version:  "0.0.0",
		Priority: 0,
	}
}

type Option func(*Options)

func WithNo(no string) Option {
	return func(o *Options) {
		o.No = no
	}
}

func WithVersion(version string) Option {
	return func(o *Options) {
		o.Version = version
	}
}

func WithPriority(priority int32) Option {
	return func(o *Options) {
		o.Priority = priority
	}
}

func WithPersistAfterSync(persistAfterSync bool) Option {
	return func(o *Options) {
		o.PersistAfterSync = persistAfterSync
	}
}

func WithReplySync(replySync bool) Option {
	return func(o *Options) {
		o.ReplySync = replySync
	}
}
