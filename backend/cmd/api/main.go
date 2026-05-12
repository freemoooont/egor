// Package main is the API binary entry point. It wires the layered components
// — domain, application, infrastructure, interfaces — into a runnable HTTP
// server. See backend/CLAUDE.md for the layering rules.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	appdecks "github.com/micocards/api/internal/application/decks"
	appiam "github.com/micocards/api/internal/application/iam"
	apppractice "github.com/micocards/api/internal/application/practice"
	"github.com/micocards/api/internal/infrastructure/ai"
	"github.com/micocards/api/internal/infrastructure/auth"
	"github.com/micocards/api/internal/infrastructure/clock"
	"github.com/micocards/api/internal/infrastructure/config"
	"github.com/micocards/api/internal/infrastructure/events"
	"github.com/micocards/api/internal/infrastructure/idgen"
	"github.com/micocards/api/internal/infrastructure/logger"
	pgsql "github.com/micocards/api/internal/infrastructure/postgres"
	"github.com/micocards/api/internal/infrastructure/postgres/decksrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/iamrepo"
	"github.com/micocards/api/internal/infrastructure/postgres/migrate"
	"github.com/micocards/api/internal/infrastructure/postgres/outbox"
	"github.com/micocards/api/internal/infrastructure/postgres/practicerepo"
	httproot "github.com/micocards/api/internal/interfaces/http"
	httpdecks "github.com/micocards/api/internal/interfaces/http/decks"
	httpiam "github.com/micocards/api/internal/interfaces/http/iam"
	"github.com/micocards/api/internal/interfaces/http/middleware"
	httppractice "github.com/micocards/api/internal/interfaces/http/practice"
	domiam "github.com/micocards/api/internal/domain/iam"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api: shutdown", slog.Any("err", err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	log := logger.New(logger.FromEnv())
	slog.SetDefault(log)
	log.Info("api: booting", slog.String("config", cfg.String()))

	rootCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(rootCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(rootCtx); err != nil {
		log.Warn("api: db unreachable on boot — continuing", slog.Any("err", err))
	}

	if err := applyMigrations(rootCtx, pool, log); err != nil {
		log.Warn("api: migrations failed — continuing", slog.Any("err", err))
	}

	signer, err := auth.NewAccessTokenSigner(cfg.JWTSecret, cfg.AccessTokenTTL)
	if err != nil {
		return err
	}
	hasher := auth.NewBcryptHasher(cfg.BcryptCost, cfg.MinPasswordLength)
	refreshMint := auth.NewRefreshMinter()
	refreshHasher := refreshHasherAdapter{}

	// AI provider: empty key → NotConfigured; non-empty → OpenAIStub (still
	// returns ErrAINotConfigured today). The handler also gates on AIEnabled
	// before calling the use case.
	var aiProv appdecks.AIProvider = ai.NotConfigured{}
	if cfg.AIAPIKey != "" {
		aiProv = ai.OpenAIStub{APIKey: cfg.AIAPIKey, Model: cfg.AIModel}
	}

	uow := pgsql.NewUnitOfWork(pool)
	clk := clock.System{}
	ids := idgen.UUID{}

	bus := events.NewPublisher()
	iamOutbox, _ := outbox.New(pool, "iam")
	decksOutbox, _ := outbox.New(pool, "decks")
	practiceOutbox, _ := outbox.New(pool, "practice")
	iamPub := events.NewIAMPublisher(bus, iamOutbox)
	decksPub := events.NewDecksPublisher(bus, decksOutbox)
	practicePub := events.NewPracticePublisher(bus, practiceOutbox)

	// Repositories.
	usersRepo := iamrepo.NewUsers(pool)
	refreshRepo := iamrepo.NewRefreshTokens(pool, nil)
	idemRepo := iamrepo.NewIdempotencyKeys(pool)
	decksRepo := decksrepo.NewDecks(pool)
	sessionsRepo := practicerepo.NewSessions(pool)
	progressRepo := practicerepo.NewUserDeckProgresses(pool)

	// IAM use cases.
	iamHandlers := httpiam.New(httpiam.Deps{
		Register: appiam.RegisterUser{
			Users: usersRepo, RefreshTokens: refreshRepo, Hasher: hasher, IDs: ids, Clock: clk,
			Tokens: refreshMint, AccessSigner: signer, UoW: uow, Events: iamPub,
		},
		Login: appiam.LoginUser{
			Users: usersRepo, RefreshTokens: refreshRepo, Hasher: hasher, IDs: ids, Clock: clk,
			Tokens: refreshMint, AccessSigner: signer, UoW: uow, Events: iamPub,
		},
		Refresh: appiam.RefreshAccessToken{
			RefreshTokens: refreshRepo, IDs: ids, Clock: clk,
			Tokens: refreshMint, AccessSigner: signer, UoW: uow, Events: iamPub, Hasher: refreshHasher,
		},
		Logout: appiam.LogoutUser{
			RefreshTokens: refreshRepo, Hasher: refreshHasher, Clock: clk, UoW: uow, Events: iamPub,
		},
		GetMe:         appiam.GetCurrentUser{Users: usersRepo},
		UpdateProfile: appiam.UpdateProfile{Users: usersRepo, UoW: uow},
		ChangePassword: appiam.ChangePassword{
			Users: usersRepo, RefreshTokens: refreshRepo, Hasher: hasher, Clock: clk, UoW: uow, Events: iamPub,
		},
		DBPool: pool,
	})

	// Decks use cases.
	decksHandlers := httpdecks.New(httpdecks.Deps{
		List:       appdecks.ListUserDecks{Decks: decksRepo},
		Create:     appdecks.CreateDeck{Decks: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: decksPub},
		Get:        appdecks.GetDeck{Decks: decksRepo},
		Rename:     appdecks.RenameDeck{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		Delete:     appdecks.DeleteDeck{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		AddCard:    appdecks.AddCard{Decks: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: decksPub},
		EditCard:   appdecks.EditCard{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		RemoveCard: appdecks.RemoveCard{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		Reorder:    appdecks.ReorderCards{Decks: decksRepo, Clock: clk, UoW: uow, Events: decksPub},
		Generate: appdecks.GenerateDeckWithAI{
			AI: aiProv, IDs: ids, Clock: clk, Events: decksPub,
		},
		AIEnabled: aiProv.IsConfigured(),
	})

	// Practice use cases.
	practiceHandlers := httppractice.New(httppractice.Deps{
		Start: apppractice.StartSession{
			Sessions: sessionsRepo, Snapshots: decksRepo, IDs: ids, Clock: clk, UoW: uow, Events: practicePub,
		},
		Rate: apppractice.RateCard{
			Sessions: sessionsRepo, Clock: clk, UoW: uow, Events: practicePub,
		},
		Finish: apppractice.FinishSession{
			Sessions: sessionsRepo, Clock: clk, UoW: uow, Events: practicePub,
		},
		Results:  apppractice.GetResults{Sessions: sessionsRepo},
		Progress: apppractice.GetUserDeckProgress{Progress: progressRepo},
	})

	authMW := middleware.NewAuth(signer)
	router := httproot.NewRouter(httproot.Deps{
		IAM: iamHandlers, Decks: decksHandlers, Practice: practiceHandlers,
		Auth: authMW, Idem: idemRepo, Logger: log, CORS: cfg.CORSOrigins,
		DBPool: pool, StartTime: time.Now().UTC(),
	})

	// Outbox dispatcher (best-effort — logs and continues on errors).
	go runOutboxDispatcher(rootCtx, log,
		outbox.NewDispatcher(iamOutbox, busForwarder{bus}, 100),
		outbox.NewDispatcher(decksOutbox, busForwarder{bus}, 100),
		outbox.NewDispatcher(practiceOutbox, busForwarder{bus}, 100),
		time.Duration(cfg.OutboxTickSeconds)*time.Second,
	)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api: listening", slog.String("addr", cfg.ListenAddr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-rootCtx.Done():
	}
	log.Info("api: shutdown")
	shutdownCtx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer scancel()
	return srv.Shutdown(shutdownCtx)
}

// busForwarder adapts events.Publisher to the outbox.EventForwarder port.
type busForwarder struct{ bus *events.Publisher }

// Forward replays the persisted event back through the in-process bus. The
// bus dispatches it to any registered handler synchronously.
func (b busForwarder) Forward(ctx context.Context, eventName string, payload []byte, _ string) error {
	// We only need to fan out by name; the payload is opaque to the bus.
	// Wrap it in a minimal Event implementation.
	_ = payload
	return b.bus.Publish(ctx, replayEvent{name: eventName})
}

type replayEvent struct{ name string }

// Name returns the event's wire name.
func (r replayEvent) Name() string { return r.name }

// applyMigrations runs every migration body once; idempotent thanks to IF
// NOT EXISTS in every CREATE.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	root := findMigrationsRoot()
	if root == "" {
		log.Warn("api: migrations root not found — skipping")
		return nil
	}
	log.Info("api: applying migrations", slog.String("root", root))
	return migrate.ApplyAll(ctx, pool, root)
}

// findMigrationsRoot looks for backend/migrations relative to the current
// process working directory and the source-file path. Convenient for both
// `go run ./cmd/api` and built binaries.
func findMigrationsRoot() string {
	candidates := []string{"backend/migrations", "migrations", "../migrations"}
	if _, here, _, ok := runtime.Caller(0); ok {
		// .../backend/cmd/api/main.go → .../backend/migrations
		candidates = append(candidates, filepath.Join(filepath.Dir(here), "..", "..", "migrations"))
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			return abs
		}
	}
	return ""
}

// runOutboxDispatcher loops on a ticker and fans every per-context dispatcher
// once per tick. Errors are logged and ignored — the next tick retries.
func runOutboxDispatcher(ctx context.Context, log *slog.Logger, ds ...any) {
	// flatten variadic of dispatchers
	dispatchers := make([]*outbox.Dispatcher, 0, len(ds)-1)
	var period time.Duration
	for _, d := range ds {
		switch v := d.(type) {
		case *outbox.Dispatcher:
			dispatchers = append(dispatchers, v)
		case time.Duration:
			period = v
		}
	}
	if period <= 0 {
		period = 30 * time.Second
	}
	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, d := range dispatchers {
				if _, err := d.DispatchOnce(ctx); err != nil {
					log.Warn("api: outbox dispatch failed", slog.Any("err", err))
				}
			}
		}
	}
}

// refreshHasherAdapter satisfies the iam application's RefreshHasher port by
// reusing the auth package's HashOpaque function.
type refreshHasherAdapter struct{}

// HashOpaque delegates to auth.HashOpaque.
func (refreshHasherAdapter) HashOpaque(plaintext string) string { return auth.HashOpaque(plaintext) }

var _ = domiam.ErrUnauthorized // ensure domain pkg is loaded (errorMapper test target).
