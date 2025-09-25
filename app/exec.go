package app

import (
	"io"
	"sync"
)

type multiSliceWriter struct {
	mu   *sync.Mutex
	a, b *[]byte
}

var _ io.Writer = &multiSliceWriter{}

func (t *multiSliceWriter) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	*t.a = append(*t.a, p...)
	*t.b = append(*t.b, p...)

	return len(p), nil
}
