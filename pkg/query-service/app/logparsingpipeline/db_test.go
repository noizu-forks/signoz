package logparsingpipeline

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory/factorytest"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstore"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/types/pipelinetypes"
	"github.com/SigNoz/signoz/pkg/valuer"

	signozv3 "github.com/SigNoz/signoz/pkg/query-service/model/v3"
)

func newTestRepo(t *testing.T) *Repo {
	t.Helper()
	sqlstoreInstance, err := sqlitesqlstore.New(t.Context(), factorytest.NewSettings(), sqlstore.Config{
		Provider: "sqlite",
		Connection: sqlstore.ConnectionConfig{
			MaxOpenConns:    1,
			MaxConnLifetime: 0,
		},
		Sqlite: sqlstore.SqliteConfig{
			Path:            filepath.Join(t.TempDir(), "test.db"),
			Mode:            "wal",
			BusyTimeout:     5 * time.Second,
			TransactionMode: "deferred",
		},
	})
	require.NoError(t, err)

	_, err = sqlstoreInstance.BunDB().NewCreateTable().
		Model((*pipelinetypes.StoreablePipeline)(nil)).
		IfNotExists().
		Exec(t.Context())
	require.NoError(t, err)

	repo := NewRepo(sqlstoreInstance)
	return &repo
}

func newClaimsContext(t *testing.T, email string) context.Context {
	t.Helper()
	return authtypes.NewContextWithClaims(t.Context(), authtypes.Claims{Email: email})
}

func newPostable(name string, alias string) pipelinetypes.PostablePipeline {
	return pipelinetypes.PostablePipeline{
		OrderID: 1,
		Name:    name,
		Alias:   alias,
		Enabled: true,
		Filter: &signozv3.FilterSet{
			Operator: "AND",
			Items: []signozv3.FilterItem{
				{Key: signozv3.AttributeKey{Key: "service_name", DataType: signozv3.AttributeKeyDataTypeString}, Value: "frontend", Operator: "="},
			},
		},
		Config: []pipelinetypes.PipelineOperator{{
			OrderId: 1,
			ID:      "add_operator",
			Type:    "add",
			Field:   "resource.test1",
			Value:   "val1",
			Enabled: true,
			Name:    "add test field",
		}},
	}
}

func TestApplyPipelinesPreservesIds(t *testing.T) {
	repo := newTestRepo(t)
	orgID := valuer.GenerateUUID()
	ctx := newClaimsContext(t, "owner@signoz.io")

	created, err := repo.insertPipeline(ctx, orgID, ptr(newPostable("drop debug", "dd")))
	require.NoError(t, err)

	postable := newPostable("drop debug renamed", "dd")
	postable.ID = created.ID.StringValue()
	postable.OrderID = 2

	existing, err := repo.GetPipeline(ctx, orgID.String(), created.ID.StringValue())
	require.NoError(t, err)

	updated, err := repo.updatePipeline(ctx, orgID, &postable, &existing.StoreablePipeline)
	require.NoError(t, err)

	assert.Equal(t, created.ID, updated.ID)
	assert.Equal(t, "drop debug renamed", updated.Name)
	assert.Equal(t, 2, updated.OrderID)
	assert.Equal(t, "owner@signoz.io", updated.CreatedBy)
	assert.False(t, updated.CreatedAt.IsZero())

	stored, err := repo.GetPipeline(ctx, orgID.String(), created.ID.StringValue())
	require.NoError(t, err)
	assert.Equal(t, created.ID, stored.ID)
	assert.Equal(t, "drop debug renamed", stored.Name)
	assert.Equal(t, "owner@signoz.io", stored.CreatedBy)
}

func TestUpdatePipelineOtherOrgNotFound(t *testing.T) {
	repo := newTestRepo(t)
	orgID := valuer.GenerateUUID()
	otherOrgID := valuer.GenerateUUID()
	ctx := newClaimsContext(t, "owner@signoz.io")

	created, err := repo.insertPipeline(ctx, orgID, ptr(newPostable("drop debug", "dd")))
	require.NoError(t, err)

	existing, err := repo.GetPipeline(ctx, orgID.String(), created.ID.StringValue())
	require.NoError(t, err)

	postable := newPostable("drop debug renamed", "dd")
	postable.ID = created.ID.StringValue()

	_, err = repo.updatePipeline(ctx, otherOrgID, &postable, &existing.StoreablePipeline)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound))

	stored, err := repo.GetPipeline(ctx, orgID.String(), created.ID.StringValue())
	require.NoError(t, err)
	assert.Equal(t, "drop debug", stored.Name)
}

func ptr(p pipelinetypes.PostablePipeline) *pipelinetypes.PostablePipeline {
	return &p
}
