package runner

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingStreamer records the calls Proxy forwards to its Streamer.
type recordingStreamer struct {
	reloads      int
	buildFailure BuildFailedMsg
	stopped      bool
}

func (s *recordingStreamer) AddSubscriber() *Subscriber { return &Subscriber{} }
func (s *recordingStreamer) RemoveSubscriber(int32)     {}
func (s *recordingStreamer) Reload()                    { s.reloads++ }
func (s *recordingStreamer) BuildFailed(m BuildFailedMsg) {
	s.buildFailure = m
}
func (s *recordingStreamer) Stop() { s.stopped = true }

// nonFlusherRecorder is an http.ResponseWriter that deliberately does not
// implement http.Flusher, so handlers take their "streaming unsupported" path.
type nonFlusherRecorder struct {
	header http.Header
	body   strings.Builder
	code   int
}

func newNonFlusherRecorder() *nonFlusherRecorder {
	return &nonFlusherRecorder{header: make(http.Header), code: http.StatusOK}
}

func (w *nonFlusherRecorder) Header() http.Header { return w.header }

func (w *nonFlusherRecorder) Write(b []byte) (int, error) { return w.body.Write(b) }

func (w *nonFlusherRecorder) WriteHeader(code int) { w.code = code }

func TestProxy_workerScriptHandler(t *testing.T) {
	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: 1112})

	rec := httptest.NewRecorder()
	proxy.workerScriptHandler(rec, httptest.NewRequest(http.MethodGet, "/__air_internal/worker.js", nil))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/javascript", rec.Header().Get("Content-Type"))
	assert.Equal(t, WorkerScript, rec.Body.String())
}

func TestProxy_ReloadAndBuildFailedDelegateToStream(t *testing.T) {
	stream := &recordingStreamer{}
	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: 1112})
	proxy.stream = stream

	proxy.Reload()
	proxy.Reload()
	assert.Equal(t, 2, stream.reloads)

	msg := BuildFailedMsg{Error: "boom"}
	proxy.BuildFailed(msg)
	assert.Equal(t, msg, stream.buildFailure)
}

func TestProxy_reloadHandlerWithoutFlusher(t *testing.T) {
	stream := &recordingStreamer{}
	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: 1112})
	proxy.stream = stream

	w := newNonFlusherRecorder()
	proxy.reloadHandler(w, httptest.NewRequest(http.MethodGet, "/__air_internal/sse", nil))

	assert.Equal(t, http.StatusInternalServerError, w.code)
	assert.Contains(t, w.body.String(), "streaming unsupported")
}

func TestProxy_proxyHandlerStreamingWithoutFlusher(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: hello\n\n")
	}))
	defer srv.Close()

	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: getServerPort(t, srv)})

	w := newNonFlusherRecorder()
	proxy.proxyHandler(w, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusInternalServerError, w.code)
	assert.Contains(t, w.body.String(), "streaming not supported")
}

func TestProxy_proxyHandlerBadForm(t *testing.T) {
	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: 1112})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("%zz=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	proxy.proxyHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "bad form")
}

func TestProxy_proxyHandlerInvalidMethod(t *testing.T) {
	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: 1112})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// A method with a space is rejected by http.NewRequest.
	req.Method = "BAD METHOD"

	rec := httptest.NewRecorder()
	proxy.proxyHandler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "unable to create request")
}

// errWriter fails every write, exercising streamCopy's error handling.
type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

// shortWriter reports fewer written bytes than it was given.
type shortWriter struct{}

func (shortWriter) Write(b []byte) (int, error) { return len(b) - 1, nil }

type noopFlusher struct{}

func (noopFlusher) Flush() {}

func TestStreamCopyWriteError(t *testing.T) {
	wantErr := errors.New("write failed")

	err := streamCopy(errWriter{err: wantErr}, strings.NewReader("hello"), noopFlusher{})
	require.Error(t, err)
	assert.Equal(t, wantErr, err)
}

func TestStreamCopyShortWrite(t *testing.T) {
	err := streamCopy(shortWriter{}, strings.NewReader("hello"), noopFlusher{})
	assert.ErrorIs(t, err, io.ErrShortWrite)
}

func TestProxy_Stop(t *testing.T) {
	stream := &recordingStreamer{}
	proxy := NewProxy(&cfgProxy{Enabled: true, ProxyPort: 1111, AppPort: 1112})
	proxy.stream = stream

	require.NoError(t, proxy.Stop())
	assert.True(t, stream.stopped)
}
