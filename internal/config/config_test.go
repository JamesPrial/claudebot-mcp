package config

import (
	"path/filepath"
	"runtime"
	"testing"
)

// testdataDir returns the absolute path to the testdata/config directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	// thisFile is internal/config/config_test.go
	// project root is two levels up
	projectRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")
	return filepath.Join(projectRoot, "testdata", "config")
}

// ---------------------------------------------------------------------------
// LoadConfig
// ---------------------------------------------------------------------------

func Test_LoadConfig_ValidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(testdataDir(t), "valid.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig returned nil config for valid YAML")
	}

	// Verify server section
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Server.AuthToken != "test-auth-token-123" {
		t.Errorf("Server.AuthToken = %q, want %q", cfg.Server.AuthToken, "test-auth-token-123")
	}

	// Verify discord section
	if cfg.Discord.Token != "discord-bot-token-abc" {
		t.Errorf("Discord.Token = %q, want %q", cfg.Discord.Token, "discord-bot-token-abc")
	}
	if cfg.Discord.GuildID != "123456789" {
		t.Errorf("Discord.GuildID = %q, want %q", cfg.Discord.GuildID, "123456789")
	}

	// Verify queue section
	if cfg.Queue.MaxSize != 500 {
		t.Errorf("Queue.MaxSize = %d, want 500", cfg.Queue.MaxSize)
	}
	if cfg.Queue.PollTimeoutSec != 15 {
		t.Errorf("Queue.PollTimeoutSec = %d, want 15", cfg.Queue.PollTimeoutSec)
	}

	// Verify audit section
	if !cfg.Audit.Enabled {
		t.Error("Audit.Enabled = false, want true")
	}
	if cfg.Audit.LogPath != "/tmp/audit.log" {
		t.Errorf("Audit.LogPath = %q, want %q", cfg.Audit.LogPath, "/tmp/audit.log")
	}

	// Verify logging section
	if cfg.Logging.Level != "debug" {
		t.Errorf("Logging.Level = %q, want %q", cfg.Logging.Level, "debug")
	}
}

func Test_LoadConfig_NonexistentFile(t *testing.T) {
	t.Parallel()
	cfg, err := LoadConfig("/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Fatal("LoadConfig with nonexistent file should return error")
	}
	if cfg != nil {
		t.Error("LoadConfig with nonexistent file should return nil config")
	}
}

func Test_LoadConfig_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(testdataDir(t), "invalid.yaml")
	cfg, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig with invalid YAML should return error")
	}
	if cfg != nil {
		t.Error("LoadConfig with invalid YAML should return nil config")
	}
}

func Test_LoadConfig_EmptyFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(testdataDir(t), "empty.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig with empty file should return non-nil config")
	}
	// All fields should be zero values
	if cfg.Server.Port != 0 {
		t.Errorf("Server.Port = %d, want 0 for empty file", cfg.Server.Port)
	}
	if cfg.Server.AuthToken != "" {
		t.Errorf("Server.AuthToken = %q, want empty for empty file", cfg.Server.AuthToken)
	}
	if cfg.Discord.Token != "" {
		t.Errorf("Discord.Token = %q, want empty for empty file", cfg.Discord.Token)
	}
}

func Test_LoadConfig_MinimalYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(testdataDir(t), "minimal.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig with minimal YAML should return non-nil config")
	}
	// Server section should be populated
	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
	}
	if cfg.Server.AuthToken != "minimal-token" {
		t.Errorf("Server.AuthToken = %q, want %q", cfg.Server.AuthToken, "minimal-token")
	}
	// Other sections should be zero values
	if cfg.Discord.Token != "" {
		t.Errorf("Discord.Token = %q, want empty for minimal config", cfg.Discord.Token)
	}
	if cfg.Queue.MaxSize != 0 {
		t.Errorf("Queue.MaxSize = %d, want 0 for minimal config", cfg.Queue.MaxSize)
	}
}

func Test_LoadConfig_UnknownKeys(t *testing.T) {
	t.Parallel()
	path := filepath.Join(testdataDir(t), "unknown_keys.yaml")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig(%q) unexpected error: %v", path, err)
	}
	if cfg == nil {
		t.Fatal("LoadConfig with unknown keys should return non-nil config")
	}
	// Known fields should be populated
	if cfg.Server.Port != 4000 {
		t.Errorf("Server.Port = %d, want 4000", cfg.Server.Port)
	}
	if cfg.Server.AuthToken != "uk-token" {
		t.Errorf("Server.AuthToken = %q, want %q", cfg.Server.AuthToken, "uk-token")
	}
}

// ---------------------------------------------------------------------------
// DefaultConfig
// ---------------------------------------------------------------------------

