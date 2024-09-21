package runner

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Reloader interface {
	AddSubscriber() *Subscriber
	RemoveSubscriber(id int32)
	Reload()
	Stop()
}

type Proxy struct {
	server *http.Server
	client *http.Client
	config *cfgProxy
	stream Reloader
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
	http.HandleFunc("/internal/reload", p.reloadHandler)
	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(p.Stop())
	}
}

func (p *Proxy) Reload() {
	p.stream.Reload()
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

	script := `<script>new EventSource("/internal/reload").onmessage = () => { location.reload() }</script>`
	return page[:body] + script + page[body:], nil
}

func (p *Proxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

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

	// air will restart the server. it may take a few milliseconds for it to start back up.
	// therefore, we retry until the server becomes available or this retry loop exits with an error.
	var resp *http.Response
	resp, err = p.client.Do(req)
	for i := 0; i < 10; i++ {
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
		resp, err = p.client.Do(req)
	}
	if err != nil {
		http.Error(w, "proxy handler: unable to reach app", http.StatusInternalServerError)
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
	w.Header().Add("Via", viaHeaderValue)
	w.WriteHeader(resp.StatusCode)

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
		if _, err := io.Copy(w, resp.Body); err != nil {
			http.Error(w, "proxy handler: failed to forward the response body", http.StatusInternalServerError)
			return
		}
	} else {
		page, err := p.injectLiveReload(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa((len([]byte(page)))))
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

	for range sub.reloadCh {
		fmt.Fprintf(w, "data: reload\n\n")
		flusher.Flush()
	}
}

func (p *Proxy) Stop() error {
	p.stream.Stop()
	return p.server.Close()
}
