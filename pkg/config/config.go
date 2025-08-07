// pkg/config/config.go - FIXED configuration loading with proper environment handling
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the complete application configuration
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

// Server configuration
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

// Load loads configuration from files and environment variables
func Load() (*Config, error) {
	// CRITICAL: Load environment file FIRST before any viper operations
	loadEnvFile()

	// Initialize viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config paths
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/oceanproxy")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Read config file (ignore if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		fmt.Println("No config file found, using environment variables and defaults")
	} else {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Setup environment variable handling
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Override with direct environment variable reading for critical values
	if bearerToken := os.Getenv("BEARER_TOKEN"); bearerToken != "" {
		viper.Set("auth.bearer_token", bearerToken)
		fmt.Printf("✅ Loaded BEARER_TOKEN from environment: %s\n", bearerToken)
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		viper.Set("auth.jwt_secret", jwtSecret)
		fmt.Printf("✅ Loaded JWT_SECRET from environment\n")
	}

	if proxiesFoKey := os.Getenv("PROXIES_FO_API_KEY"); proxiesFoKey != "" {
		viper.Set("providers.proxies_fo.api_key", proxiesFoKey)
		fmt.Printf("✅ Loaded PROXIES_FO_API_KEY from environment\n")
	}

	if nettifyKey := os.Getenv("NETTIFY_API_KEY"); nettifyKey != "" {
		viper.Set("providers.nettify.api_key", nettifyKey)
		fmt.Printf("✅ Loaded NETTIFY_API_KEY from environment\n")
	}

	// Unmarshal into config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate critical configuration
	if cfg.Auth.BearerToken == "" {
		return nil, fmt.Errorf("BEARER_TOKEN is required but not set")
	}

	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required but not set")
	}

	fmt.Printf("✅ Configuration loaded successfully\n")
	fmt.Printf("   - Bearer Token: %s\n", cfg.Auth.BearerToken)
	fmt.Printf("   - Environment: %s\n", cfg.Environment)
	fmt.Printf("   - Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)

	return &cfg, nil
}

// loadEnvFile manually loads the environment file with priority order
func loadEnvFile() {
	// Priority order: system location first, then local fallbacks
	envPaths := []string{
		"/etc/oceanproxy/oceanproxy.env", // System location (highest priority)
		"./oceanproxy.env",               // Current directory
		"./.env",                         // Standard .env file
	}

	for _, path := range envPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fmt.Printf("📄 Loading environment file: %s\n", path)
			if loadEnvFromFile(path) {
				return // Stop after successfully loading the first found file
			}
		}
	}

	fmt.Println("⚠️  No environment file found in any of the expected locations:")
	for _, path := range envPaths {
		fmt.Printf("   - %s\n", path)
	}
}

// loadEnvFromFile reads and sets environment variables from a file
func loadEnvFromFile(filename string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("❌ Error reading env file %s: %v\n", filename, err)
		return false
	}

	loadedCount := 0
	// Parse and set environment variables
	lines := strings.Split(string(content), "\n")
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("⚠️  Skipping invalid line %d in %s: %s\n", lineNum+1, filename, line)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			value = value[1 : len(value)-1]
		}

		// Set environment variable only if not already set
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				fmt.Printf("❌ Error setting env var %s: %v\n", key, err)
				continue
			}
			loadedCount++

			// Show masked value for security
			displayValue := value
			if len(value) > 8 && (strings.Contains(strings.ToLower(key), "secret") ||
				strings.Contains(strings.ToLower(key), "token") ||
				strings.Contains(strings.ToLower(key), "key") ||
				strings.Contains(strings.ToLower(key), "password")) {
				displayValue = value[:4] + "****" + value[len(value)-4:]
			}
			fmt.Printf("   %s=%s\n", key, displayValue)
		} else {
			fmt.Printf("   %s already set (skipping)\n", key)
		}
	}

	fmt.Printf("✅ Loaded %d environment variables from %s\n", loadedCount, filename)
	return true
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("environment", "development")
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

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")

	// Auth defaults
	viper.SetDefault("auth.bearer_token", "simple123")
	viper.SetDefault("auth.jwt_secret", "0153c4b35f43569b0591350011a38efdcbdd460f11fb6fe6687ae6504b0aa982")
	viper.SetDefault("auth.token_ttl", "24h")

	// Provider defaults
	viper.SetDefault("providers.proxies_fo.base_url", "https://api.proxies.fo")
	viper.SetDefault("providers.proxies_fo.timeout", "30s")
	viper.SetDefault("providers.nettify.base_url", "https://api.nettify.xyz")
	viper.SetDefault("providers.nettify.timeout", "30s")

	// Proxy defaults
	viper.SetDefault("proxy.domain", "oceanproxy.io")
	viper.SetDefault("proxy.start_port", 10000)
	viper.SetDefault("proxy.end_port", 30000)
	viper.SetDefault("proxy.config_dir", "/etc/3proxy")
	viper.SetDefault("proxy.log_dir", "/var/log/oceanproxy")
	viper.SetDefault("proxy.script_dir", "/opt/oceanproxy/scripts")
	viper.SetDefault("proxy.nginx_conf_dir", "/etc/nginx/conf.d")
}
