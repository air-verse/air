package runner

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed proxy.js
var ProxyScript string

type Streamer interface {
	AddSubscriber() *Subscriber
	RemoveSubscriber(id int32)
	Reload()
	BuildFailed(msg BuildFailedMsg)
	Stop()
}

type Proxy struct {
	server *http.Server
	client *http.Client
	config *cfgProxy
	stream Streamer
}

func NewProxy(cfg *cfgProxy) *Proxy {
	p := &Proxy{
		config: cfg,
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.ProxyPort),
		},
		client: &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		stream: NewProxyStream(),
	}
	return p
}

func (p *Proxy) Run() {
	http.HandleFunc("/", p.proxyHandler)
	http.HandleFunc("/__air_internal/sse", p.reloadHandler)
	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(p.Stop())
	}
}

func (p *Proxy) Reload() {
	p.stream.Reload()
}

func (p *Proxy) BuildFailed(msg BuildFailedMsg) {
	p.stream.BuildFailed(msg)
}

func (p *Proxy) injectLiveReload(resp *http.Response) (string, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("proxy inject: failed to read body from http response")
	}
	page := buf.String()

	// the script will be injected before the end of the body tag. In case the tag is missing, the injection will be skipped with no error.
	body := strings.LastIndex(page, "</body>")
	if body == -1 {
		return page, nil
	}

	script := "<script>" + ProxyScript + "</script>"
	return page[:body] + script + page[body:], nil
}

func (p *Proxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	appURL := r.URL
	appURL.Scheme = "http"
	appURL.Host = fmt.Sprintf("localhost:%d", p.config.AppPort)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "proxy handler: bad form", http.StatusInternalServerError)
		return
	}
	var body io.Reader
	if len(r.Form) > 0 {
		body = strings.NewReader(r.Form.Encode())
	} else {
		body = r.Body
	}
	req, err := http.NewRequest(r.Method, appURL.String(), body)
	if err != nil {
		http.Error(w, "proxy handler: unable to create request", http.StatusInternalServerError)
		return
	}

	// Copy the headers from the original request
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)

	// set the via header
	viaHeaderValue := fmt.Sprintf("%s %s", r.Proto, r.Host)
	req.Header.Set("Via", viaHeaderValue)

	// air will restart the server. it may take a few seconds for it to start back up.
	// therefore, we retry until the server becomes available or this retry loop exits with an error.
	timeout := time.Duration(p.config.AppStartTimeout) * time.Millisecond
	if timeout == 0 {
		timeout = defaultProxyAppStartTimeout * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var resp *http.Response
	resp, err = p.client.Do(req.WithContext(ctx))
	for err != nil {
		// Check if timeout has been exceeded
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		time.Sleep(100 * time.Millisecond)
		resp, err = p.client.Do(req.WithContext(ctx))
	}
	if err != nil {
		http.Error(w, "proxy handler: unable to reach app (try increasing the proxy.app_start_timeout)", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the headers from the proxy response except Content-Length
	for k, vv := range resp.Header {
		for _, v := range vv {
			if k == "Content-Length" {
				continue
			}
			w.Header().Add(k, v)
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Via", viaHeaderValue)

	// Determine if this is a streaming response
	streaming := isStreamingResponse(resp)

	// Handle non-HTML responses
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		// Check flusher support BEFORE writing headers for streaming responses
		var flusher http.Flusher
		if streaming {
			var ok bool
			flusher, ok = w.(http.Flusher)
			if !ok {
				http.Error(w, "proxy handler: streaming not supported", http.StatusInternalServerError)
				return
			}
		}

		// Set Content-Length only for non-streaming responses
		if !streaming {
			if cl := resp.Header.Get("Content-Length"); cl != "" {
				w.Header().Set("Content-Length", cl)
			}
		}

		w.WriteHeader(resp.StatusCode)

		if streaming {
			// Use streaming copy with immediate flushing
			_ = streamCopy(w, resp.Body, flusher)
			return
		}
		// Use standard copy for non-streaming responses
		if _, err := io.Copy(w, resp.Body); err != nil {
			http.Error(w, "proxy handler: failed to forward the response body", http.StatusInternalServerError)
			return
		}
	} else {
		// HTML: inject live reload script
		page, err := p.injectLiveReload(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa((len([]byte(page)))))
		w.WriteHeader(resp.StatusCode)
		if _, err := io.WriteString(w, page); err != nil {
			http.Error(w, "proxy handler: unable to inject live reload script", http.StatusInternalServerError)
			return
		}
	}
}

func (p *Proxy) reloadHandler(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "reload handler: streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sub := p.stream.AddSubscriber()
	go func() {
		<-r.Context().Done()
		p.stream.RemoveSubscriber(sub.id)
	}()

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for msg := range sub.msgCh {
		fmt.Fprint(w, msg.AsSSE())
		flusher.Flush()
	}
}

func (p *Proxy) Stop() error {
	p.stream.Stop()
	return p.server.Close()
}

// isStreamingResponse determines if the response should be streamed immediately
// without buffering. This applies to:
// 1. Server-Sent Events (SSE): Content-Type contains "text/event-stream"
// 2. Chunked transfer encoding: Transfer-Encoding is "chunked"
func isStreamingResponse(resp *http.Response) bool {
	// Check for SSE
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return true
	}

	// Check for chunked encoding
	transferEncoding := resp.Header.Get("Transfer-Encoding")
	return transferEncoding == "chunked"
}

// streamCopy copies data from src to dst, flushing after each read.
// This ensures real-time delivery for streaming responses like SSE.
// Uses a 512-byte buffer to balance between latency and performance.
func streamCopy(dst io.Writer, src io.Reader, flusher http.Flusher) error {
	// Use 512-byte buffer for better responsiveness
	buf := make([]byte, 512)

	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[:nr])
			if writeErr != nil {
				return writeErr
			}
			if nr != nw {
				return io.ErrShortWrite
			}

			// Flush immediately after each write
			flusher.Flush()
		}

		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}
