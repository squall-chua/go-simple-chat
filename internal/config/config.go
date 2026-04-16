package config

import (
	"log"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	Port               string   `mapstructure:"PORT"`
	MongoURI           string   `mapstructure:"MONGO_URI"`
	RedisAddr          string   `mapstructure:"REDIS_ADDR"`
	BrokerType         string   `mapstructure:"BROKER_TYPE"`
	LogLevel           string   `mapstructure:"LOG_LEVEL"`
	S3Endpoint         string   `mapstructure:"S3_ENDPOINT"`
	S3AccessKey        string   `mapstructure:"S3_ACCESS_KEY"`
	S3SecretKey        string   `mapstructure:"S3_SECRET_KEY"`
	S3Bucket           string   `mapstructure:"S3_BUCKET"`
	S3UseSSL           bool     `mapstructure:"S3_USE_SSL"`
	CertDir            string   `mapstructure:"CERT_DIR"`
	CertCN             string   `mapstructure:"CERT_CN"`
	CertDNS            []string `mapstructure:"CERT_DNS"`
	GracefulTimeoutSec int      `mapstructure:"GRACEFUL_TIMEOUT_SEC"`
}

func LoadConfig() *Config {
	pflag.String("port", "8080", "Port to listen on")
	pflag.String("mongo-uri", "mongodb://user:password@localhost:27017/chat_db", "MongoDB connection URI")
	pflag.String("redis-addr", "localhost:6379", "Redis address")
	pflag.String("broker-type", "local", "Broker type (local or redis)")
	pflag.String("log-level", "info", "Log level (debug, info, warn, error)")
	pflag.String("s3-endpoint", "localhost:9000", "S3 storage endpoint")
	pflag.String("s3-access-key", "minioadmin", "S3 access key")
	pflag.String("s3-secret-key", "minioadmin", "S3 secret key")
	pflag.String("s3-bucket", "chat-media", "S3 bucket name")
	pflag.Bool("s3-use-ssl", false, "Use SSL for S3")
	pflag.String("cert-dir", "certs", "Directory containing TLS certificates")
	pflag.String("cert-cn", "localhost", "Common Name for the server certificate")
	pflag.StringSlice("cert-dns", []string{"localhost", "127.0.0.1"}, "DNS names for the server certificate")
	pflag.Int("graceful-timeout", 30, "Graceful shutdown timeout in seconds")

	pflag.Parse()

	// Bind flags to viper
	viper.BindPFlag("PORT", pflag.Lookup("port"))
	viper.BindPFlag("MONGO_URI", pflag.Lookup("mongo-uri"))
	viper.BindPFlag("REDIS_ADDR", pflag.Lookup("redis-addr"))
	viper.BindPFlag("BROKER_TYPE", pflag.Lookup("broker-type"))
	viper.BindPFlag("LOG_LEVEL", pflag.Lookup("log-level"))
	viper.BindPFlag("S3_ENDPOINT", pflag.Lookup("s3-endpoint"))
	viper.BindPFlag("S3_ACCESS_KEY", pflag.Lookup("s3-access-key"))
	viper.BindPFlag("S3_SECRET_KEY", pflag.Lookup("s3-secret-key"))
	viper.BindPFlag("S3_BUCKET", pflag.Lookup("s3-bucket"))
	viper.BindPFlag("S3_USE_SSL", pflag.Lookup("s3-use-ssl"))
	viper.BindPFlag("CERT_DIR", pflag.Lookup("cert-dir"))
	viper.BindPFlag("CERT_CN", pflag.Lookup("cert-cn"))
	viper.BindPFlag("CERT_DNS", pflag.Lookup("cert-dns"))
	viper.BindPFlag("GRACEFUL_TIMEOUT_SEC", pflag.Lookup("graceful-timeout"))

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("unable to decode into struct, %v", err)
	}

	return &cfg
}
