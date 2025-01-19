package config

type Config struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	NATSURL  string `mapstructure:"nats_url"`
}
