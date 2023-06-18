package pkg

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
	// TODO implement me
	panic("implement me")
}
