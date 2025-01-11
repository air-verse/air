package runner

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

type ProxyStream struct {
	mu          sync.Mutex
	subscribers map[int32]*Subscriber
	count       atomic.Int32
}

type StreamMessageType string

const (
	StreamMessageReload      StreamMessageType = "reload"
	StreamMessageBuildFailed StreamMessageType = "build-failed"
)

type StreamMessage struct {
	Type StreamMessageType
	Data interface{}
}

type BuildFailedMsg struct {
	Error   string `json:"error"`
	Command string `json:"command"`
	Output  string `json:"output"`
}

type Subscriber struct {
	id    int32
	msgCh chan StreamMessage
}

func NewProxyStream() *ProxyStream {
	return &ProxyStream{subscribers: make(map[int32]*Subscriber)}
}

func (stream *ProxyStream) Stop() {
	for id := range stream.subscribers {
		stream.RemoveSubscriber(id)
	}
	stream.count = atomic.Int32{}
}

func (stream *ProxyStream) AddSubscriber() *Subscriber {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	stream.count.Add(1)

	sub := &Subscriber{id: stream.count.Load(), msgCh: make(chan StreamMessage)}
	stream.subscribers[stream.count.Load()] = sub
	return sub
}

func (stream *ProxyStream) RemoveSubscriber(id int32) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if _, ok := stream.subscribers[id]; ok {
		close(stream.subscribers[id].msgCh)
		delete(stream.subscribers, id)
	}
}

func (stream *ProxyStream) Reload() {
	for _, sub := range stream.subscribers {
		sub.msgCh <- StreamMessage{
			Type: StreamMessageReload,
			Data: nil,
		}
	}
}

func (stream *ProxyStream) BuildFailed(err BuildFailedMsg) {
	for _, sub := range stream.subscribers {
		sub.msgCh <- StreamMessage{
			Type: StreamMessageBuildFailed,
			Data: err,
		}
	}
}

func (m StreamMessage) AsSSE() string {
	s := "event: " + string(m.Type) + "\n"
	s += "data: " + stringify(m.Data) + "\n"
	return s + "\n"
}

func stringify(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("{\"error\":\"Failed to marshal message: %s\"}", err)
	}
	return string(b)
}
