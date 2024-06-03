package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"syscall"
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

func (p *Proxy) injectLiveReload(resp *http.Response) (page string, didReadBody bool) {
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return page, false
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return page, false
	}
	page = buf.String()

	// the script will be injected before the end of the head tag. In case the tag is missing, the injection will be skipped.
	injectIdx := strings.LastIndex(page, "</head>")
	if injectIdx == -1 {
		return page, true
	}

	script := fmt.Sprintf(
		`<script>new EventSource("http://localhost:%d/internal/reload").onmessage = () => { location.reload() }</script>`,
		p.config.ProxyPort,
	)
	return page[:injectIdx] + script + page[injectIdx:], true
}

func (p *Proxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	appURL := r.URL
	appURL.Scheme = "http"
	appURL.Host = fmt.Sprintf("localhost:%d", p.config.AppPort)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "proxy handler: bad form", http.StatusInternalServerError)
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
	}

	// Copy the headers from the original request
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)

	// retry on connection refused error since after a file change air will restart the server and it may take a few milliseconds for the server to be up-and-running.
	var resp *http.Response
	for i := 0; i < 10; i++ {
		resp, err = p.client.Do(req)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			http.Error(w, "proxy handler: unable to reach app", http.StatusInternalServerError)
		}
		time.Sleep(100 * time.Millisecond)
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
	w.WriteHeader(resp.StatusCode)

	page, didReadBody := p.injectLiveReload(resp)
	if didReadBody {
		// injectLiveReload() did read the response body, so we have to use 'page', whether it was modified or not
		w.Header().Set("Content-Length", strconv.Itoa(len([]byte(page))))
		if _, err := io.WriteString(w, page); err != nil {
			http.Error(w, "proxy handler: unable to inject live reload script", http.StatusInternalServerError)
		}
	} else {
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
		if _, err := io.Copy(w, resp.Body); err != nil {
			http.Error(w, "proxy handler: failed to forward the response body", http.StatusInternalServerError)
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
