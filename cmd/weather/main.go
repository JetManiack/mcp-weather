package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/JetManiack/mcp-weather/internal/mcpserver"
	"github.com/JetManiack/mcp-weather/internal/restapi"
	"github.com/JetManiack/mcp-weather/internal/storage"
	"github.com/JetManiack/mcp-weather/internal/ui"
	"github.com/JetManiack/mcp-weather/internal/weather"
)

// version is stamped at build time by the Makefile (-X main.version=...).
var version = "dev"

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
				Usage:   "SQLite file path for agent tokens and audit log (use :memory: to disable persistence)",
				Sources: cli.EnvVars("DB_DSN"),
			},
			&cli.StringFlag{
				Name:    "admin-token",
				Usage:   "bearer token required by the admin UI REST API; if empty the API is unauthenticated",
				Sources: cli.EnvVars("ADMIN_TOKEN"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			db, err := storage.Open(cmd.String("db-dsn"))
			if err != nil {
				return err
			}

			deps := mcpserver.Deps{
				Client:  weather.NewClient(cmd.Duration("timeout")),
				DB:      db,
				Version: version,
			}
			return serve(ctx, cmd.String("listen-addr"), deps, cmd.String("admin-token"))
		},
	}
}

func serve(ctx context.Context, addr string, deps mcpserver.Deps, adminToken string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/mcp", mcpserver.NewHTTPHandler(deps))
	mux.Handle("/api/", http.StripPrefix("/api", restapi.NewHandler(deps.DB, adminToken)))
	mux.Handle("/static/", ui.NewHandler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/static/index.html", http.StatusFound)
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		// WriteTimeout must exceed the API call timeout so in-flight tool
		// responses are not cut off by the transport before delivery.
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("starting server", "addr", addr, "version", version)
		serveErr <- srv.ListenAndServe()
	}()

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
