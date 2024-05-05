package runner

import (
	"sync"
)

type ProxyStream struct {
	sync.Mutex
	subscribers map[int]*Subscriber
	count       int
}

type Subscriber struct {
	id       int
	reloadCh chan struct{}
}

func NewProxyStream() *ProxyStream {
	return &ProxyStream{subscribers: make(map[int]*Subscriber)}
}

func (stream *ProxyStream) Stop() {
	for id := range stream.subscribers {
		stream.RemoveSubscriber(id)
	}
	stream.count = 0
}

func (stream *ProxyStream) AddSubscriber() *Subscriber {
	stream.Lock()
	defer stream.Unlock()
	stream.count++

	sub := &Subscriber{id: stream.count, reloadCh: make(chan struct{})}
	stream.subscribers[stream.count] = sub
	return sub
}

func (stream *ProxyStream) RemoveSubscriber(id int) {
	stream.Lock()
	defer stream.Unlock()
	close(stream.subscribers[id].reloadCh)
	delete(stream.subscribers, id)
}

func (stream *ProxyStream) Reload() {
	for _, sub := range stream.subscribers {
		sub.reloadCh <- struct{}{}
	}
}
