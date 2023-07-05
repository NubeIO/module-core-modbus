package pkg

import (
	"github.com/NubeIO/module-core-modbus/logger"
	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
	"strings"
)

type Config struct {
	EnablePolling     bool   `yaml:"enable_polling"`
	LogLevel          string `yaml:"log_level"`
	PollQueueLogLevel string `yaml:"poll_queue_log_level"`
}

func (m *Module) DefaultConfig() *Config {
	return &Config{
		EnablePolling:     true,
		LogLevel:          "ERROR",
		PollQueueLogLevel: "ERROR",
	}
}

func (m *Module) GetConfig() interface{} {
	return m.config
}

func (m *Module) ValidateAndSetConfig(config []byte) ([]byte, error) {
	newConfig := m.DefaultConfig()
	_ = yaml.Unmarshal(config, newConfig)

	logLevel, err := log.ParseLevel(newConfig.LogLevel)
	if err != nil {
		logLevel = log.ErrorLevel
	}
	logger.SetLogger(logLevel)
	newConfig.LogLevel = strings.ToUpper(logLevel.String())

	newConfValid, err := yaml.Marshal(newConfig)
	if err != nil {
		return nil, err
	}
	m.config = newConfig

	log.Info("config is set")
	return newConfValid, nil
}
