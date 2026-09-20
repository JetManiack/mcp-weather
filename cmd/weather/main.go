package main

import (
	"context"
	"encoding/hex"
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
	"github.com/JetManiack/mcp-weather/internal/humanauth"
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
	listenAddr    string
	dbDSN         string
	timeout       time.Duration
	authStub      bool
	oidcIssuer    string
	oidcClientID  string
	oidcClientSecret string
	oidcPublicURL string
	oidcAdminGroup string
	oidcEncryptionKey string
}

func buildHandler(ctx context.Context, db *gorm.DB, cfg config) (http.Handler, error) {
	var provider humanauth.Provider
	var oidcHandlers humanauth.OIDCHandlers

	if cfg.authStub {
		provider = humanauth.StubProvider{}
	} else {
		keyBytes, err := hex.DecodeString(cfg.oidcEncryptionKey)
		if err != nil || len(keyBytes) != 32 {
			return nil, errors.New("OIDC_ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
		}
		p, h, err := humanauth.NewOIDCHandlers(ctx, db, humanauth.OIDCConfig{
			Issuer:        cfg.oidcIssuer,
			ClientID:      cfg.oidcClientID,
			ClientSecret:  cfg.oidcClientSecret,
			PublicURL:     cfg.oidcPublicURL,
			AdminGroup:    cfg.oidcAdminGroup,
			EncryptionKey: keyBytes,
		})
		if err != nil {
			return nil, err
		}
		provider = p
		oidcHandlers = h
	}

	client := weathertools.NewClient(cfg.timeout)
	r := chi.NewRouter()
	r.Get("/livez", health.LivezHandler())
	r.Get("/readyz", health.ReadyzHandler(db))

	if !cfg.authStub {
		r.Get("/auth/login", oidcHandlers.Login)
		r.Get("/auth/callback", oidcHandlers.Callback)
		r.Post("/auth/logout", oidcHandlers.Logout)
	}

	r.Mount("/mcp", mcpserver.Handler(db, []mcpserver.ToolRegistrar{
		weathertools.NewRegistrar(client),
	}))
	r.Mount("/api", restapi.NewHandler(db, provider))
	r.Mount("/", frontend.Handler())
	return r, nil
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
			&cli.BoolFlag{
				Name:  "auth-stub",
				Usage: "skip OIDC and grant every request admin access (dev only)",
			},
			&cli.StringFlag{
				Name:    "oidc-issuer",
				Usage:   "Keycloak issuer URL",
				Sources: cli.EnvVars("OIDC_ISSUER"),
			},
			&cli.StringFlag{
				Name:    "oidc-client-id",
				Usage:   "OAuth2 client ID",
				Sources: cli.EnvVars("OIDC_CLIENT_ID"),
			},
			&cli.StringFlag{
				Name:    "oidc-client-secret",
				Usage:   "OAuth2 client secret",
				Sources: cli.EnvVars("OIDC_CLIENT_SECRET"),
			},
			&cli.StringFlag{
				Name:    "oidc-public-url",
				Usage:   "public base URL of this server (e.g. https://weather.example.com)",
				Sources: cli.EnvVars("OIDC_PUBLIC_URL"),
			},
			&cli.StringFlag{
				Name:    "oidc-admin-group",
				Value:   "mcp/weather/admin",
				Usage:   "Keycloak group whose members receive the admin role",
				Sources: cli.EnvVars("OIDC_ADMIN_GROUP"),
			},
			&cli.StringFlag{
				Name:    "oidc-encryption-key",
				Usage:   "64-char hex AES-256 key for encrypting stored refresh tokens",
				Sources: cli.EnvVars("OIDC_ENCRYPTION_KEY"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config{
				listenAddr:        cmd.String("listen-addr"),
				dbDSN:             cmd.String("db-dsn"),
				timeout:           cmd.Duration("timeout"),
				authStub:          cmd.Bool("auth-stub"),
				oidcIssuer:        cmd.String("oidc-issuer"),
				oidcClientID:      cmd.String("oidc-client-id"),
				oidcClientSecret:  cmd.String("oidc-client-secret"),
				oidcPublicURL:     cmd.String("oidc-public-url"),
				oidcAdminGroup:    cmd.String("oidc-admin-group"),
				oidcEncryptionKey: cmd.String("oidc-encryption-key"),
			}
			if !cfg.authStub {
				missing := []string{}
				if cfg.oidcIssuer == "" {
					missing = append(missing, "--oidc-issuer / OIDC_ISSUER")
				}
				if cfg.oidcClientID == "" {
					missing = append(missing, "--oidc-client-id / OIDC_CLIENT_ID")
				}
				if cfg.oidcClientSecret == "" {
					missing = append(missing, "--oidc-client-secret / OIDC_CLIENT_SECRET")
				}
				if cfg.oidcPublicURL == "" {
					missing = append(missing, "--oidc-public-url / OIDC_PUBLIC_URL")
				}
				if cfg.oidcEncryptionKey == "" {
					missing = append(missing, "--oidc-encryption-key / OIDC_ENCRYPTION_KEY")
				}
				if len(missing) > 0 {
					return errors.New("missing required flags (or set --auth-stub for dev): " + joinStrings(missing))
				}
			}
			return run(ctx, cfg)
		},
	}
}

func joinStrings(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return err
	}

	handler, err := buildHandler(ctx, db, cfg)
	if err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return err
	}

	ah.swap(handler)
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
