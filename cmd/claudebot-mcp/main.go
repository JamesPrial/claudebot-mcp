// Command claudebot-mcp is the entry point for the Discord MCP server.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/auth"
	"github.com/jamesprial/claudebot-mcp/internal/channel"
	"github.com/jamesprial/claudebot-mcp/internal/config"
	"github.com/jamesprial/claudebot-mcp/internal/discord"
	"github.com/jamesprial/claudebot-mcp/internal/guild"
	"github.com/jamesprial/claudebot-mcp/internal/message"
	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/reaction"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/jamesprial/claudebot-mcp/internal/user"
	"github.com/mark3labs/mcp-go/server"
)

const defaultConfigPath = "config.yaml"

func main() {
	logger := log.New(os.Stderr, "claudebot-mcp: ", log.LstdFlags)

	// 1. Load config.
	cfg := loadConfig(logger)

	// 2. Apply environment variable overrides.
	config.ApplyEnvOverrides(cfg)

	// 3. Open audit log file if enabled.
	var auditLogger *safety.AuditLogger
	if cfg.Audit.Enabled {
		f, err := os.OpenFile(cfg.Audit.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			logger.Printf("warning: could not open audit log %q: %v — audit logging disabled", cfg.Audit.LogPath, err)
		} else {
			auditLogger = safety.NewAuditLogger(f)
			defer func() { _ = f.Close() }()
		}
	}

	// 4. Build safety components.
	channelFilter := safety.NewFilter(
		cfg.Safety.Channels.Allowlist,
		cfg.Safety.Channels.Denylist,
	)
	confirm := safety.NewConfirmationTracker(message.DestructiveTools)

	// 5. Build queue.
	q := queue.New(queue.WithMaxSize(cfg.Queue.MaxSize))

	// 6. Create raw discordgo session.
	rawDG, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		logger.Fatalf("failed to create Discord session: %v", err)
	}

	// 7. Create resolver.
	resolver := resolve.New(rawDG, cfg.Discord.GuildID)

	// 8. Create discord.Session (registers event handlers and intents).
	discordSession := discord.NewFromSession(rawDG, q, resolver)
	_ = discordSession // event handlers registered; Close called on shutdown

	// 9. Open Discord connection.
	if err := rawDG.Open(); err != nil {
		logger.Fatalf("failed to open Discord connection: %v", err)
	}

	// 10. Build MCP server.
	mcpServer := server.NewMCPServer(
		"claudebot-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// 11. Register all tools.
	var registrations []tools.Registration
	registrations = append(registrations,
		message.MessageTools(rawDG, q, resolver, channelFilter, confirm, auditLogger)...,
	)
	registrations = append(registrations,
		reaction.ReactionTools(rawDG, resolver, channelFilter, auditLogger)...,
	)
	registrations = append(registrations,
		channel.ChannelTools(rawDG, resolver, cfg.Discord.GuildID, channelFilter, auditLogger)...,
	)
	registrations = append(registrations,
		user.UserTools(rawDG, auditLogger)...,
	)
	registrations = append(registrations,
		guild.GuildTools(rawDG, cfg.Discord.GuildID, auditLogger)...,
	)

	tools.RegisterAll(mcpServer, registrations)

	// 12. Build StreamableHTTPServer and wrap with auth middleware.
	httpHandler := server.NewStreamableHTTPServer(mcpServer)
	authMiddleware := auth.NewAuthMiddleware(cfg.Server.AuthToken)
	wrappedHandler := authMiddleware(httpHandler)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           wrappedHandler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 13. Start HTTP server in goroutine.
	go func() {
		logger.Printf("listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 14. Wait for SIGINT or SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Println("shutting down...")

	// 15. Graceful shutdown: HTTP server (15s timeout), then Discord session.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Printf("HTTP shutdown error: %v", err)
	}

	if err := rawDG.Close(); err != nil {
		logger.Printf("Discord close error: %v", err)
	}

	logger.Println("server stopped")
}

// loadConfig attempts to read the config file from the path specified by
// CLAUDEBOT_CONFIG_PATH or the default "config.yaml". If the file cannot be
// read, DefaultConfig is returned.
func loadConfig(logger *log.Logger) *config.Config {
	path := os.Getenv("CLAUDEBOT_CONFIG_PATH")
	if path == "" {
		path = defaultConfigPath
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		logger.Printf("could not load config from %q (%v), using defaults", path, err)
		return config.DefaultConfig()
	}

	logger.Printf("loaded config from %q", path)
	return cfg
}
