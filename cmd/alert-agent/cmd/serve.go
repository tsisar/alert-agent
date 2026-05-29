package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/tsisar/alert-agent/internal/agent"
	"github.com/tsisar/alert-agent/internal/llm"
	"github.com/tsisar/alert-agent/internal/llm/anthropic"
	"github.com/tsisar/alert-agent/internal/llm/openai"
	"github.com/tsisar/alert-agent/internal/mcp"
	"github.com/tsisar/alert-agent/internal/notify"
	"github.com/tsisar/alert-agent/internal/notify/slack"
	"github.com/tsisar/alert-agent/internal/notify/telegram"
	"github.com/tsisar/alert-agent/internal/scenario"
	"github.com/tsisar/alert-agent/internal/server"
	"github.com/tsisar/alert-agent/internal/storage"
	"github.com/tsisar/alert-agent/internal/web"
	"github.com/tsisar/alert-agent/internal/webhook"
	"github.com/tsisar/extended-log-go/log"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP webhook server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		// Initialize database
		db, err := storage.Open(cfg.Database.Driver, cfg.Database.DSN)
		if err != nil {
			return fmt.Errorf("open database: %w", err)
		}
		if err := storage.Seed(db); err != nil {
			return fmt.Errorf("seed database: %w", err)
		}
		log.Infof("database initialized: driver=%s", cfg.Database.Driver)

		scenarioRepo := storage.NewScenarioRepository(db)
		promptRepo := storage.NewPromptRepository(db)

		// Initialize notifiers
		notifiers := make(map[string]notify.Notifier)

		if cfg.Telegram.BotToken != "" {
			tg, err := telegram.New(cfg.Telegram.BotToken)
			if err != nil {
				return err
			}
			notifiers["telegram"] = tg
			log.Info("telegram notifier initialized")
		}

		if cfg.Slack.BotToken != "" {
			notifiers["slack"] = slack.New(cfg.Slack.BotToken)
			log.Info("slack notifier initialized")
		}

		// Initialize MCP manager
		mcpMgr, err := mcp.NewManager(ctx, cfg.MCP.ConfigPath)
		if err != nil {
			return err
		}
		defer mcpMgr.Close()
		log.Infof("MCP manager initialized: %d tools loaded", len(mcpMgr.Tools()))

		warnStaleToolFilters(scenarioRepo, mcpMgr)

		// Create LLM provider
		var provider llm.Provider
		switch strings.ToLower(cfg.LLM.Provider) {
		case "anthropic", "claude":
			provider = anthropic.New(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.ReasoningEffort)
		case "openai", "":
			provider = openai.New(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.ReasoningEffort)
		default:
			return fmt.Errorf("unsupported llm provider %q (want \"openai\" or \"anthropic\")", cfg.LLM.Provider)
		}
		log.Infof("LLM provider initialized: provider=%s model=%s", cfg.LLM.Provider, cfg.LLM.Model)

		// Create agent
		ag := agent.New(provider, mcpMgr, cfg.LLM, promptRepo)

		// Create alert deduplicator and queue
		cooldown, err := time.ParseDuration(cfg.AlertCooldown)
		if err != nil {
			return fmt.Errorf("invalid alert_cooldown %q: %w", cfg.AlertCooldown, err)
		}

		var (
			dedup    webhook.DedupStore
			rdb      *redis.Client
			useRedis = cfg.Redis.URL != ""
		)

		if useRedis {
			redisOpt, err := redis.ParseURL(cfg.Redis.URL)
			if err != nil {
				return fmt.Errorf("invalid redis URL %q: %w", cfg.Redis.URL, err)
			}
			rdb = redis.NewClient(redisOpt)
			if err := rdb.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("redis ping failed: %w", err)
			}
			defer rdb.Close() //nolint:errcheck
			dedup = webhook.NewRedisDedup(rdb, cooldown)
			log.Infof("redis dedup initialized: addr=%s cooldown=%s", redisOpt.Addr, cooldown)
		} else {
			dedup = webhook.NewDeduplicator(cooldown)
			log.Infof("in-memory dedup initialized: cooldown=%s", cooldown)
		}

		// Create webhook handler
		defaultChannels := scenario.Channels{
			Telegram: cfg.Telegram.DefaultChannel,
			Slack:    cfg.Slack.DefaultChannel,
		}
		wh := webhook.NewHandler(scenarioRepo, ag, notifiers, defaultChannels, dedup)

		var queue webhook.JobQueue
		if ag != nil {
			if useRedis {
				rq, err := webhook.NewRedisQueue(ctx, rdb, wh, scenarioRepo)
				if err != nil {
					return fmt.Errorf("init redis queue: %w", err)
				}
				queue = rq
				webhook.WarnLegacyQueue(ctx, rdb)
				log.Info("redis stream queue initialized")
			} else {
				queue = webhook.NewQueue(wh, 100)
				log.Info("in-memory queue initialized")
			}
			wh.SetQueue(queue)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("GET /healthz", handleHealthz)
		mux.HandleFunc("POST /webhook", wh.HandleWebhook)

		webUI := web.NewHandler(scenarioRepo, promptRepo, mcpMgr)
		webUI.RegisterRoutes(mux)
		log.Info("web UI initialized")

		log.Infof("webhook endpoint: http://%s/webhook", cfg.HTTP.Addr)

		srv := server.New(cfg.HTTP.Addr, mux)
		if err := srv.Run(ctx); err != nil {
			log.Errorf("server stopped with error: %v", err)
			return err
		}

		if queue != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()
			queue.Shutdown(shutdownCtx)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// warnStaleToolFilters logs a warning for each scenario that references
// tool names not present in the loaded MCP tools. This helps operators
// update scenario filters after tool names have been namespaced.
func warnStaleToolFilters(scenarios storage.ScenarioRepository, mgr *mcp.Manager) {
	list, err := scenarios.List()
	if err != nil {
		log.Errorf("failed to list scenarios for tool filter check: %v", err)
		return
	}

	known := mgr.ToolNames()
	for _, sc := range list {
		for _, tool := range sc.Tools {
			if _, ok := known[tool]; !ok {
				log.Warnf("scenario %q references unknown tool %q — update the tools filter", sc.Name, tool)
			}
		}
	}
}
