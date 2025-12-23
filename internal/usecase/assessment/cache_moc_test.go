package assessment

import (
	"context"
	"time"

	"github.com/Caritas-Team/reviewer/internal/memcached"
)

type memCacheMock struct{ m map[string][]byte }

func newMemCacheMock() *memCacheMock { return &memCacheMock{m: map[string][]byte{}} }

func (c *memCacheMock) Get(ctx context.Context, key string) ([]byte, error) {
	if v, ok := c.m[key]; ok {
		return v, nil
	}
	return nil, memcached.ErrCacheMiss
}
func (c *memCacheMock) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.m[key] = value
	return nil
}
func (c *memCacheMock) Increment(ctx context.Context, key string, value uint64) (uint64, error) {
	return 0, memcached.ErrCacheMiss
}
func (c *memCacheMock) Decrement(ctx context.Context, key string, value uint64) (uint64, error) {
	return 0, memcached.ErrCacheMiss
}
func (c *memCacheMock) Delete(ctx context.Context, key string) error { delete(c.m, key); return nil }
func (c *memCacheMock) Close() error                                 { return nil }
func (c *memCacheMock) Ping() error                                  { return nil }
func (c *memCacheMock) IsEnabled() error                             { return nil }
