package unit_test

import (
	"testing"
	"time"

	"github.com/mant7s/qps-counter/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg, err := config.Load("../../config/config.yaml")
		assert.NoError(t, err)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, 1*time.Second, cfg.Counter.WindowSize)
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := config.Load("invalid_path.yaml")
		assert.Error(t, err)
	})
}
