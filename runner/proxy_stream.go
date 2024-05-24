package runner

import (
	"sync"
	"sync/atomic"
)

type ProxyStream struct {
	mu          sync.Mutex
	subscribers map[int32]*Subscriber
	count       atomic.Int32
}

type Subscriber struct {
	id       int32
	reloadCh chan struct{}
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

	sub := &Subscriber{id: stream.count.Load(), reloadCh: make(chan struct{})}
	stream.subscribers[stream.count.Load()] = sub
	return sub
}

func (stream *ProxyStream) RemoveSubscriber(id int32) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	if _, ok := stream.subscribers[id]; ok {
		close(stream.subscribers[id].reloadCh)
		delete(stream.subscribers, id)
	}
}

func (stream *ProxyStream) Reload() {
	for _, sub := range stream.subscribers {
		sub.reloadCh <- struct{}{}
	}
}
