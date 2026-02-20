// Command claudebot-mcp is the entry point for the Discord MCP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
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

var stdioFlag = flag.Bool("stdio", false, "use stdio transport instead of HTTP")

func main() {
	flag.Parse()

	// 1. Load config (before structured logger exists, uses stderr for errors).
	cfg := loadConfig()

	// 2. Apply environment variable overrides.
	config.ApplyEnvOverrides(cfg)

	// 3. Build structured logger from config.
	logLevel := config.ParseLogLevel(cfg.Logging.Level)
	slogHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(slogHandler)

	// Create a *log.Logger bridge for mcp-go compatibility.
	stdLogger := slog.NewLogLogger(slogHandler, slog.LevelError)

	// 4. Open audit log file if enabled.
	var auditLogger *safety.AuditLogger
	if cfg.Audit.Enabled {
		f, err := os.OpenFile(cfg.Audit.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			logger.Warn("could not open audit log, audit logging disabled",
				"path", cfg.Audit.LogPath, "error", err)
		} else {
			auditLogger = safety.NewAuditLogger(f)
			defer func() { _ = f.Close() }()
		}
	}

	// 5. Build safety components.
	channelFilter := safety.NewFilter(
		cfg.Safety.Channels.Allowlist,
		cfg.Safety.Channels.Denylist,
	)
	confirm := safety.NewConfirmationTracker(message.DestructiveToolNames())

	// 6. Build queue.
	q := queue.New(queue.WithMaxSize(cfg.Queue.MaxSize))

	// 7. Create raw discordgo session.
	rawDG, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		logger.Error("failed to create Discord session", "error", err)
		os.Exit(1)
	}

	// 8. Create resolver.
	resolver := resolve.New(rawDG, cfg.Discord.GuildID)

	// 9. Create discord.Session (registers event handlers and intents).
	discordSession := discord.NewFromSession(rawDG, q, resolver, logger)
	_ = discordSession // event handlers registered; Close called on shutdown

	// 9a. Set initial presence (online from first connect).
	rawDG.Identify.Presence = discordgo.GatewayStatusUpdate{
		Status: "online",
		Game: discordgo.Activity{
			Name: "the server",
			Type: discordgo.ActivityTypeWatching,
		},
	}

	// 10. Open Discord connection.
	if err := rawDG.Open(); err != nil {
		logger.Error("failed to open Discord connection", "error", err)
		os.Exit(1)
	}

	// 11. Build MCP server.
	mcpServer := server.NewMCPServer(
		"claudebot-mcp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// 12. Register all tools.
	var registrations []tools.Registration
	registrations = append(registrations,
		message.MessageTools(rawDG, q, resolver, channelFilter, confirm, auditLogger, logger)...,
	)
	registrations = append(registrations,
		reaction.ReactionTools(rawDG, resolver, channelFilter, auditLogger, logger)...,
	)
	registrations = append(registrations,
		channel.ChannelTools(rawDG, resolver, cfg.Discord.GuildID, channelFilter, auditLogger, logger)...,
	)
	registrations = append(registrations,
		user.UserTools(rawDG, auditLogger, logger)...,
	)
	registrations = append(registrations,
		guild.GuildTools(rawDG, cfg.Discord.GuildID, auditLogger, logger)...,
	)

	tools.RegisterAll(mcpServer, registrations)

	// 13. Start in stdio or HTTP mode.
	if *stdioFlag {
		logger.Info("starting in stdio mode")
		if err := server.ServeStdio(mcpServer, server.WithErrorLogger(stdLogger)); err != nil {
			logger.Error("stdio server error", "error", err)
		}
	} else {
		httpHandler := server.NewStreamableHTTPServer(mcpServer)
		authMiddleware := auth.NewAuthMiddleware(cfg.Server.AuthToken, logger)
		wrappedHandler := authMiddleware(httpHandler)

		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		httpSrv := &http.Server{
			Addr:              addr,
			Handler:           wrappedHandler,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}

		go func() {
			logger.Info("listening", "addr", addr)
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("HTTP server error", "error", err)
				os.Exit(1)
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		logger.Info("shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := httpSrv.Shutdown(ctx); err != nil {
			logger.Error("HTTP shutdown error", "error", err)
		}
	}

	// 14. Close Discord session.
	if err := rawDG.Close(); err != nil {
		logger.Error("Discord close error", "error", err)
	}

	logger.Info("server stopped")
}

// loadConfig attempts to read the config file from the path specified by
// CLAUDEBOT_CONFIG_PATH or the default "config.yaml". If the file cannot be
// read, DefaultConfig is returned. Uses fmt.Fprintf to stderr because the
// structured logger has not been constructed yet (it depends on config).
func loadConfig() *config.Config {
	path := os.Getenv("CLAUDEBOT_CONFIG_PATH")
	if path == "" {
		path = defaultConfigPath
	}

	cfg, err := config.LoadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claudebot-mcp: could not load config from %q (%v), using defaults\n", path, err)
		return config.DefaultConfig()
	}

	return cfg
}
