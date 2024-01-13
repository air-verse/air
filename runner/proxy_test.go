package runner

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
)

type reloader struct {
	subCh    chan struct{}
	reloadCh chan struct{}
}

func (r *reloader) AddSubscriber() *Subscriber {
	r.subCh <- struct{}{}
	return &Subscriber{reloadCh: r.reloadCh}
}

func (r *reloader) RemoveSubscriber(_ int) {
	close(r.subCh)
}

func (r *reloader) Reload() {}
func (r *reloader) Stop()   {}

func setupMockServer(t *testing.T) (srv *httptest.Server, port int) {
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "thin air")
	}))
	mockURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err = strconv.Atoi(mockURL.Port())
	if err != nil {
		t.Fatal(err)
	}
	return srv, port
}

func TestNewProxy(t *testing.T) {
	_ = os.Unsetenv(airWd)
	cfg := &cfgProxy{
		Enabled:   true,
		ProxyPort: 1111,
		AppPort:   2222,
	}
	proxy := NewProxy(cfg)
	if proxy.config == nil {
		t.Fatal("config should not be nil")
	}
	if proxy.server.Addr == "" {
		t.Fatal("server address should not be nil")
	}
}

func TestProxy_proxyHandler(t *testing.T) {
	srv, mockPort := setupMockServer(t)
	defer srv.Close()

	cfg := &cfgProxy{
		Enabled:   true,
		ProxyPort: 8090,
		AppPort:   mockPort,
	}
	proxy := NewProxy(cfg)

	req := httptest.NewRequest("GET", "http://localhost:8090/", nil)
	rec := httptest.NewRecorder()

	proxy.proxyHandler(rec, req)
	resp := rec.Result()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if got, exp := string(bodyBytes), "thin air"; got != exp {
		t.Errorf("expected %q but got %q", exp, got)
	}
}

func TestProxy_reloadHandler(t *testing.T) {
	srv, mockPort := setupMockServer(t)
	defer srv.Close()

	reloader := &reloader{subCh: make(chan struct{}), reloadCh: make(chan struct{})}
	cfg := &cfgProxy{
		Enabled:   true,
		ProxyPort: 8090,
		AppPort:   mockPort,
	}
	proxy := &Proxy{
		config: cfg,
		server: &http.Server{
			Addr: "localhost:8090",
		},
		stream: reloader,
	}

	req := httptest.NewRequest("GET", "http://localhost:8090/internal/reload", nil)
	rec := httptest.NewRecorder()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy.reloadHandler(rec, req)
	}()

	// wait for subscriber to be added
	<-reloader.subCh

	// send a reload event and wait for http response
	reloader.reloadCh <- struct{}{}
	close(reloader.reloadCh)
	wg.Wait()

	if !rec.Flushed {
		t.Errorf("request should have been flushed")
	}

	resp := rec.Result()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("reading body: %v", err)
	}
	if got, exp := string(bodyBytes), "data: reload\n\n"; got != exp {
		t.Errorf("expected %q but got %q", exp, got)
	}
}
