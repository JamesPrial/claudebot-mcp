// Package config provides configuration loading and defaults for the claudebot-mcp server.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ServerConfig holds network and authentication settings.
type ServerConfig struct {
	Port      int    `yaml:"port"`
	AuthToken string `yaml:"auth_token"`
}

// DiscordConfig holds Discord bot credentials and guild targeting.
type DiscordConfig struct {
	Token   string `yaml:"token"`
	GuildID string `yaml:"guild_id"`
}

// QueueConfig controls the internal message queue behaviour.
type QueueConfig struct {
	MaxSize int `yaml:"max_size"`
	// PollTimeoutSec is loaded from config but currently unused at runtime.
	// The poll timeout is specified per-request by the MCP client (default 30, max 300).
	PollTimeoutSec int `yaml:"poll_timeout_sec"`
}

// ChannelFilter holds allowlist and denylist entries for Discord channel filtering.
type ChannelFilter struct {
	Allowlist []string `yaml:"allowlist"`
	Denylist  []string `yaml:"denylist"`
}

// SafetyConfig groups channel filters and destructive tool declarations.
type SafetyConfig struct {
	Channels ChannelFilter `yaml:"channels"`
	// DestructiveTools is loaded from config but currently unused at runtime.
	// The destructive tool list is defined in message.DestructiveTools.
	DestructiveTools []string `yaml:"destructive_tools"`
}

// AuditConfig controls audit logging behaviour.
type AuditConfig struct {
	Enabled bool   `yaml:"enabled"`
	LogPath string `yaml:"log_path"`
	// MaxSizeMB is loaded from config but no log rotation is currently implemented.
	MaxSizeMB int `yaml:"max_size_mb"`
}

// LoggingConfig controls structured log output.
type LoggingConfig struct {
	Level string `yaml:"level"`
}

// Config is the top-level configuration structure for the claudebot-mcp server.
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Discord DiscordConfig `yaml:"discord"`
	Queue   QueueConfig   `yaml:"queue"`
	Safety  SafetyConfig  `yaml:"safety"`
	Audit   AuditConfig   `yaml:"audit"`
	Logging LoggingConfig `yaml:"logging"`
}

// LoadConfig reads and parses a YAML configuration file from the given path.
// It returns a pointer to the populated Config and any error encountered.
// On error, nil is returned for the config pointer.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// DefaultConfig returns a new Config populated with sensible default values.
// Each call returns a distinct instance.
//
// Defaults:
//   - Server.Port = 8080
//   - Queue.MaxSize = 1000
//   - Audit.Enabled = true
//   - Audit.LogPath = "audit.log"
//   - Logging.Level = "info"
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Queue: QueueConfig{
			MaxSize: 1000,
		},
		Audit: AuditConfig{
			Enabled: true,
			LogPath: "audit.log",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}

// ApplyEnvOverrides updates cfg in place with values from environment variables.
// Only non-empty environment variable values override existing config values.
//
// Recognized variables:
//   - CLAUDEBOT_DISCORD_TOKEN  -> cfg.Discord.Token
//   - CLAUDEBOT_DISCORD_GUILD_ID -> cfg.Discord.GuildID
//   - CLAUDEBOT_AUTH_TOKEN -> cfg.Server.AuthToken
func ApplyEnvOverrides(cfg *Config) {
	if token := os.Getenv("CLAUDEBOT_DISCORD_TOKEN"); token != "" {
		cfg.Discord.Token = token
	}
	if guildID := os.Getenv("CLAUDEBOT_DISCORD_GUILD_ID"); guildID != "" {
		cfg.Discord.GuildID = guildID
	}
	if authToken := os.Getenv("CLAUDEBOT_AUTH_TOKEN"); authToken != "" {
		cfg.Server.AuthToken = authToken
	}
}
