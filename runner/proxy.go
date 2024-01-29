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
	RemoveSubscriber(id int)
	Reload()
	Stop()
}

type Proxy struct {
	server *http.Server
	config *cfgProxy
	stream Reloader
}

func NewProxy(cfg *cfgProxy) *Proxy {
	p := &Proxy{
		config: cfg,
		server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.ProxyPort),
		},
		stream: NewProxyStream(),
	}
	return p
}

func (p *Proxy) Run() {
	http.HandleFunc("/", p.proxyHandler)
	http.HandleFunc("/internal/reload", p.reloadHandler)
	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func (p *Proxy) Stop() {
	p.server.Close()
	p.stream.Stop()
}

func (p *Proxy) Reload() {
	p.stream.Reload()
}

func (p *Proxy) injectLiveReload(origURL string, respBody io.ReadCloser) string {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(respBody); err != nil {
		log.Fatalf("failed to convert request body to bytes buffer, err: %+v\n", err)
	}
	original := buf.String()

	// the script will be injected before the end of the body tag. In case the tag is missing, the injection will be skipped without an error to ensure that a page with partial reloads only has at most one injected script.
	body := strings.LastIndex(original, "</body>")
	if body == -1 {
		return original
	}

	script := `
	<script>
	const parser = new DOMParser();
	const proxyURL = "http://localhost:%d";
	new EventSource(proxyURL + "/internal/reload").onmessage = () => {
		fetch(proxyURL + "%s").then(res => res.text()).then(resStr => {
			const newPage = parser.parseFromString(resStr, "text/html");
			document.replaceChild(newPage.documentElement, document.documentElement);
		});
	};
	</script>
	`
	parsedScript := fmt.Sprintf(script, p.config.ProxyPort, origURL)
	return original[:body] + parsedScript + original[body:]
}

func (p *Proxy) proxyHandler(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("http://localhost:%d%s", p.config.AppPort, r.URL.Path)
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		log.Fatalf("proxy could not create request, err: %+v\n", err)
	}
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)

	client := &http.Client{}
	var resp *http.Response
	for i := 0; i < 10; i++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			log.Fatalf("proxy failed to call %s, err: %+v\n", url, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer resp.Body.Close()

	// copy all headers except Content-Length
	for k, vv := range resp.Header {
		for _, v := range vv {
			if k == "Content-Length" {
				continue
			}
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		s := p.injectLiveReload(r.URL.String(), resp.Body)
		w.Header().Set("Content-Length", strconv.Itoa((len([]byte(s)))))
		if _, err := io.WriteString(w, s); err != nil {
			log.Fatalf("proxy failed injected live reloading script, err: %+v\n", err)
		}
	} else {
		w.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Fatalf("proxy failed to forward request, err: %+v\n", err)
		}
	}
}

func (p *Proxy) reloadHandler(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

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
