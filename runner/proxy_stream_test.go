package runner

import (
	"sync"
	"sync/atomic"
	"testing"
)

func find(s map[int32]*Subscriber, id int32) bool {
	for _, sub := range s {
		if sub.id == id {
			return true
		}
	}
	return false
}

func TestProxyStream(t *testing.T) {
	stream := NewProxyStream()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			_ = stream.AddSubscriber()
		}(i)
	}
	wg.Wait()

	if got, exp := len(stream.subscribers), 10; got != exp {
		t.Errorf("expect subscribers count to be %d, got %d", exp, got)
	}

	doneCh := make(chan struct{})
	go func() {
		stream.Reload()
		doneCh <- struct{}{}
	}()

	var reloadCount atomic.Int32
	for _, sub := range stream.subscribers {
		wg.Add(1)
		go func(sub *Subscriber) {
			defer wg.Done()
			<-sub.reloadCh
			reloadCount.Add(1)
		}(sub)
	}
	wg.Wait()
	<-doneCh

	if got, exp := reloadCount.Load(), int32(10); got != exp {
		t.Errorf("expect reloadCount %d, got %d", exp, got)
	}

	stream.RemoveSubscriber(2)
	if find(stream.subscribers, 2) {
		t.Errorf("expected subscriber 2 not to be found")
	}

	stream.AddSubscriber()
	if !find(stream.subscribers, 11) {
		t.Errorf("expected subscriber 11 to be found")
	}

	stream.Stop()
	if got, exp := len(stream.subscribers), 0; got != exp {
		t.Errorf("expected subscribers count to be %d, got %d", exp, got)
	}
}
