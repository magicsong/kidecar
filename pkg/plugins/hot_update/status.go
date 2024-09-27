package hot_update

import "sync"

type HotUpdateStatus struct {
	status           string     // 记录当前状态
	err              error      // 记录最后一次发生的错误
	activeGoroutines int        // 当前活跃的 goroutine 数量
	mu               sync.Mutex // 用于保护状态字段的并发访问
}

func (h *HotUpdateStatus) setStatus(status string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = status
}

func (h *HotUpdateStatus) getStatus() string {
	return h.status
}

func (h *HotUpdateStatus) setError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

func (h *HotUpdateStatus) getError() error {
	return h.err
}

func (h *HotUpdateStatus) incrementGoroutines() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeGoroutines++
}

func (h *HotUpdateStatus) decrementGoroutines() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeGoroutines--
}

func (h *HotUpdateStatus) getActiveGoroutines() int {
	return h.activeGoroutines
}
