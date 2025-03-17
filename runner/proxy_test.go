package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reloader struct {
	subCh    chan struct{}
	reloadCh chan StreamMessage
}

func (r *reloader) AddSubscriber() *Subscriber {
	r.subCh <- struct{}{}
	return &Subscriber{msgCh: r.reloadCh}
}

func (r *reloader) RemoveSubscriber(_ int32) {
	close(r.subCh)
}

func (r *reloader) Reload()                    {}
func (r *reloader) BuildFailed(BuildFailedMsg) {}
func (r *reloader) Stop()                      {}

var proxyPort = 8090

func getServerPort(t *testing.T, srv *httptest.Server) int {
	mockURL, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(mockURL.Port())
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func TestProxy_run(t *testing.T) {
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
	go func() {
		proxy.Run()
	}()
	if err := proxy.Stop(); err != nil {
		t.Errorf("failed stopping the proxy: %v", err)
	}
}

func TestProxy_proxyHandler(t *testing.T) {
	tests := []struct {
		name   string
		req    func() *http.Request
		assert func(*http.Request)
	}{
		{
			name: "get_request_with_headers",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", fmt.Sprintf("http://localhost:%d", proxyPort), nil)
				req.Header.Set("foo", "bar")
				return req
			},
			assert: func(resp *http.Request) {
				assert.Equal(t, "bar", resp.Header.Get("foo"))
			},
		},
		{
			name: "post_form_request",
			req: func() *http.Request {
				formData := url.Values{}
				formData.Add("foo", "bar")
				req := httptest.NewRequest("POST", fmt.Sprintf("http://localhost:%d", proxyPort), strings.NewReader(formData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			assert: func(resp *http.Request) {
				require.NoError(t, resp.ParseForm())
				assert.Equal(t, "bar", resp.Form.Get("foo"))
			},
		},
		{
			name: "get_request_with_query_string",
			req: func() *http.Request {
				return httptest.NewRequest("GET", fmt.Sprintf("http://localhost:%d?q=%s", proxyPort, "air"), nil)
			},
			assert: func(resp *http.Request) {
				q := resp.URL.Query()
				assert.Equal(t, "q=air", q.Encode())
			},
		},
		{
			name: "put_json_request",
			req: func() *http.Request {
				body := []byte(`{"foo": "bar"}`)
				req := httptest.NewRequest("PUT", fmt.Sprintf("http://localhost:%d/a/b/c", proxyPort), bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json; charset=UTF-8")
				return req
			},
			assert: func(resp *http.Request) {
				type Response struct {
					Foo string `json:"foo"`
				}
				var r Response
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&r))
				assert.Equal(t, "/a/b/c", resp.URL.Path)
				assert.Equal(t, "bar", r.Foo)
			},
		},
		{
			name: "set_via_header",
			req: func() *http.Request {
				req := httptest.NewRequest("GET", fmt.Sprintf("http://localhost:%d", proxyPort), nil)
				return req
			},
			assert: func(resp *http.Request) {
				assert.Equal(t, fmt.Sprintf("HTTP/1.1 localhost:%d", proxyPort), resp.Header.Get("Via"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				tt.assert(r)
			}))
			defer srv.Close()
			srvPort := getServerPort(t, srv)
			proxy := NewProxy(&cfgProxy{
				Enabled:   true,
				ProxyPort: proxyPort,
				AppPort:   srvPort,
			})
			proxy.proxyHandler(httptest.NewRecorder(), tt.req())
		})
	}
}

func TestProxy_injectLiveReload(t *testing.T) {
	tests := []struct {
		name   string
		given  *http.Response
		expect string
	}{
		{
			name: "when_no_body_should_not_be_injected",
			given: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
			},
			expect: "",
		},
		{
			name: "when_missing_body_should_not_be_injected",
			given: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/html"},
				},
				Body: io.NopCloser(strings.NewReader(`<h1>test</h1>`)),
			},
			expect: "<h1>test</h1>",
		},
		{
			name: "when_text_html_and_body_is_present_should_be_injected",
			given: &http.Response{
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/html"},
				},
				Body: io.NopCloser(strings.NewReader(`<body><h1>test</h1></body>`)),
			},
			expect: fmt.Sprintf(`<body><h1>test</h1><script>%s</script></body>`, ProxyScript),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := NewProxy(&cfgProxy{
				Enabled:   true,
				ProxyPort: 1111,
				AppPort:   2222,
			})
			got, _ := proxy.injectLiveReload(tt.given)
			if got != tt.expect {
				// Use a more descriptive error message
				if len(got) > 100 || len(tt.expect) > 100 {
					t.Errorf("Script injection mismatch.\nGot length: %d\nExpected length: %d",
						len(got), len(tt.expect))
				} else {
					t.Errorf("expected page %+v, got %v", tt.expect, got)
				}
			}
		})
	}
}

func TestProxy_reloadHandler(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "thin air")
	}))
	srvPort := getServerPort(t, srv)
	defer srv.Close()

	reloader := &reloader{subCh: make(chan struct{}), reloadCh: make(chan StreamMessage)}
	cfg := &cfgProxy{
		Enabled:   true,
		ProxyPort: proxyPort,
		AppPort:   srvPort,
	}
	proxy := &Proxy{
		config: cfg,
		server: &http.Server{
			Addr: fmt.Sprintf("localhost:%d", proxyPort),
		},
		stream: reloader,
	}

	req := httptest.NewRequest("GET", fmt.Sprintf("http://localhost:%d/internal/reload", proxyPort), nil)
	rec := httptest.NewRecorder()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy.reloadHandler(rec, req)
	}()

	<-reloader.subCh

	reloader.reloadCh <- StreamMessage{
		Type: StreamMessageReload,
		Data: nil,
	}
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

	expected := "event: reload\ndata: null\n\n"
	if got := string(bodyBytes); got != expected {
		t.Errorf("expected %q but got %q", expected, got)
	}

	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin": "*",
		"Content-Type":                "text/event-stream",
		"Cache-Control":               "no-cache",
		"Connection":                  "keep-alive",
	}

	for key, value := range expectedHeaders {
		if got := resp.Header.Get(key); got != value {
			t.Errorf("expected header %s to be %q but got %q", key, value, got)
		}
	}
}
