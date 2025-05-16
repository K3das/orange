package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/K3das/orange/store/db"
	"github.com/golang-migrate/migrate/v4"
	migratePgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	log *zap.Logger

	conn *pgxpool.Pool

	*db.Queries
}

func NewStore(ctx context.Context, parentLogger *zap.Logger) *Store {
	s := &Store{}
	s.log = parentLogger.Named("store")

	return s
}

func (s *Store) Connect(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("opening postgres: %w", err)
	}

	mFS, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("creating iofs driver: %w", err)
	}

	stdDB := stdlib.OpenDBFromPool(pool)
	defer stdDB.Close()

	mDriver, err := migratePgx.WithInstance(stdDB, &migratePgx.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver instance: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", mFS, "sqlite3", mDriver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}
	if err := m.Up(); err == migrate.ErrNoChange {
		s.log.Info("migrations done (no change)")
	} else if err != nil {
		return fmt.Errorf("running migrations: %w", err)
	} else {
		s.log.Info("migrations done")
	}

	s.Queries = db.New(pool)
	s.conn = pool

	return nil
}

func (s *Store) Close() {
	s.conn.Close()
}
