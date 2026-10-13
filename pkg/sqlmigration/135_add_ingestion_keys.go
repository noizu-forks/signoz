package sqlmigration

import (
	"context"

	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

type addIngestionKeys struct {
	sqlstore  sqlstore.SQLStore
	sqlschema sqlschema.SQLSchema
}

func NewAddIngestionKeysFactory(sqlstore sqlstore.SQLStore, sqlschema sqlschema.SQLSchema) factory.ProviderFactory[SQLMigration, Config] {
	return factory.NewProviderFactory(factory.MustNewName("add_ingestion_keys"), func(ctx context.Context, ps factory.ProviderSettings, c Config) (SQLMigration, error) {
		return &addIngestionKeys{
			sqlstore:  sqlstore,
			sqlschema: sqlschema,
		}, nil
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

	sqls := [][]byte{}

	tableSQLs := migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "ingestion_keys",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "org_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "name", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "value", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "tags", DataType: sqlschema.DataTypeBytea, Nullable: true},
			{Name: "expires_at", DataType: sqlschema.DataTypeInteger, Nullable: true},
			{Name: "created_at", DataType: sqlschema.DataTypeInteger, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeInteger, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
	})
	sqls = append(sqls, tableSQLs...)

	tableSQLs = migration.sqlschema.Operator().CreateTable(&sqlschema.Table{
		Name: "ingestion_key_limits",
		Columns: []*sqlschema.Column{
			{Name: "id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "key_id", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "signal", DataType: sqlschema.DataTypeText, Nullable: false},
			{Name: "config", DataType: sqlschema.DataTypeBytea, Nullable: false},
			{Name: "tags", DataType: sqlschema.DataTypeBytea, Nullable: true},
			{Name: "created_at", DataType: sqlschema.DataTypeInteger, Nullable: false},
			{Name: "updated_at", DataType: sqlschema.DataTypeInteger, Nullable: false},
		},
		PrimaryKeyConstraint: &sqlschema.PrimaryKeyConstraint{
			ColumnNames: []sqlschema.ColumnName{"id"},
		},
		ForeignKeyConstraints: []*sqlschema.ForeignKeyConstraint{
			{
				ReferencingColumnName: sqlschema.ColumnName("key_id"),
				ReferencedTableName:   sqlschema.TableName("ingestion_keys"),
				ReferencedColumnName:  sqlschema.ColumnName("id"),
			},
		},
	})
	sqls = append(sqls, tableSQLs...)

	indexSQLs := migration.sqlschema.Operator().CreateIndex(&sqlschema.UniqueIndex{
		TableName:   "ingestion_keys",
		ColumnNames: []sqlschema.ColumnName{"value"},
	})
	sqls = append(sqls, indexSQLs...)

	for _, sql := range sqls {
		if _, err := tx.ExecContext(ctx, string(sql)); err != nil {
			return err
		}
	}

	if _, err := tx.NewCreateIndex().
		Table("ingestion_keys").
		Column("org_id").
		Index("idx_ingestion_keys_org_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	if _, err := tx.NewCreateIndex().
		Table("ingestion_key_limits").
		Column("key_id").
		Index("idx_ingestion_key_limits_key_id").
		IfNotExists().
		Exec(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (migration *addIngestionKeys) Down(_ context.Context, _ *bun.DB) error {
	return nil
}
