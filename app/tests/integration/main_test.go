//go:build integration

// Package integration runs the whole service (HTTP, usecases, storage) against a real PostgreSQL.
// It needs Docker. Run it like this:
//
//	go test -tags integration -v ./app/tests/integration/
package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dqso/coin-transfer-service/app/internal/repository/postgres"
	transporthttp "github.com/dqso/coin-transfer-service/app/internal/transport/http"
	"github.com/dqso/coin-transfer-service/app/internal/usecase"
)

const (
	postgresImage = "postgres:18-alpine3.24"
	// templateDatabase has all migrations and the seed. Every test gets its own copy of it.
	templateDatabase = "template_app"
	migrationsDir    = "../../../migrations"
)

var (
	// connString points to the PostgreSQL container. Tests change only the database name in it.
	connString string
	// createDatabaseMu makes copies of the template one by one:
	// PostgreSQL cannot copy a database while it is used by another copy.
	createDatabaseMu sync.Mutex
	databaseCounter  atomic.Int64
)

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase(templateDatabase),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres container: %v\n", err)
		}
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		return 1
	}

	connString, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "get connection string: %v\n", err)
		return 1
	}

	if err := migrateTemplate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "migrate template database: %v\n", err)
		return 1
	}

	return m.Run()
}

// migrateTemplate applies the migrations from the project in the same way as the migrate service does.
func migrateTemplate(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return err
	}
	// The template must have no open connections, otherwise it cannot be copied.
	defer conn.Close(ctx)

	migrator, err := migrate.NewMigrator(ctx, conn, "public.schema_version")
	if err != nil {
		return err
	}
	if err := migrator.LoadMigrations(os.DirFS(migrationsDir)); err != nil {
		return err
	}
	return migrator.Migrate(ctx)
}

// app is the service with its own clean database: the seed accounts and nothing else.
type app struct {
	t    *testing.T
	url  string
	pool *pgxpool.Pool
}

func newApp(t *testing.T) *app {
	t.Helper()
	ctx := t.Context()

	database := fmt.Sprintf("test_%d", databaseCounter.Add(1))
	createDatabase(t, database)

	config, err := pgxpool.ParseConfig(connString)
	require.NoError(t, err, "parse connection string")
	config.ConnConfig.Database = database
	pool, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err, "create database pool")
	t.Cleanup(pool.Close)

	log := slog.New(slog.DiscardHandler)
	handler, err := transporthttp.NewHandler(log, usecase.NewUsecase(postgres.NewStorage(pool)), pool.Ping)
	require.NoError(t, err, "create http handler")
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &app{t: t, url: server.URL, pool: pool}
}

func createDatabase(t *testing.T, name string) {
	t.Helper()
	createDatabaseMu.Lock()
	defer createDatabaseMu.Unlock()

	config, err := pgx.ParseConfig(connString)
	require.NoError(t, err, "parse connection string")
	// Connect to the default database, because the template must stay free.
	config.Database = "postgres"
	conn, err := pgx.ConnectConfig(t.Context(), config)
	require.NoError(t, err, "connect to postgres")
	defer conn.Close(t.Context())

	query := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", pgx.Identifier{name}.Sanitize(), templateDatabase)
	_, err = conn.Exec(t.Context(), query)
	require.NoError(t, err, "create database %s", name)
}
