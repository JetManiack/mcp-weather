package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/urfave/cli/v3"
	"gorm.io/gorm"

	"github.com/JetManiack/mcp-weather/internal/frontend"
	"github.com/JetManiack/mcp-weather/internal/health"
	"github.com/JetManiack/mcp-weather/internal/mcpserver"
	"github.com/JetManiack/mcp-weather/internal/restapi"
	"github.com/JetManiack/mcp-weather/internal/storage"
	weathertools "github.com/JetManiack/mcp-weather/internal/tools/weather"
)

// version is stamped at build time by the Makefile (-X main.version=...).
var version = "dev"

// atomicHandler lets the serve goroutine start immediately with a placeholder
// and swap in the real handler once the database is ready.
type atomicHandler struct {
	h atomic.Pointer[http.Handler]
}

func (a *atomicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	(*a.h.Load()).ServeHTTP(w, r)
}

func (a *atomicHandler) swap(h http.Handler) {
	a.h.Store(&h)
}

// placeholder replies 503 with a Retry-After hint while the server is warming up.
func placeholder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /livez must always answer during warm-up so load-balancers don't drop us.
		if r.URL.Path == "/livez" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Retry-After", "5")
		http.Error(w, "server starting", http.StatusServiceUnavailable)
	})
}

type config struct {
	listenAddr string
	dbDSN      string
	timeout    time.Duration
	adminToken string
}

func buildHandler(db *gorm.DB, cfg config) http.Handler {
	client := weathertools.NewClient(cfg.timeout)
	r := chi.NewRouter()
	r.Get("/livez", health.LivezHandler())
	r.Get("/readyz", health.ReadyzHandler(db))
	r.Mount("/mcp", mcpserver.Handler(db, []mcpserver.ToolRegistrar{
		weathertools.NewRegistrar(client),
	}))
	r.Mount("/api", restapi.Handler(db, cfg.adminToken))
	r.Mount("/", frontend.Handler())
	return r
}

func newRootCommand() *cli.Command {
	return &cli.Command{
		Name:    "weather",
		Usage:   "MCP server exposing weather tools to AI agents via Open-Meteo",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "listen-addr",
				Value:   ":8080",
				Usage:   "address the HTTP server listens on",
				Sources: cli.EnvVars("LISTEN_ADDR"),
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Value:   10 * time.Second,
				Usage:   "per-request timeout for Open-Meteo API calls",
				Sources: cli.EnvVars("REQUEST_TIMEOUT"),
			},
			&cli.StringFlag{
				Name:    "db-dsn",
				Value:   "data/weather.db",
				Usage:   "database DSN (SQLite file path or postgres:// URL)",
				Sources: cli.EnvVars("DB_DSN"),
			},
			&cli.StringFlag{
				Name:    "admin-token",
				Usage:   "bearer token required by the admin UI REST API",
				Sources: cli.EnvVars("ADMIN_TOKEN"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config{
				listenAddr: cmd.String("listen-addr"),
				dbDSN:      cmd.String("db-dsn"),
				timeout:    cmd.Duration("timeout"),
				adminToken: cmd.String("admin-token"),
			}
			if cfg.adminToken == "" {
				return errors.New("--admin-token / ADMIN_TOKEN is required; set a strong random token to protect the admin API")
			}
			return run(ctx, cfg)
		},
	}
}

// run starts the HTTP server immediately with a placeholder handler, then
// connects to the database and atomically swaps in the real handler.
func run(ctx context.Context, cfg config) error {
	ah := &atomicHandler{}
	ph := placeholder()
	ah.swap(ph)

	srv := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           ah,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// WriteTimeout is intentionally generous so MCP streaming responses
		// are not cut by the transport before delivery completes.
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "addr", cfg.listenAddr, "version", version)
		serveErr <- srv.ListenAndServe()
	}()

	// Open DB (with implicit retry via GORM + busy_timeout for SQLite).
	db, err := storage.Open(cfg.dbDSN)
	if err != nil {
		// Shut down the server we just started before returning the error.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return err
	}

	// Swap in the real handler now that the DB is ready.
	ah.swap(buildHandler(db, cfg))
	slog.Info("server ready", "addr", cfg.listenAddr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func main() {
	// SIGTERM is how a container runtime asks for a graceful stop; without
	// this, in-flight tool calls are cut mid-response on every rollout.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCommand().Run(ctx, os.Args); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}
