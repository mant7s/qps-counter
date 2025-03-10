package config

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"sync"
	"time"
)

var (
	once   sync.Once
	config *AppConfig
)

type AppConfig struct {
	Server  ServerConfig  `mapstructure:"server"`
	Counter CounterConfig `mapstructure:"counter"`
	Logger  LoggerConfig  `mapstructure:"logger"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type CounterConfig struct {
	Type       string        `mapstructure:"type"`
	WindowSize time.Duration `mapstructure:"window_size"`
	SlotNum    int           `mapstructure:"slot_num"`
	Precision  time.Duration `mapstructure:"precision"`
}

type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

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

	v.AutomaticEnv()
	v.SetEnvPrefix("QPS")

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
	if cfg.Counter.WindowSize <= 0 {
		return fmt.Errorf("invalid counter config window_size")
	}

	if cfg.Counter.SlotNum <= 0 {
		return fmt.Errorf("invalid counter config slot_num")
	}

	if cfg.Counter.Precision <= 0 {
		return fmt.Errorf("invalid counter config precision")
	}

	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port")
	}

	return nil
}
