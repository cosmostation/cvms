package common

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/jinzhu/inflection"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/schema"
)

type IndexerDB struct {
	*bun.DB
	RetentionPeriod string
}

type IndexerDBConfig struct {
	Host     string `toml:"host"`
	Database string `toml:"database"`
	Port     string `toml:"port"`
	User     string `toml:"user"`
	Password string `toml:"password"`
	Timeout  int64  `toml:"db_timeout"`
}

func NewIndexerDB(cfg IndexerDBConfig) (*IndexerDB, error) {
	if cfg == (IndexerDBConfig{}) || cfg.User == "" {
		return nil, errors.New("you should provide DB envs like DB_HOST, DB_PORT...")
	}

	timeout := cfg.Timeout
	if cfg.Timeout == 0 {
		timeout = 10
	}

	// setup variables for db connection
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
	)
	timeoutDuration := time.Second * time.Duration(timeout)

	// initiate db
	sqldb := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithTimeout(timeoutDuration),
		pgdriver.WithDialTimeout(timeoutDuration),
		pgdriver.WithReadTimeout(timeoutDuration),
	))

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		// disable the hook
		bundebug.WithEnabled(false),
		bundebug.WithVerbose(false),
		// BUNDEBUG=1 logs failed queries
		// BUNDEBUG=2 logs all queries
		bundebug.FromEnv("BUNDEBUG"),
	))

	err := db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping: %s", err)
	}

	schema.SetTableNameInflector(inflection.Singular)

	return &IndexerDB{
		DB: db,
	}, nil
}

func (db *IndexerDB) SetRetentionTime(retentionPeriod string) {
	db.RetentionPeriod = retentionPeriod
}

// TODO: currently, we don't use this helper.DB
func (db *IndexerDB) CloseConn() error {
	return db.DB.Close()
}

func NewTestIndexerDB(dsn string) (*IndexerDB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	// NOTE: No need to ping in test mode
	// err := db.Ping()
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to ping: %s", err)
	// }

	schema.SetTableNameInflector(inflection.Singular)

	return &IndexerDB{
		DB: db,
	}, nil
}

// db is creating by Postgre Application in Mac
func NewTestLoaclIndexerDB(tempDBName string) (*IndexerDB, error) {
	// setup
	cmd := exec.Command("go", "env", "GOMOD")
	out, _ := cmd.Output()
	rootPath := strings.Split(string(out), "/go.mod")[0]
	dirPath := filepath.Join(rootPath, "./docker/postgres/schema")
	dsn := "postgres://cvms:mysecretpassword@localhost:5432/postgres?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	err := sqldb.Ping()
	if err != nil {
		panic("failed to connect local db")
	}

	// Clean up
	_, err = sqldb.Exec(fmt.Sprintf("DROP DATABASE %s;", tempDBName))
	if err != nil {
		log.Fatalf("%s", err)
		panic("failed to clean up temp db")
	}

	// Create a temporary database
	_, err = sqldb.Exec(fmt.Sprintf("CREATE DATABASE %s;", tempDBName))
	if err != nil {
		log.Println(err)
	}

	// Connect to the temporary database
	dsn = fmt.Sprintf("postgres://cvms:mysecretpassword@localhost:5432/%s?sslmode=disable", tempDBName)
	sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	sqldb.Exec(fmt.Sprintf("DROP DATABASE %s;", tempDBName)) // Clean up

	// Initialize Bun with the temporary database
	db := bun.NewDB(sqldb, pgdialect.New())
	db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(false)))

	// Initialize schema from a directory
	if err := initSchemaFromDir(db, dirPath); err != nil {
		log.Fatalf("failed to initialize schema: %v", err)
	}
	log.Println("Schema initialized successfully!")

	schema.SetTableNameInflector(inflection.Singular)
	return &IndexerDB{
		DB: db,
	}, nil
}

func initSchemaFromDir(db *bun.DB, dir string) error {
	// Read all files in the schema directory
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read schema directory: %w", err)
	}

	for _, file := range files {
		// Read the SQL file content
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file, err)
		}

		// Execute the SQL content
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("failed to execute schema file %s: %w", file, err)
		}
	}
	return nil
}
