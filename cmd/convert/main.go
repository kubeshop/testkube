// Command convert migrates Testkube control-plane data from MongoDB to
// PostgreSQL.
//
// It is meant to be run once, either as a Kubernetes Job during a cutover or
// directly against port-forwarded databases:
//
//	convert --mongo-dsn mongodb://localhost:27017 --mongo-db testkube \
//	        --postgres-dsn 'postgres://user:pass@localhost:5432/backend?sslmode=disable'
//
// Runs are resumable and safe to repeat: progress is checkpointed in the same
// transaction as the data it describes, so an interrupted run continues where it
// stopped rather than starting over or double-writing.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/convert"
	postgresdb "github.com/kubeshop/testkube/pkg/database/postgres"
	postgresmigrations "github.com/kubeshop/testkube/pkg/database/postgres/migrations"
	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/repository/storage"
	"github.com/kubeshop/testkube/pkg/version"
)

// Set via -ldflags at build time, matching the other binaries in this repo.
var (
	commit  string
	builtBy string
	date    string
)

// shutdownTimeout bounds the cleanup that runs after the main context is done,
// so a hung close cannot keep a Job's pod alive indefinitely.
const shutdownTimeout = 10 * time.Second

type options struct {
	mongoDSN        string
	mongoDB         string
	mongoDBType     string
	mongoAllowTLS   bool
	mongoClientCert string
	mongoClientPass string
	mongoCAFile     string
	postgresDSN     string
	skipDBCreation  bool
	skipMigrations  bool
	batchSize       int
	readBatchSize   int
	dryRun          bool
	reset           bool
	resetConfirmed  bool
	skipErrors      bool
	skipTasks       []string
	verify          bool
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		// cobra has already printed the error.
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Migrate Testkube control-plane data from MongoDB to PostgreSQL",
		Long: `Migrate Testkube control-plane data from MongoDB to PostgreSQL.

Copies Test Workflow executions and their execution-number counters into the
PostgreSQL schema, so that switching the API server from API_MONGO_DSN to
API_POSTGRES_DSN preserves execution history and numbering.

Execution logs, outputs and artifacts are not touched: they live in object
storage, and only their references travel with the execution. Test workflow
definitions and triggers are Kubernetes custom resources, and cluster
configuration lives in a ConfigMap, so none of them are stored in MongoDB.

Runs are resumable. Progress is checkpointed alongside the data it describes, so
re-running after a failure continues from the last committed batch. Use --reset
to discard migrated data and start over.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return run(cmd.Context(), &o)
		},
	}

	f := cmd.Flags()
	f.StringVar(&o.mongoDSN, "mongo-dsn", env("API_MONGO_DSN", ""),
		"MongoDB connection string to read from [API_MONGO_DSN]")
	f.StringVar(&o.mongoDB, "mongo-db", env("API_MONGO_DB", "testkube"),
		"MongoDB database name [API_MONGO_DB]")
	f.StringVar(&o.mongoDBType, "mongo-db-type", env("API_MONGO_DB_TYPE", storage.TypeMongoDB),
		"MongoDB flavour: mongo or docdb [API_MONGO_DB_TYPE]")
	f.BoolVar(&o.mongoAllowTLS, "mongo-allow-tls", envBool("API_MONGO_ALLOW_TLS", false),
		"Enable TLS to MongoDB/DocumentDB [API_MONGO_ALLOW_TLS]")
	f.StringVar(&o.mongoClientCert, "mongo-ssl-client-cert-file", env("API_MONGO_SSL_CLIENT_CERT_FILE", ""),
		"Path to the MongoDB client certificate key file")
	f.StringVar(&o.mongoClientPass, "mongo-ssl-client-cert-password", env("API_MONGO_SSL_CLIENT_CERT_PASSWORD", ""),
		"Password for the MongoDB client certificate key file")
	f.StringVar(&o.mongoCAFile, "mongo-ssl-ca-file", env("API_MONGO_SSL_CA_FILE", ""),
		"Path to the MongoDB certificate authority file")

	f.StringVar(&o.postgresDSN, "postgres-dsn", env("API_POSTGRES_DSN", ""),
		"PostgreSQL connection string to write to [API_POSTGRES_DSN]")
	f.BoolVar(&o.skipDBCreation, "skip-db-creation", envBool("SKIP_DB_CREATION", false),
		"Assume the PostgreSQL database already exists [SKIP_DB_CREATION]")
	f.BoolVar(&o.skipMigrations, "skip-migrations", envBool("DISABLE_POSTGRES_MIGRATIONS", false),
		"Do not apply PostgreSQL schema migrations before migrating data [DISABLE_POSTGRES_MIGRATIONS]")

	f.IntVar(&o.batchSize, "batch-size", envInt("CONVERT_BATCH_SIZE", convert.DefaultBatchSize),
		"Executions committed per transaction [CONVERT_BATCH_SIZE]")
	f.IntVar(&o.readBatchSize, "read-batch-size", envInt("CONVERT_READ_BATCH_SIZE", convert.DefaultReadBatchSize),
		"MongoDB cursor batch size [CONVERT_READ_BATCH_SIZE]")
	f.BoolVar(&o.dryRun, "dry-run", envBool("CONVERT_DRY_RUN", false),
		"Read and serialize everything but commit nothing [CONVERT_DRY_RUN]")
	f.BoolVar(&o.reset, "reset", envBool("CONVERT_RESET", false),
		"Discard already-migrated data and checkpoints, then start over [CONVERT_RESET]")
	f.BoolVar(&o.resetConfirmed, "yes", envBool("CONVERT_RESET_CONFIRMED", false),
		"Confirm --reset without prompting [CONVERT_RESET_CONFIRMED]")
	f.BoolVar(&o.skipErrors, "skip-errors", envBool("CONVERT_SKIP_ERRORS", false),
		"Continue past documents that cannot be migrated [CONVERT_SKIP_ERRORS]")
	f.StringSliceVar(&o.skipTasks, "skip", envSlice("CONVERT_SKIP"),
		fmt.Sprintf("Tasks to leave out: %s [CONVERT_SKIP]", strings.Join(convert.AllTasks, ", ")))
	f.BoolVar(&o.verify, "verify", envBool("CONVERT_VERIFY", true),
		"Compare source and target counts after migrating [CONVERT_VERIFY]")

	return cmd
}

func run(ctx context.Context, o *options) error {
	logger := log.New().With("name", "convert", "version", version.Version)
	logger.Infow("starting conversion", "commit", commit, "builtBy", builtBy, "buildDate", date)

	if err := o.validate(); err != nil {
		return err
	}

	// Stop cleanly on SIGINT/SIGTERM. Because each batch commits with its own
	// checkpoint, an interrupt loses at most the batch in flight.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoDB, err := storage.GetMongoDatabase(o.mongoDSN, o.mongoDB, o.mongoDBType, o.mongoAllowTLS, o.sslConfig())
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		// Not ctx: a Job that took a SIGTERM reaches this defer with ctx already
		// cancelled, and disconnecting on a cancelled context gives up before it
		// has closed anything. Cleanup gets its own deadline so the sockets are
		// released on the shutdown path too, which is the path that matters.
		disconnectCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := mongoDB.Client().Disconnect(disconnectCtx); err != nil {
			logger.Warnf("failed to disconnect from MongoDB: %v", err)
		}
	}()
	logger.Infof("connected to MongoDB database %q", o.mongoDB)

	pg, err := o.connectPostgres(ctx, logger)
	if err != nil {
		return err
	}
	defer pg.Close()
	logger.Info("connected to PostgreSQL")

	converter := convert.New(mongoDB, pg, logger, convert.Config{
		BatchSize:     o.batchSize,
		ReadBatchSize: o.readBatchSize,
		DryRun:        o.dryRun,
		Reset:         o.reset,
		SkipErrors:    o.skipErrors,
		Skip:          o.skipTasks,
		Verify:        o.verify,
	})

	result, err := converter.Run(ctx)
	if result != nil {
		result.PrintSummary(logger)
	}
	if err != nil {
		return err
	}

	if result.Failed() {
		// Exit non-zero so a Job is marked failed and the operator looks at the
		// report, but only after the full summary has been printed.
		return convert.ErrIncomplete
	}

	logger.Info("conversion completed successfully")
	return nil
}

func (o *options) validate() error {
	if o.mongoDSN == "" {
		return errors.New("--mongo-dsn (or API_MONGO_DSN) is required")
	}
	if o.postgresDSN == "" {
		return errors.New("--postgres-dsn (or API_POSTGRES_DSN) is required")
	}

	for _, task := range o.skipTasks {
		task = strings.TrimSpace(task)
		if task == "" {
			continue
		}
		if !slicesContainsFold(convert.AllTasks, task) {
			return fmt.Errorf("unknown task %q in --skip; valid tasks are %s",
				task, strings.Join(convert.AllTasks, ", "))
		}
	}

	// --reset destroys already-migrated rows, so require an explicit
	// acknowledgement rather than letting a stray flag wipe a populated target.
	if o.reset && !o.resetConfirmed && !o.dryRun {
		return errors.New("--reset discards all migrated data; pass --yes to confirm")
	}

	return nil
}

// sslConfig returns the MongoDB TLS material, or nil when none was given.
//
// This deliberately takes file paths rather than reading a Kubernetes secret the
// way the API server's commons.MustGetMongoDatabase does: the convert tool has
// no cluster client, so callers must mount any required TLS material and pass
// the file paths via flags/env vars.
func (o *options) sslConfig() *storage.MongoSSLConfig {
	if o.mongoClientCert == "" && o.mongoCAFile == "" {
		return nil
	}
	return &storage.MongoSSLConfig{
		SSLClientCertificateKeyFile:         o.mongoClientCert,
		SSLClientCertificateKeyFilePassword: o.mongoClientPass,
		SSLCertificateAuthoritiyFile:        o.mongoCAFile,
	}
}

// connectPostgres opens the pool and brings the schema up to date. The tool runs
// the migrations itself so it can be pointed at a fresh database before the API
// server has ever started against it.
func (o *options) connectPostgres(ctx context.Context, logger *zap.SugaredLogger) (*pgxpool.Pool, error) {
	if !o.skipDBCreation {
		if err := postgresdb.CreateDatabaseIfNotExists(ctx, o.postgresDSN); err != nil {
			return nil, fmt.Errorf("failed to create the PostgreSQL database: %w", err)
		}
	}

	pool, err := pgxpool.New(ctx, o.postgresDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to reach PostgreSQL: %w", err)
	}

	if o.skipMigrations {
		logger.Warn("skipping PostgreSQL migrations; the target schema must already be current")
		return pool, nil
	}

	// goose speaks database/sql, so the pool is wrapped for the migration and the
	// wrapper closed again once it is done. Closing it releases the connections it
	// borrowed without touching the pool, which pgx documents explicitly and which
	// matters here because the pool is the caller's return value.
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Warnf("failed to close the migration connection: %v", err)
		}
	}()

	if err := applyMigrations(ctx, db, logger); err != nil {
		pool.Close()
		// Unlike the API server, which only warns, an out-of-date schema here is
		// fatal: the COPY statements below name columns that may not exist yet.
		return nil, err
	}

	return pool, nil
}

func applyMigrations(ctx context.Context, db *sql.DB, logger *zap.SugaredLogger) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, postgresmigrations.Fs,
		goose.WithAllowOutofOrder(true))
	if err != nil {
		return fmt.Errorf("failed to initialize the migrations provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply PostgreSQL migrations: %w", err)
	}

	if len(results) == 0 {
		logger.Info("PostgreSQL schema is already up to date")
		return nil
	}
	for _, r := range results {
		logger.Infof("applied migration %s", filepath.Base(r.Source.Path))
	}
	return nil
}

func slicesContainsFold(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.EqualFold(h, needle) {
			return true
		}
	}
	return false
}

// Flag defaults are read from the environment so that the Kubernetes Job can
// configure the tool with the same variable names the API server already uses.
func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "t", "true", "y", "yes":
		return true
	case "0", "f", "false", "n", "no":
		return false
	default:
		return fallback
	}
}

func envInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}

func envSlice(key string) []string {
	v := os.Getenv(key)
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
