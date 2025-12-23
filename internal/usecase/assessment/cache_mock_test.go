package assessment

import (
	"context"
	"sync"
	"time"

	"github.com/Caritas-Team/reviewer/internal/memcached"
)

type memCacheMock struct {
	mu sync.RWMutex
	m  map[string][]byte
}

func newMemCacheMock() *memCacheMock {
	return &memCacheMock{m: map[string][]byte{}}
}

func (c *memCacheMock) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	v, ok := c.m[key]
	c.mu.RUnlock()

	if !ok {
		return nil, memcached.ErrCacheMiss
	}

	out := make([]byte, len(v))
	copy(out, v)
	return out, nil
}

func (c *memCacheMock) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	in := make([]byte, len(value))
	copy(in, value)

	c.mu.Lock()
	c.m[key] = in
	c.mu.Unlock()
	return nil
}

func (c *memCacheMock) Increment(ctx context.Context, key string, value uint64) (uint64, error) {
	return 0, memcached.ErrCacheMiss
}

func (c *memCacheMock) Decrement(ctx context.Context, key string, value uint64) (uint64, error) {
	return 0, memcached.ErrCacheMiss
}

func (c *memCacheMock) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.m, key)
	c.mu.Unlock()
	return nil
}

func (c *memCacheMock) Close() error     { return nil }
func (c *memCacheMock) Ping() error      { return nil }
func (c *memCacheMock) IsEnabled() error { return nil }
