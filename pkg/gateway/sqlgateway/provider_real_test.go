package sqlgateway_test

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/gateway"
	"github.com/SigNoz/signoz/pkg/gateway/sqlgateway"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/sqlmigration"
	"github.com/SigNoz/signoz/pkg/sqlschema"
	"github.com/SigNoz/signoz/pkg/sqlschema/sqlitesqlschema"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstore"
	"github.com/SigNoz/signoz/pkg/types/gatewaytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type realStore struct {
	gateway.Gateway
	sqlStore sqlstore.SQLStore
}

func newRealStore(t *testing.T) realStore {
	t.Helper()
	ctx := context.Background()
	providerSettings := instrumentationtest.New().ToProviderSettings()

	sqlStore, err := sqlitesqlstore.New(ctx, providerSettings, sqlstore.Config{
		Provider: "sqlite",
		Connection: sqlstore.ConnectionConfig{
			MaxOpenConns: 1,
		},
		Sqlite: sqlstore.SqliteConfig{
			Path:            ":memory:",
			Mode:            "memory",
			BusyTimeout:     time.Second,
			TransactionMode: "deferred",
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlStore.SQLDB().Close() })

	sqlSchema, err := sqlitesqlschema.New(ctx, providerSettings, sqlschema.Config{}, sqlStore)
	require.NoError(t, err)

	migration, err := sqlmigration.NewAddIngestionKeysFactory(sqlStore, sqlSchema).New(ctx, providerSettings, sqlmigration.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.Up(ctx, sqlStore.BunDB()))

	prov, err := sqlgateway.New(ctx, factory.ProviderSettings{}, gateway.Config{}, sqlStore)
	require.NoError(t, err)

	return realStore{Gateway: prov, sqlStore: sqlStore}
}

func TestRealStore_CRUD_RoundTrip(t *testing.T) {
	prov := newRealStore(t)
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	created, err := prov.CreateIngestionKey(ctx, orgID, "round-trip", []string{"team:a"}, time.Time{})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.NotEmpty(t, created.Value)

	key, err := prov.GetIngestionKey(ctx, orgID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "round-trip", key.Name)
	assert.Equal(t, []string{"team:a"}, key.Tags)
	assert.Empty(t, key.Limits)

	sizeVal := int64(2048)
	limit, err := prov.CreateIngestionKeyLimit(ctx, orgID, created.ID, "traces", gatewaytypes.LimitConfig{Day: &gatewaytypes.LimitValue{Size: &sizeVal}}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, limit.ID)

	key, err = prov.GetIngestionKey(ctx, orgID, created.ID)
	require.NoError(t, err)
	require.Len(t, key.Limits, 1)
	assert.Equal(t, limit.ID, key.Limits[0].ID)
	assert.Equal(t, "traces", key.Limits[0].Signal)
	require.NotNil(t, key.Limits[0].Config.Day)
	require.NotNil(t, key.Limits[0].Config.Day.Size)
	assert.Equal(t, int64(2048), *key.Limits[0].Config.Day.Size)
	require.NotNil(t, key.Limits[0].Metric.Day)
	assert.Equal(t, int64(2048), key.Limits[0].Metric.Day.Size)
	assert.Nil(t, key.Limits[0].Metric.Second)

	expiresAt := time.Now().Add(24 * time.Hour)
	require.NoError(t, prov.UpdateIngestionKey(ctx, orgID, created.ID, "round-trip-2", []string{"team:b"}, expiresAt))

	key, err = prov.GetIngestionKey(ctx, orgID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "round-trip-2", key.Name)
	assert.Equal(t, []string{"team:b"}, key.Tags)
	assert.Equal(t, expiresAt.UnixMilli(), key.ExpiresAt.UnixMilli())

	countVal := int64(50)
	require.NoError(t, prov.UpdateIngestionKeyLimit(ctx, orgID, limit.ID, gatewaytypes.LimitConfig{Second: &gatewaytypes.LimitValue{Count: &countVal}}, []string{"env:prod"}))

	key, err = prov.GetIngestionKey(ctx, orgID, created.ID)
	require.NoError(t, err)
	require.Len(t, key.Limits, 1)
	assert.Nil(t, key.Limits[0].Config.Day)
	require.NotNil(t, key.Limits[0].Config.Second)
	assert.Equal(t, int64(50), *key.Limits[0].Config.Second.Count)
	assert.Equal(t, []string{"env:prod"}, key.Limits[0].Tags)
	assert.Nil(t, key.Limits[0].Metric.Day)
	require.NotNil(t, key.Limits[0].Metric.Second)
	assert.Equal(t, int64(50), key.Limits[0].Metric.Second.Count)

	fetched, err := prov.GetIngestionKeyLimit(ctx, orgID, limit.ID)
	require.NoError(t, err)
	assert.Equal(t, limit.ID, fetched.ID)
	assert.Equal(t, "traces", fetched.Signal)

	require.NoError(t, prov.DeleteIngestionKeyLimit(ctx, orgID, limit.ID))

	key, err = prov.GetIngestionKey(ctx, orgID, created.ID)
	require.NoError(t, err)
	assert.Empty(t, key.Limits)

	require.NoError(t, prov.DeleteIngestionKey(ctx, orgID, created.ID))

	_, err = prov.GetIngestionKey(ctx, orgID, created.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)
}

func TestRealStore_ListReturnsPopulatedLimits(t *testing.T) {
	prov := newRealStore(t)
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	keyA, err := prov.CreateIngestionKey(ctx, orgID, "list-alpha", nil, time.Time{})
	require.NoError(t, err)
	keyB, err := prov.CreateIngestionKey(ctx, orgID, "list-beta", nil, time.Time{})
	require.NoError(t, err)

	sizeVal := int64(1024)
	_, err = prov.CreateIngestionKeyLimit(ctx, orgID, keyA.ID, "metrics", gatewaytypes.LimitConfig{Day: &gatewaytypes.LimitValue{Size: &sizeVal}}, nil)
	require.NoError(t, err)
	countVal := int64(100)
	_, err = prov.CreateIngestionKeyLimit(ctx, orgID, keyB.ID, "logs", gatewaytypes.LimitConfig{Second: &gatewaytypes.LimitValue{Count: &countVal}}, nil)
	require.NoError(t, err)

	result, err := prov.GetIngestionKeys(ctx, orgID, 1, 10)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)

	limitsByKeyID := make(map[string]gatewaytypes.Limit, len(result.Keys))
	for _, key := range result.Keys {
		require.Len(t, key.Limits, 1, "key %s should have its limits populated", key.ID)
		limitsByKeyID[key.ID] = key.Limits[0]
	}

	limitA, ok := limitsByKeyID[keyA.ID]
	require.True(t, ok)
	assert.Equal(t, "metrics", limitA.Signal)
	require.NotNil(t, limitA.Metric.Day)
	assert.Equal(t, int64(1024), limitA.Metric.Day.Size)

	limitB, ok := limitsByKeyID[keyB.ID]
	require.True(t, ok)
	assert.Equal(t, "logs", limitB.Signal)
	require.NotNil(t, limitB.Metric.Second)
	assert.Equal(t, int64(100), limitB.Metric.Second.Count)
}

