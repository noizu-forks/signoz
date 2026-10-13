package sqlmigration

import (
	"context"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlstore"
)

type addIngestionKeys struct {
	sqlstore sqlstore.SQLStore
}

func NewAddIngestionKeysFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_ingestion_keys"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return &addIngestionKeys{sqlstore: sqlstore}, nil
	})
}

func (migration *addIngestionKeys) Register(migrations *migrate.Migrations) error {
	return migrations.Register(migration.Up, migration.Down)
}

func (migration *addIngestionKeys) Up(ctx context.Context, db *bun.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewCreateTable().
		Model((*ingestionKey)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewCreateTable().
		Model((*ingestionKeyLimit)(nil)).
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *addIngestionKeys) Down(ctx context.Context, db *bun.DB) error {
	return nil
}

type ingestionKey struct {
	bun.BaseModel `bun:"table:ingestion_keys"`

	ID          string `bun:"id,pk,type:text"`
	OrgID       string `bun:"org_id,notnull,type:text"`
	Name        string `bun:"name,notnull,type:text"`
	Value       string `bun:"value,notnull,type:text"`
	Tags        []byte `bun:"tags,type:blob"`
	ExpiresAt   int64  `bun:"expires_at,type:integer"`
	CreatedAt   int64  `bun:"created_at,notnull,type:integer"`
	UpdatedAt   int64  `bun:"updated_at,notnull,type:integer"`
}

type ingestionKeyLimit struct {
	bun.BaseModel `bun:"table:ingestion_key_limits"`

	ID        string `bun:"id,pk,type:text"`
	KeyID     string `bun:"key_id,notnull,type:text"`
	Signal    string `bun:"signal,notnull,type:text"`
	Config    []byte `bun:"config,notnull,type:blob"`
	Tags      []byte `bun:"tags,type:blob"`
	CreatedAt int64  `bun:"created_at,notnull,type:integer"`
	UpdatedAt int64  `bun:"updated_at,notnull,type:integer"`
}