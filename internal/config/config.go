package config

import (
	"errors"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP          HTTPConfig     `mapstructure:"http"`
	LLM           LLMConfig      `mapstructure:"llm"`
	MCP           MCPConfig      `mapstructure:"mcp"`
	Database      DatabaseConfig `mapstructure:"database"`
	Telegram      TelegramConfig `mapstructure:"telegram"`
	Slack         SlackConfig    `mapstructure:"slack"`
	Redis         RedisConfig    `mapstructure:"redis"`
	AlertCooldown string         `mapstructure:"alert_cooldown"`
}

type HTTPConfig struct {
	Addr string `mapstructure:"addr"`
}

type LLMConfig struct {
	Provider       string `mapstructure:"provider"`
	APIKey         string `mapstructure:"api_key"`
	BaseURL        string `mapstructure:"base_url"`
	Model          string `mapstructure:"model"`
	MaxTokens      int    `mapstructure:"max_tokens"`
	SummaryMaxToks int    `mapstructure:"summary_max_tokens"`
	ContextLimit   int    `mapstructure:"context_limit"`
}

type MCPConfig struct {
	ConfigPath string `mapstructure:"config_path"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type TelegramConfig struct {
	BotToken       string `mapstructure:"bot_token"`
	DefaultChannel string `mapstructure:"default_channel"`
}

type SlackConfig struct {
	BotToken       string `mapstructure:"bot_token"`
	DefaultChannel string `mapstructure:"default_channel"`
}

type RedisConfig struct {
	URL string `mapstructure:"url"`
}

func Load(path string) (Config, error) {
	// Load .env if present; ignore missing file
	_ = godotenv.Load()

	v := viper.New()

	v.SetDefault("http.addr", ":8080")
	v.SetDefault("llm.provider", "openai")
	// Empty model lets each provider apply its own default
	// (openai → gpt-4o, anthropic → claude-opus-4-8).
	v.SetDefault("llm.model", "")
	v.SetDefault("llm.max_tokens", 32768)
	v.SetDefault("llm.summary_max_tokens", 4096)
	v.SetDefault("llm.context_limit", 0)
	v.SetDefault("alert_cooldown", "48h")

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/alert-agent")
	}

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return Config{}, err
		}
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	_ = v.BindEnv("http.addr", "HTTP_ADDR")
	_ = v.BindEnv("llm.provider", "LLM_PROVIDER")
	_ = v.BindEnv("llm.api_key", "LLM_API_KEY", "OPENAI_API_KEY")
	_ = v.BindEnv("llm.base_url", "LLM_BASE_URL", "OPENAI_BASE_URL")
	_ = v.BindEnv("llm.model", "LLM_MODEL")
	_ = v.BindEnv("llm.max_tokens", "LLM_MAX_TOKENS")
	_ = v.BindEnv("llm.summary_max_tokens", "LLM_SUMMARY_MAX_TOKENS")
	_ = v.BindEnv("llm.context_limit", "LLM_CONTEXT_LIMIT")
	v.SetDefault("mcp.config_path", ".mcp.json")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "alert-agent.db")

	_ = v.BindEnv("mcp.config_path", "MCP_CONFIG_PATH")
	_ = v.BindEnv("database.driver", "DATABASE_DRIVER")
	_ = v.BindEnv("database.dsn", "DATABASE_DSN")
	_ = v.BindEnv("telegram.bot_token", "TELEGRAM_BOT_TOKEN")
	_ = v.BindEnv("telegram.default_channel", "TELEGRAM_DEFAULT_CHANNEL")
	_ = v.BindEnv("slack.bot_token", "SLACK_BOT_TOKEN")
	_ = v.BindEnv("slack.default_channel", "SLACK_DEFAULT_CHANNEL")
	_ = v.BindEnv("redis.url", "REDIS_URL")
	_ = v.BindEnv("alert_cooldown", "ALERT_COOLDOWN")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
