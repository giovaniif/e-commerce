package loki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Writer buffers log lines and sends them to Loki's push API.
type Writer struct {
	url    string
	job    string
	client *http.Client
	mu     sync.Mutex
	buf    []lokiEntry
	ticker *time.Ticker
	done   chan struct{}
}

type lokiEntry struct {
	ts  string
	line string
}

// NewWriter returns a Writer that sends logs to the given Loki push URL (e.g. http://loki:3100).
// job is the stream label (e.g. "order", "payment", "stock"). If url is empty, returns nil.
func NewWriter(url, job string) *Writer {
	if url == "" || job == "" {
		return nil
	}
	u := strings.TrimSuffix(url, "/") + "/loki/api/v1/push"
	w := &Writer{
		url:    u,
		job:    job,
		client: &http.Client{Timeout: 5 * time.Second},
		buf:    make([]lokiEntry, 0, 64),
		ticker: time.NewTicker(1 * time.Second),
		done:   make(chan struct{}),
	}
	go w.flushLoop()
	return w
}

// Write implements io.Writer. Each line (newline-separated) is buffered and sent to Loki.
func (w *Writer) Write(p []byte) (n int, err error) {
	n = len(p)
	lines := bytes.Split(p, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		w.mu.Lock()
		w.buf = append(w.buf, lokiEntry{
			ts:  fmt.Sprintf("%d", time.Now().UnixNano()),
			line: string(line),
		})
		needFlush := len(w.buf) >= 20
		w.mu.Unlock()
		if needFlush {
			w.flush()
		}
	}
	return n, nil
}

func (w *Writer) flushLoop() {
	for {
		select {
		case <-w.done:
			return
		case <-w.ticker.C:
			w.flush()
		}
	}
}

func (w *Writer) flush() {
	w.mu.Lock()
	if len(w.buf) == 0 {
		w.mu.Unlock()
		return
	}
	entries := w.buf
	w.buf = make([]lokiEntry, 0, 64)
	w.mu.Unlock()

	values := make([][]string, len(entries))
	for i, e := range entries {
		values[i] = []string{e.ts, e.line}
	}
	body := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": map[string]string{"job": w.job},
				"values": values,
			},
		},
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(raw))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// Close flushes remaining buffer and stops the background flusher.
func (w *Writer) Close() error {
	w.ticker.Stop()
	close(w.done)
	w.flush()
	return nil
}
