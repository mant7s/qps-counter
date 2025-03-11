package config

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	once   sync.Once
	config *AppConfig
)

// AppConfig 应用配置结构体
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server" env:"SERVER"`
	Counter  CounterConfig  `mapstructure:"counter" env:"COUNTER"`
	Logger   LoggerConfig   `mapstructure:"logger" env:"LOGGER"`
	Limiter  LimiterConfig  `mapstructure:"limiter" env:"LIMITER"`
	Metrics  MetricsConfig  `mapstructure:"metrics" env:"METRICS"`
	Shutdown ShutdownConfig `mapstructure:"shutdown" env:"SHUTDOWN"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int           `mapstructure:"port" env:"PORT"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" env:"READ_TIMEOUT"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" env:"WRITE_TIMEOUT"`
}

// CounterConfig 计数器配置
type CounterConfig struct {
	Type       string        `mapstructure:"type" env:"TYPE"`
	WindowSize time.Duration `mapstructure:"window_size" env:"WINDOW_SIZE"`
	SlotNum    int           `mapstructure:"slot_num" env:"SLOT_NUM"`
	Precision  time.Duration `mapstructure:"precision" env:"PRECISION"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level" env:"LEVEL"`
	Format     string `mapstructure:"format" env:"FORMAT"`
	FilePath   string `mapstructure:"file_path" env:"FILE_PATH"`
	MaxSize    int    `mapstructure:"max_size" env:"MAX_SIZE"`
	MaxBackups int    `mapstructure:"max_backups" env:"MAX_BACKUPS"`
	MaxAge     int    `mapstructure:"max_age" env:"MAX_AGE"`
}

// LimiterConfig 限流器配置
type LimiterConfig struct {
	Enabled  bool  `mapstructure:"enabled" env:"ENABLED"`
	Rate     int64 `mapstructure:"rate" env:"RATE"`
	Burst    int64 `mapstructure:"burst" env:"BURST"`
	Adaptive bool  `mapstructure:"adaptive" env:"ADAPTIVE"`
}

// MetricsConfig 指标收集配置
type MetricsConfig struct {
	Enabled  bool          `mapstructure:"enabled" env:"ENABLED"`
	Interval time.Duration `mapstructure:"interval" env:"INTERVAL"`
	Endpoint string        `mapstructure:"endpoint" env:"ENDPOINT"`
}

// ShutdownConfig 优雅关闭配置
type ShutdownConfig struct {
	Timeout time.Duration `mapstructure:"timeout" env:"TIMEOUT"`
	MaxWait time.Duration `mapstructure:"max_wait" env:"MAX_WAIT"`
}

// Load 加载配置
// 支持从配置文件和环境变量加载配置
// 环境变量前缀为QPS，例如：QPS_SERVER_PORT
func Load(configPath string) (*AppConfig, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/qps-counter")

	if configPath != "" {
		v.SetConfigFile(configPath)
	}

	// 设置环境变量前缀并自动绑定环境变量
	v.AutomaticEnv()
	v.SetEnvPrefix("QPS")

	// 设置环境变量分隔符为下划线
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 绑定环境变量
	// 服务器配置
	v.BindEnv("server.port", "QPS_SERVER_PORT")
	v.BindEnv("server.read_timeout", "QPS_SERVER_READ_TIMEOUT")
	v.BindEnv("server.write_timeout", "QPS_SERVER_WRITE_TIMEOUT")

	// 计数器配置
	v.BindEnv("counter.type", "QPS_COUNTER_TYPE")
	v.BindEnv("counter.window_size", "QPS_COUNTER_WINDOW_SIZE")
	v.BindEnv("counter.slot_num", "QPS_COUNTER_SLOT_NUM")
	v.BindEnv("counter.precision", "QPS_COUNTER_PRECISION")

	// 日志配置
	v.BindEnv("logger.level", "QPS_LOGGER_LEVEL")
	v.BindEnv("logger.format", "QPS_LOGGER_FORMAT")
	v.BindEnv("logger.file_path", "QPS_LOGGER_FILE_PATH")
	v.BindEnv("logger.max_size", "QPS_LOGGER_MAX_SIZE")
	v.BindEnv("logger.max_backups", "QPS_LOGGER_MAX_BACKUPS")
	v.BindEnv("logger.max_age", "QPS_LOGGER_MAX_AGE")

	// 限流器配置
	v.BindEnv("limiter.enabled", "QPS_LIMITER_ENABLED")
	v.BindEnv("limiter.rate", "QPS_LIMITER_RATE")
	v.BindEnv("limiter.burst", "QPS_LIMITER_BURST")
	v.BindEnv("limiter.adaptive", "QPS_LIMITER_ADAPTIVE")

	// 指标收集配置
	v.BindEnv("metrics.enabled", "QPS_METRICS_ENABLED")
	v.BindEnv("metrics.interval", "QPS_METRICS_INTERVAL")
	v.BindEnv("metrics.endpoint", "QPS_METRICS_ENDPOINT")

	// 优雅关闭配置
	v.BindEnv("shutdown.timeout", "QPS_SHUTDOWN_TIMEOUT")
	v.BindEnv("shutdown.max_wait", "QPS_SHUTDOWN_MAX_WAIT")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("config file changed:", e.Name)
	})

	return &cfg, nil
}

func validateConfig(cfg *AppConfig) error {
	// 验证计数器配置
	if cfg.Counter.WindowSize <= 0 {
		return fmt.Errorf("invalid counter config window_size")
	}

	if cfg.Counter.SlotNum <= 0 {
		return fmt.Errorf("invalid counter config slot_num")
	}

	if cfg.Counter.Precision <= 0 {
		return fmt.Errorf("invalid counter config precision")
	}

	// 验证服务器配置
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port")
	}

	// 验证限流器配置
	if cfg.Limiter.Enabled && cfg.Limiter.Rate <= 0 {
		return fmt.Errorf("invalid limiter rate")
	}

	if cfg.Limiter.Enabled && cfg.Limiter.Burst <= 0 {
		return fmt.Errorf("invalid limiter burst")
	}

	// 验证指标收集配置
	if cfg.Metrics.Enabled && cfg.Metrics.Interval <= 0 {
		return fmt.Errorf("invalid metrics interval")
	}

	// 验证优雅关闭配置
	if cfg.Shutdown.Timeout <= 0 {
		return fmt.Errorf("invalid shutdown timeout")
	}

	if cfg.Shutdown.MaxWait <= 0 {
		return fmt.Errorf("invalid shutdown max wait")
	}

	return nil
}
