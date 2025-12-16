package user

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Caritas-Team/reviewer/internal/config"
	"github.com/bradfitz/gomemcache/memcache"
)

// Мок для memcached
type mockCache struct {
	storage    map[string][]byte
	alwaysFail bool
}

func newMockCache() *mockCache {
	return &mockCache{
		storage: make(map[string][]byte),
	}
}

func newBrokenCache() *mockCache {
	return &mockCache{
		storage:    make(map[string][]byte),
		alwaysFail: true,
	}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if m.alwaysFail {
		return nil, errors.New("cache broken")
	}

	value, exists := m.storage[key]
	if !exists {
		return nil, memcache.ErrCacheMiss
	}
	return value, nil
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if m.alwaysFail {
		return errors.New("cache broken")
	}

	m.storage[key] = value
	return nil
}

func (m *mockCache) Increment(ctx context.Context, key string, value uint64) (uint64, error) {
	if m.alwaysFail {
		return 0, errors.New("cache broken")
	}

	current, exists := m.storage[key]
	if !exists {
		newValue := value
		m.storage[key] = []byte(strconv.FormatUint(newValue, 10))
		return newValue, nil
	}

	currentInt, err := strconv.ParseUint(string(current), 10, 64)
	if err != nil {
		return 0, err
	}

	newValue := currentInt + value
	m.storage[key] = []byte(strconv.FormatUint(newValue, 10))
	return newValue, nil
}

func (m *mockCache) Decrement(ctx context.Context, key string, value uint64) (uint64, error) {
	if m.alwaysFail {
		return 0, errors.New("cache broken")
	}

	current, exists := m.storage[key]
	if !exists {
		m.storage[key] = []byte("0")
		return 0, nil
	}

	currentInt, err := strconv.ParseUint(string(current), 10, 64)
	if err != nil {
		return 0, err
	}

	var newValue uint64
	if value > currentInt {
		newValue = 0
	} else {
		newValue = currentInt - value
	}

	m.storage[key] = []byte(strconv.FormatUint(newValue, 10))
	return newValue, nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	if m.alwaysFail {
		return errors.New("cache broken")
	}
	delete(m.storage, key)
	return nil
}

<<<<<<< HEAD
=======
func (m *mockCache) Ping() error {
	if m.alwaysFail {
		return errors.New("cache broken")
	}
	return nil
}

func (m *mockCache) IsEnabled() error {
	if m.alwaysFail {
		return errors.New("cache broken")
	}
	return nil
}

>>>>>>> origin/feature/NS-49-assessments-get-results
func (m *mockCache) Close() error {
	return nil
}

func TestRateLimiter_AllowRequest(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockCache()

	cfg := config.Config{
		RateLimiter: config.RateLimiter{
			Enabled:           true,
			RequestsPerWindow: 1,
			WindowSize:        30,
			Storage:           "memcached",
		},
	}

	limiter := NewRateLimiter(mockCache, cfg)

	t.Run("первый запрос разрешен", func(t *testing.T) {
		err := limiter.AllowRequest(ctx, "user1")
		if err != nil {
			t.Errorf("ожидался nil, получил %v", err)
		}
	})

	t.Run("второй запрос запрещен", func(t *testing.T) {
		err := limiter.AllowRequest(ctx, "user1")
		if !errors.Is(err, ErrRateLimitExceeded) {
			t.Errorf("ожидалась ошибка лимита, получил %v", err)
		}
	})

	t.Run("разные пользователи - разные лимиты", func(t *testing.T) {
		err := limiter.AllowRequest(ctx, "user1")
		if !errors.Is(err, ErrRateLimitExceeded) {
			t.Errorf("user1 должен быть заблокирован")
		}

		err = limiter.AllowRequest(ctx, "user2")
		if err != nil {
			t.Errorf("user2 должен быть разрешен")
		}
	})
}

func TestRateLimiter_30RequestsPerMinute(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockCache()

	cfg := config.Config{
		RateLimiter: config.RateLimiter{
			Enabled:           true,
			RequestsPerWindow: 30,
			WindowSize:        60,
			Storage:           "memcached",
		},
	}

	limiter := NewRateLimiter(mockCache, cfg)

	for i := 0; i < 30; i++ {
		err := limiter.AllowRequest(ctx, "user1")
		if err != nil {
			t.Errorf("запрос %d должен был пройти, но получил ошибку: %v", i+1, err)
		}
	}

	err := limiter.AllowRequest(ctx, "user1")
	if !errors.Is(err, ErrRateLimitExceeded) {
		t.Errorf("31-й запрос должен был упасть с ошибкой лимита")
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	ctx := context.Background()
	mockCache := newMockCache()

	cfg := config.Config{
		RateLimiter: config.RateLimiter{
			Enabled: false,
		},
	}

	limiter := NewRateLimiter(mockCache, cfg)

	for i := 0; i < 10; i++ {
		err := limiter.AllowRequest(ctx, "user1")
		if err != nil {
			t.Errorf("при выключенном лимитере ошибок быть не должно")
		}
	}
}

func TestRateLimiter_CacheError(t *testing.T) {
	ctx := context.Background()
	brokenCache := newBrokenCache()

	cfg := config.Config{
		RateLimiter: config.RateLimiter{
			Enabled: true,
		},
	}

	limiter := NewRateLimiter(brokenCache, cfg)

	err := limiter.AllowRequest(ctx, "user1")
	if err != nil {
		t.Errorf("при ошибках кэша должен разрешать запрос")
	}
}
