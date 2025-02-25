package pdk

import (
	"errors"
	"strings"

	"github.com/WuKongIM/go-pdk/pdk/pluginproto"
)

type Stream struct {
	streamNo   string
	streamInfo *pluginproto.Stream
	s          *Server
}

func newStream(streamNo string, streamInfo *pluginproto.Stream, s *Server) *Stream {
	return &Stream{
		streamNo:   streamNo,
		streamInfo: streamInfo,
		s:          s,
	}
}

func (s *Stream) Close() error {

	if strings.TrimSpace(s.streamNo) == "" {
		return errors.New("streamNo is empty")
	}
	return s.s.RequestStreamClose(s.streamNo)
}

func (s *Stream) Write(data []byte) error {

	if s.streamInfo != nil && s.streamInfo.Header != nil {
		s.streamInfo.Header.RedDot = false // 流消息不需要红点
	} else {
		s.streamInfo.Header = &pluginproto.Header{RedDot: false}
	}

	return s.s.RequestStreamWrite(&pluginproto.StreamWriteReq{
		Header:      s.streamInfo.Header,
		StreamNo:    s.streamNo,
		FromUid:     s.streamInfo.FromUid,
		ChannelId:   s.streamInfo.ChannelId,
		ChannelType: s.streamInfo.ChannelType,
		Payload:     data,
	})
}

type StreamOption func(*pluginproto.Stream)

func StreamWithHeader(header *pluginproto.Header) StreamOption {
	return func(s *pluginproto.Stream) {
		s.Header = header
	}
}

func StreamWithPayload(payload Payload) StreamOption {
	return func(s *pluginproto.Stream) {
		data, err := payload.Encode()
		if err != nil {
			panic(err)
		}
		s.Payload = data
	}
}
