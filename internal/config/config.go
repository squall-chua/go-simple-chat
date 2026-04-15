package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port               string `mapstructure:"PORT"`
	MongoURI           string `mapstructure:"MONGO_URI"`
	RedisAddr          string `mapstructure:"REDIS_ADDR"`
	BrokerType         string `mapstructure:"BROKER_TYPE"`
	LogLevel           string `mapstructure:"LOG_LEVEL"`
	S3Endpoint         string `mapstructure:"S3_ENDPOINT"`
	S3AccessKey        string `mapstructure:"S3_ACCESS_KEY"`
	S3SecretKey        string `mapstructure:"S3_SECRET_KEY"`
	S3Bucket           string `mapstructure:"S3_BUCKET"`
	S3UseSSL           bool   `mapstructure:"S3_USE_SSL"`
	CertDir            string `mapstructure:"CERT_DIR"`
	MetricsPort        string `mapstructure:"METRICS_PORT"`
	GracefulTimeoutSec int    `mapstructure:"GRACEFUL_TIMEOUT_SEC"`
}

func LoadConfig() *Config {
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("BROKER_TYPE", "local")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("S3_ENDPOINT", "localhost:9000")
	viper.SetDefault("S3_ACCESS_KEY", "minioadmin")
	viper.SetDefault("S3_SECRET_KEY", "minioadmin")
	viper.SetDefault("S3_BUCKET", "chat-media")
	viper.SetDefault("S3_USE_SSL", false)
	viper.SetDefault("CERT_DIR", "certs")
	viper.SetDefault("METRICS_PORT", "9090")
	viper.SetDefault("GRACEFUL_TIMEOUT_SEC", 30)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	return &cfg
}
