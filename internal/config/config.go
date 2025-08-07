package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment string    `mapstructure:"environment"`
	Server      Server    `mapstructure:"server"`
	Database    Database  `mapstructure:"database"`
	Redis       Redis     `mapstructure:"redis"`
	Logger      Logger    `mapstructure:"logger"`
	Auth        Auth      `mapstructure:"auth"`
	Providers   Providers `mapstructure:"providers"`
	Proxy       Proxy     `mapstructure:"proxy"`
}

type Server struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	CORS            CORS          `mapstructure:"cors"`
}

type CORS struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

type Database struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Logger struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type Auth struct {
	BearerToken string        `mapstructure:"bearer_token"`
	JWTSecret   string        `mapstructure:"jwt_secret"`
	TokenTTL    time.Duration `mapstructure:"token_ttl"`
}

type Providers struct {
	ProxiesFo ProxiesFoConfig `mapstructure:"proxies_fo"`
	Nettify   NettifyConfig   `mapstructure:"nettify"`
}

type ProxiesFoConfig struct {
	APIKey  string        `mapstructure:"api_key"`
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type NettifyConfig struct {
	APIKey  string        `mapstructure:"api_key"`
	BaseURL string        `mapstructure:"base_url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type Proxy struct {
	Domain       string `mapstructure:"domain"`
	StartPort    int    `mapstructure:"start_port"`
	EndPort      int    `mapstructure:"end_port"`
	ConfigDir    string `mapstructure:"config_dir"`
	LogDir       string `mapstructure:"log_dir"`
	ScriptDir    string `mapstructure:"script_dir"`
	NginxConfDir string `mapstructure:"nginx_conf_dir"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.shutdown_timeout", "30s")

	// CORS defaults
	viper.SetDefault("server.cors.allow_origins", []string{"*"})
	viper.SetDefault("server.cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("server.cors.allow_headers", []string{"*"})
	viper.SetDefault("server.cors.allow_credentials", true)

	// Database defaults
	viper.SetDefault("database.driver", "json")
	viper.SetDefault("database.dsn", "/var/lib/oceanproxy/data/proxies.json") // ADD THIS LINE
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 25)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")

	// Auth defaults
	viper.SetDefault("auth.token_ttl", "24h")

	// Provider defaults
	viper.SetDefault("providers.proxies_fo.base_url", "https://app.proxies.fo")
	viper.SetDefault("providers.proxies_fo.timeout", "30s")
	viper.SetDefault("providers.nettify.base_url", "https://api.nettify.xyz")
	viper.SetDefault("providers.nettify.timeout", "30s")

	// Proxy defaults
	viper.SetDefault("proxy.domain", "oceanproxy.io")
	viper.SetDefault("proxy.start_port", 10000)
	viper.SetDefault("proxy.end_port", 20000)
	viper.SetDefault("proxy.config_dir", "/etc/3proxy")
	viper.SetDefault("proxy.log_dir", "/var/log/oceanproxy")
	viper.SetDefault("proxy.script_dir", "./scripts")
	viper.SetDefault("proxy.nginx_conf_dir", "/etc/nginx/conf.d")

	// Environment
	viper.SetDefault("environment", "development")
}
