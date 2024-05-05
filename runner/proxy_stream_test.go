package runner

import (
	"sync"
	"testing"
)

func find(s map[int]*Subscriber, id int) bool {
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
		go func(i int) {
			defer wg.Done()
			_ = stream.AddSubscriber()
		}(i)
	}
	wg.Wait()

	if got, exp := len(stream.subscribers), 10; got != exp {
		t.Errorf("expected %d but got %d", exp, got)
	}

	go func() {
		stream.Reload()
	}()

	reloadCount := 0
	for _, sub := range stream.subscribers {
		wg.Add(1)
		go func(sub *Subscriber) {
			defer wg.Done()
			<-sub.reloadCh
			reloadCount++
		}(sub)
	}
	wg.Wait()

	if got, exp := reloadCount, 10; got != exp {
		t.Errorf("expected %d but got %d", exp, got)
	}

	stream.RemoveSubscriber(2)
	stream.AddSubscriber()
	if got, exp := find(stream.subscribers, 2), false; got != exp {
		t.Errorf("expected subscriber found to be %t but got %t", exp, got)
	}
	if got, exp := find(stream.subscribers, 11), true; got != exp {
		t.Errorf("expected subscriber found to be %t but got %t", exp, got)
	}

	stream.Stop()
	if got, exp := len(stream.subscribers), 0; got != exp {
		t.Errorf("expected %d but got %d", exp, got)
	}
}