func Test_DefaultConfig_Fields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(cfg *Config) bool
		want  string
	}{
		{
			name:  "Server.Port is 8080",
			check: func(cfg *Config) bool { return cfg.Server.Port == 8080 },
			want:  "Server.Port == 8080",
		},
		{
			name:  "Server.AuthToken is empty",
			check: func(cfg *Config) bool { return cfg.Server.AuthToken == "" },
			want:  "Server.AuthToken == \"\"",
		},
		{
			name:  "Discord.Token is empty",
			check: func(cfg *Config) bool { return cfg.Discord.Token == "" },
			want:  "Discord.Token == \"\"",
		},
		{
			name:  "Queue.MaxSize is 1000",
			check: func(cfg *Config) bool { return cfg.Queue.MaxSize == 1000 },
			want:  "Queue.MaxSize == 1000",
		},
		{
			name:  "Queue.PollTimeoutSec is 30",
			check: func(cfg *Config) bool { return cfg.Queue.PollTimeoutSec == 30 },
			want:  "Queue.PollTimeoutSec == 30",
		},
		{
			name:  "Audit.Enabled is true",
			check: func(cfg *Config) bool { return cfg.Audit.Enabled },
			want:  "Audit.Enabled == true",
		},
		{
			name:  "Audit.LogPath is audit.log",
			check: func(cfg *Config) bool { return cfg.Audit.LogPath == "audit.log" },
			want:  "Audit.LogPath == \"audit.log\"",
		},
		{
			name:  "Logging.Level is info",
			check: func(cfg *Config) bool { return cfg.Logging.Level == "info" },
			want:  "Logging.Level == \"info\"",
		},
	}

	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !tt.check(cfg) {
				t.Errorf("DefaultConfig() failed check: want %s", tt.want)
			}
		})
	}
}

func Test_DefaultConfig_SafetyDestructiveTools(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	found := false
	for _, tool := range cfg.Safety.DestructiveTools {
		if tool == "discord_delete_message" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("DefaultConfig().Safety.DestructiveTools should contain %q, got %v",
			"discord_delete_message", cfg.Safety.DestructiveTools)
	}
}

func Test_DefaultConfig_ReturnsDistinctPointers(t *testing.T) {
	t.Parallel()
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()
	if cfg1 == cfg2 {
		t.Error("DefaultConfig() should return distinct pointers on each call")
	}
}

// ---------------------------------------------------------------------------
// ApplyEnvOverrides
// ---------------------------------------------------------------------------

func Test_ApplyEnvOverrides_Cases(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		initial    Config
		checkField string
		wantValue  string
	}{
		{
			name:       "CLAUDEBOT_DISCORD_TOKEN overrides empty token",
			envVars:    map[string]string{"CLAUDEBOT_DISCORD_TOKEN": "tok1"},
			initial:    Config{},
			checkField: "Discord.Token",
			wantValue:  "tok1",
		},
		{
			name:       "CLAUDEBOT_DISCORD_TOKEN overrides existing token",
			envVars:    map[string]string{"CLAUDEBOT_DISCORD_TOKEN": "tok1"},
			initial:    Config{Discord: DiscordConfig{Token: "old"}},
			checkField: "Discord.Token",
			wantValue:  "tok1",
		},
		{
			name:       "empty CLAUDEBOT_DISCORD_TOKEN does not override",
			envVars:    map[string]string{"CLAUDEBOT_DISCORD_TOKEN": ""},
			initial:    Config{Discord: DiscordConfig{Token: "old"}},
			checkField: "Discord.Token",
			wantValue:  "old",
		},
		{
			name:       "CLAUDEBOT_DISCORD_GUILD_ID sets guild ID",
			envVars:    map[string]string{"CLAUDEBOT_DISCORD_GUILD_ID": "g1"},
			initial:    Config{},
			checkField: "Discord.GuildID",
			wantValue:  "g1",
		},
		{
			name:       "CLAUDEBOT_AUTH_TOKEN sets auth token",
			envVars:    map[string]string{"CLAUDEBOT_AUTH_TOKEN": "auth1"},
			initial:    Config{},
			checkField: "Server.AuthToken",
			wantValue:  "auth1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars using t.Setenv (auto-cleanup)
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg := tt.initial
			ApplyEnvOverrides(&cfg)

			var got string
			switch tt.checkField {
			case "Discord.Token":
				got = cfg.Discord.Token
			case "Discord.GuildID":
				got = cfg.Discord.GuildID
			case "Server.AuthToken":
				got = cfg.Server.AuthToken
			default:
				t.Fatalf("unknown checkField: %s", tt.checkField)
			}

			if got != tt.wantValue {
				t.Errorf("%s = %q, want %q", tt.checkField, got, tt.wantValue)
			}
		})
	}
}

func Test_ApplyEnvOverrides_AllThreeSet(t *testing.T) {
	t.Setenv("CLAUDEBOT_DISCORD_TOKEN", "tok1")
	t.Setenv("CLAUDEBOT_DISCORD_GUILD_ID", "g1")
	t.Setenv("CLAUDEBOT_AUTH_TOKEN", "auth1")

	cfg := Config{}
	ApplyEnvOverrides(&cfg)

	if cfg.Discord.Token != "tok1" {
		t.Errorf("Discord.Token = %q, want %q", cfg.Discord.Token, "tok1")
	}
	if cfg.Discord.GuildID != "g1" {
		t.Errorf("Discord.GuildID = %q, want %q", cfg.Discord.GuildID, "g1")
	}
	if cfg.Server.AuthToken != "auth1" {
		t.Errorf("Server.AuthToken = %q, want %q", cfg.Server.AuthToken, "auth1")
	}
}
