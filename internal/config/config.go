package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Server struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Debug           bool   `mapstructure:"debug"`
	LogLevel        string `mapstructure:"log_level"`
	ReadTimeoutSec  int    `mapstructure:"read_timeout"`
	WriteTimeoutSec int    `mapstructure:"write_timeout"`
}

func (s Server) Addr() string                { return fmt.Sprintf("%s:%d", s.Host, s.Port) }
func (s Server) ReadTimeout() time.Duration  { return time.Duration(s.ReadTimeoutSec) * time.Second }
func (s Server) WriteTimeout() time.Duration { return time.Duration(s.WriteTimeoutSec) * time.Second }

type CORS struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type RateLimiter struct {
	Enabled           bool   `mapstructure:"enabled"`
	RequestsPerWindow int    `mapstructure:"requests_per_window"`
	Storage           string `mapstructure:"storage"`
	WindowSize        int    `mapstructure:"window_size"`
}

type Memcached struct {
	Enable     bool     `mapstructure:"enable"`
	Servers    []string `mapstructure:"servers"`
	DefaultTTL int      `mapstructure:"default_ttl"`
	KeyPrefix  string   `mapstructure:"key_prefix"`
}

type Files struct {
	MaxFilesPerRequest int      `mapstructure:"max_files_per_request"`
	MaxFileSize        int64    `mapstructure:"max_file_size"`
	MaxProcessingTime  int      `mapstructure:"max_processing_time"`
	AllowedMIMETypes   []string `mapstructure:"allowed_mime_types"`
}

type Metrics struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type Logging struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type Jaeger struct {
	Endpoint string `mapstructure:"endpoint"`
}

// PipelineConfig - конфигурация пайплайна
type PipelineConfig struct {
	WorkerPoolSize   int `mapstructure:"worker_pool_size"`   // Количество воркеров
	InputBufferSize  int `mapstructure:"input_buffer_size"`  // Размер буфера входящего канала
	ResultBufferSize int `mapstructure:"result_buffer_size"` // Размер буфера канала результатов
}

type Config struct {
	Server      Server         `mapstructure:"server"`
	CORS        CORS           `mapstructure:"cors"`
	RateLimiter RateLimiter    `mapstructure:"rate_limiter"`
	Memcached   Memcached      `mapstructure:"memcached"`
	Files       Files          `mapstructure:"files"`
	Metrics     Metrics        `mapstructure:"metrics"`
	Logging     Logging        `mapstructure:"logging"`
	Jaeger      Jaeger         `mapstructure:"jaeger"`
	Pipeline    PipelineConfig `mapstructure:"pipeline"`
}

func Load() (Config, error) {
	var cfg Config

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("cfg")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.SetDefault("cors.allowed_origins", []string{"http://localhost:8000"})

	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if s, ok := os.LookupEnv("CORS_ALLOWED_ORIGINS"); ok && strings.TrimSpace(s) != "" {
		parts := strings.Split(s, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			v.Set("cors.allowed_origins", out)
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}