func TestRealStore_SearchReturnsPopulatedLimits(t *testing.T) {
	prov := newRealStore(t)
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	keyA, err := prov.CreateIngestionKey(ctx, orgID, "search-alpha", nil, time.Time{})
	require.NoError(t, err)
	keyB, err := prov.CreateIngestionKey(ctx, orgID, "search-beta", nil, time.Time{})
	require.NoError(t, err)

	_, err = prov.CreateIngestionKey(ctx, orgID, "unrelated", nil, time.Time{})
	require.NoError(t, err)

	sizeVal := int64(2048)
	_, err = prov.CreateIngestionKeyLimit(ctx, orgID, keyA.ID, "traces", gatewaytypes.LimitConfig{Day: &gatewaytypes.LimitValue{Size: &sizeVal}}, nil)
	require.NoError(t, err)
	countVal := int64(20)
	_, err = prov.CreateIngestionKeyLimit(ctx, orgID, keyB.ID, "logs", gatewaytypes.LimitConfig{Second: &gatewaytypes.LimitValue{Count: &countVal}}, nil)
	require.NoError(t, err)

	result, err := prov.SearchIngestionKeysByName(ctx, orgID, "search", 1, 10)
	require.NoError(t, err)
	require.Len(t, result.Keys, 2)

	limitsByKeyID := make(map[string]gatewaytypes.Limit, len(result.Keys))
	for _, key := range result.Keys {
		require.Len(t, key.Limits, 1, "key %s should have its limits populated", key.ID)
		limitsByKeyID[key.ID] = key.Limits[0]
	}

	limitA, ok := limitsByKeyID[keyA.ID]
	require.True(t, ok)
	assert.Equal(t, "traces", limitA.Signal)
	require.NotNil(t, limitA.Metric.Day)
	assert.Equal(t, int64(2048), limitA.Metric.Day.Size)

	limitB, ok := limitsByKeyID[keyB.ID]
	require.True(t, ok)
	assert.Equal(t, "logs", limitB.Signal)
	require.NotNil(t, limitB.Metric.Second)
	assert.Equal(t, int64(20), limitB.Metric.Second.Count)
}

