package httpprobe

import "sync"

type HttpProbeStatus struct {
	status           string     // 记录当前状态
	err              error      // 记录最后一次发生的错误
	activeGoroutines int        // 当前活跃的 goroutine 数量
	mu               sync.Mutex // 用于保护状态字段的并发访问
}

func (h *HttpProbeStatus) setStatus(status string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
}

func (h *HttpProbeStatus) getStatus() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.status
}

func (h *HttpProbeStatus) setError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

func (h *HttpProbeStatus) getError() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

func (h *HttpProbeStatus) incrementGoroutines() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeGoroutines++
}

func (h *HttpProbeStatus) decrementGoroutines() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeGoroutines--
}

func (h *HttpProbeStatus) getActiveGoroutines() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.activeGoroutines
}
