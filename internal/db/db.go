package db

import (
	"context"
	"io/fs"

	"github.com/robherley/gw-bot/internal/db/sqlgen"
)

type DB interface {
	sqlgen.Querier

	Close() error
	Migrate(context.Context, fs.FS) error
}