func TestRealStore_DeleteCascadesLimits(t *testing.T) {
	store := newRealStore(t)
	prov := store.Gateway
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	created, err := prov.CreateIngestionKey(ctx, orgID, "cascade", nil, time.Time{})
	require.NoError(t, err)

	sizeVal := int64(4096)
	limitA, err := prov.CreateIngestionKeyLimit(ctx, orgID, created.ID, "traces", gatewaytypes.LimitConfig{Day: &gatewaytypes.LimitValue{Size: &sizeVal}}, nil)
	require.NoError(t, err)
	countVal := int64(10)
	limitB, err := prov.CreateIngestionKeyLimit(ctx, orgID, created.ID, "logs", gatewaytypes.LimitConfig{Second: &gatewaytypes.LimitValue{Count: &countVal}}, nil)
	require.NoError(t, err)

	require.NoError(t, prov.DeleteIngestionKey(ctx, orgID, created.ID))

	remaining, err := store.sqlStore.BunDB().NewSelect().Table("ingestion_key_limits").Where("key_id = ?", created.ID).Count(ctx)
	require.NoError(t, err)
	assert.Zero(t, remaining, "limits must be deleted with the key")

	_, err = prov.GetIngestionKeyLimit(ctx, orgID, limitA.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	_, err = prov.GetIngestionKeyLimit(ctx, orgID, limitB.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)
}

func TestRealStore_PageClamp(t *testing.T) {
	prov := newRealStore(t)
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	for _, name := range []string{"page-one", "page-two", "page-three"} {
		_, err := prov.CreateIngestionKey(ctx, orgID, name, nil, time.Time{})
		require.NoError(t, err)
	}

	result, err := prov.GetIngestionKeys(ctx, orgID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Pagination.Page)
	assert.Equal(t, 3, result.Pagination.Total)
	assert.Len(t, result.Keys, 3)

	result, err = prov.GetIngestionKeys(ctx, orgID, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, gateway.DefaultPageSize, result.Pagination.PerPage)
	assert.Equal(t, 1, result.Pagination.Pages)
	assert.Len(t, result.Keys, 3)

	result, err = prov.GetIngestionKeys(ctx, orgID, 1, 999999)
	require.NoError(t, err)
	assert.Equal(t, 100, result.Pagination.PerPage)
	assert.Equal(t, 1, result.Pagination.Pages)
	assert.Len(t, result.Keys, 3)

	otherOrg := valuer.GenerateUUID()
	err = prov.UpdateIngestionKey(ctx, otherOrg, "page-one", "renamed", nil, time.Time{})
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	err = prov.DeleteIngestionKey(ctx, otherOrg, "page-one")
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)
}

func TestRealStore_CrossOrgIsolation(t *testing.T) {
	prov := newRealStore(t)
	ctx := context.Background()
	orgA := valuer.GenerateUUID()
	orgB := valuer.GenerateUUID()

	created, err := prov.CreateIngestionKey(ctx, orgA, "isolated", nil, time.Time{})
	require.NoError(t, err)

	sizeVal := int64(512)
	limit, err := prov.CreateIngestionKeyLimit(ctx, orgA, created.ID, "traces", gatewaytypes.LimitConfig{Day: &gatewaytypes.LimitValue{Size: &sizeVal}}, nil)
	require.NoError(t, err)

	_, err = prov.GetIngestionKey(ctx, orgB, created.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	result, err := prov.GetIngestionKeys(ctx, orgB, 1, 10)
	require.NoError(t, err)
	assert.Empty(t, result.Keys)

	result, err = prov.SearchIngestionKeysByName(ctx, orgB, "isolated", 1, 10)
	require.NoError(t, err)
	assert.Empty(t, result.Keys)

	err = prov.UpdateIngestionKey(ctx, orgB, created.ID, "renamed", nil, time.Time{})
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	err = prov.DeleteIngestionKey(ctx, orgB, created.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	_, err = prov.CreateIngestionKeyLimit(ctx, orgB, created.ID, "logs", gatewaytypes.LimitConfig{}, nil)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	_, err = prov.GetIngestionKeyLimits(ctx, orgB, created.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	_, err = prov.GetIngestionKeyLimit(ctx, orgB, limit.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	err = prov.UpdateIngestionKeyLimit(ctx, orgB, limit.ID, gatewaytypes.LimitConfig{}, nil)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	err = prov.DeleteIngestionKeyLimit(ctx, orgB, limit.ID)
	require.Error(t, err)
	assert.True(t, errors.Ast(err, errors.TypeNotFound), "expected a not-found error, got %v", err)

	key, err := prov.GetIngestionKey(ctx, orgA, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "isolated", key.Name)
	require.Len(t, key.Limits, 1)
}
