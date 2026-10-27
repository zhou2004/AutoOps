package kubeasz

import (
	"bytes"
	"io"
	"regexp"
	"strings"
	"sync"
)

// Ansible is run with no_log=True. This second layer prevents credentials in
// shell diagnostics or split output chunks from reaching the persistent log.
var secretLine = regexp.MustCompile(`(?i)(password|passwd|token|secret|client-key-data|authorization)\s*[=:]`)
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

type logWriter struct {
	out     io.Writer
	mu      sync.Mutex
	pending []byte
	private bool
	written int
	capped  bool
}

func newLogWriter(out io.Writer) *logWriter { return &logWriter{out: out} }
func (w *logWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = append(w.pending, data...)
	for {
		i := bytes.IndexByte(w.pending, '\n')
		if i < 0 {
			break
		}
		w.line(string(w.pending[:i]))
		w.pending = w.pending[i+1:]
	}
	// Bound memory even if a subprocess emits an enormous line without newlines.
	if len(w.pending) > 64<<10 {
		w.pending = nil
		w.line("[省略超长日志行]")
	}
	return len(data), nil
}
func (w *logWriter) line(line string) {
	if w.capped {
		return
	}
	if w.written > 20<<20 {
		io.WriteString(w.out, "[日志达到 20 MiB 上限，后续输出已省略]\n")
		w.capped = true
		return
	}
	line = ansiEscape.ReplaceAllString(line, "")
	if strings.Contains(line, "-----BEGIN") && strings.Contains(line, "PRIVATE KEY-----") {
		w.private = true
	}
	if w.private {
		if strings.Contains(line, "-----END") && strings.Contains(line, "PRIVATE KEY-----") {
			w.private = false
		}
		line = "[私钥内容已隐藏]"
	} else if secretLine.MatchString(line) {
		line = "[含敏感信息的日志行已隐藏]"
	}
	n, _ := io.WriteString(w.out, line+"\n")
	w.written += n
}
func (w *logWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) > 0 {
		w.line(string(w.pending))
		w.pending = nil
	}
}
