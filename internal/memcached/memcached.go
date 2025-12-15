package memcached

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/bradfitz/gomemcache/memcache"
)

var ErrCacheMiss = memcache.ErrCacheMiss

type Cache struct {
	client *memcache.Client
	ttl    time.Duration
	prefix string
	enable bool
}

type CacheInterface interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Increment(ctx context.Context, key string, value uint64) (uint64, error)
	Decrement(ctx context.Context, key string, value uint64) (uint64, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

func NewCache(ctx context.Context, cfg config.Config) (*Cache, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !cfg.Memcached.Enable {
		return &Cache{enable: false}, nil
	} else {
		client := memcache.New(cfg.Memcached.Servers...)

		if err := client.Ping(); err != nil {
			return nil, err
		}

		return &Cache{
			client: client,
			ttl:    time.Duration(cfg.Memcached.DefaultTTL) * time.Second,
			prefix: cfg.Memcached.KeyPrefix,
			enable: true,
		}, nil
	}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !c.enable || c.client == nil {
		return nil, memcache.ErrCacheMiss
	}
	prefix := c.prefix + ":" + key
	item, err := c.client.Get(prefix)
	if err != nil {
		return nil, err
	}
	return item.Value, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !c.enable || c.client == nil {
		return memcache.ErrCacheMiss
	}
	prefix := c.prefix + ":" + key
	err := c.client.Set(&memcache.Item{
		Key:        prefix,
		Value:      value,
		Expiration: int32(ttl.Seconds()),
	})
	return err
}

func (c *Cache) Increment(ctx context.Context, key string, value uint64) (newValue uint64, err error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if !c.enable || c.client == nil {
		return 0, memcache.ErrCacheMiss
	}
	prefix := c.prefix + ":" + key

	newValue, err = c.client.Increment(prefix, value)
	if err != nil {
		// Если ключа нет - создаем его с начальным значением
		if errors.Is(err, memcache.ErrCacheMiss) {
			initial := value
			err = c.client.Set(&memcache.Item{
				Key:        prefix,
				Value:      []byte(strconv.FormatUint(initial, 10)),
				Expiration: int32(c.ttl.Seconds()),
			})
			if err != nil {
				return 0, err
			}
			return initial, nil
		}
		return 0, err
	}
	return newValue, nil
}

func (c *Cache) Decrement(ctx context.Context, key string, value uint64) (newValue uint64, err error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if !c.enable || c.client == nil {
		return 0, memcache.ErrCacheMiss
	}
	prefix := c.prefix + ":" + key

	newValue, err = c.client.Decrement(prefix, value)
	if err != nil {
		// Если ключа нет - создаем его с нулевым значением
		if errors.Is(err, memcache.ErrCacheMiss) {
			err = c.client.Set(&memcache.Item{
				Key:        prefix,
				Value:      []byte("0"),
				Expiration: int32(c.ttl.Seconds()),
			})
			if err != nil {
				return 0, err
			}
			return 0, nil
		}
		return 0, err
	}
	return newValue, nil
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !c.enable || c.client == nil {
		return nil
	}

	prefix := c.prefix + ":" + key
	return c.client.Delete(prefix)
}

var ErrCacheDisabled = errors.New("memcached caching is disabled")

// Проверки
// Ping проверяет доступность memcached
func (c *Cache) Ping() error {
	if c.client == nil {
		return ErrCacheDisabled
	}
	return c.client.Ping()
}

// IsEnabled проверяет, активирован ли кэш
func (c *Cache) IsEnabled() error {
	if !c.enable {
		return ErrCacheDisabled
	}
	return nil
}
