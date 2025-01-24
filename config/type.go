package config

type Config struct {
	Port     int    `mapstructure:"port"`
	LogLevel string `mapstructure:"log_level"`
	LogFile  string `mapstructure:"log_file"`
	NATSURL  string `mapstructure:"nats_url"`
	RedisURL string `mapstructure:"redis_url"`
}
