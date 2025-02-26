package db

import (
	"context"
	"database/sql"
	_ "embed"
	"io/fs"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/robherley/gw-bot/internal/db/sqlgen"
)

type SQLite struct {
	*sql.DB
	*sqlgen.Queries
}

func NewSQLite(dsn string) (DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	return &SQLite{db, sqlgen.New(db)}, nil
}

func (s *SQLite) Migrate(ctx context.Context, migrations fs.FS) error {
	goose.SetBaseFS(migrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	return goose.Up(s.DB, "database/migrations")
}
