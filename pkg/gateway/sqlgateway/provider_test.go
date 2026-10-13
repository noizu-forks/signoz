package sqlgateway_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/gateway"
	"github.com/SigNoz/signoz/pkg/gateway/sqlgateway"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlstoretest"
	"github.com/SigNoz/signoz/pkg/types/gatewaytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestProvider() (gateway.Gateway, sqlmock.Sqlmock) {
	sqlStore := sqlstoretest.New(sqlstore.Config{Provider: "sqlite"}, sqlmock.QueryMatcherRegexp)
	prov, err := sqlgateway.New(context.Background(), factory.ProviderSettings{}, gateway.Config{}, sqlStore)
	if err != nil {
		panic(err)
	}
	return prov, sqlStore.Mock()
}

func TestCreateAndGetIngestionKey(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectExec(`INSERT INTO "ingestion_keys"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	created, err := prov.CreateIngestionKey(ctx, orgID, "test-key", nil, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.Value)

	keyColumns := []string{"id", "org_id", "name", "value", "tags", "expires_at", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_keys".+ WHERE \(org_id = '` + orgID.StringValue() + `' AND id = '` + created.ID + `'\)`).
		WillReturnRows(sqlmock.NewRows(keyColumns).
			AddRow(created.ID, orgID.StringValue(), "test-key", created.Value, nil, int64(0), int64(1000), int64(1000)))

	limitColumns := []string{"id", "key_id", "signal", "config", "tags", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_key_limits"`).
		WillReturnRows(sqlmock.NewRows(limitColumns))

	key, err := prov.GetIngestionKey(ctx, orgID, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, key.ID)
	assert.Equal(t, "test-key", key.Name)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListIngestionKeys(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	keyColumns := []string{"id", "org_id", "name", "value", "tags", "expires_at", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_keys".+ WHERE \(org_id = '` + orgID.StringValue() + `'\)`).
		WillReturnRows(sqlmock.NewRows(keyColumns).
			AddRow("key-1", orgID.StringValue(), "key-a", "val-a", nil, int64(0), int64(1000), int64(1000)).
			AddRow("key-2", orgID.StringValue(), "key-b", "val-b", nil, int64(0), int64(2000), int64(2000)))

	result, err := prov.GetIngestionKeys(ctx, orgID, 1, 10)
	require.NoError(t, err)
	assert.Len(t, result.Keys, 2)
	assert.Equal(t, 2, result.Pagination.Total)
	assert.Equal(t, 1, result.Pagination.Pages)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateIngestionKey(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectExec(`UPDATE "ingestion_keys".+ WHERE \(id = 'key-1'\) AND \(org_id = '` + orgID.StringValue() + `'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := prov.UpdateIngestionKey(ctx, orgID, "key-1", "new-name", []string{"tag1"}, time.Time{})
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteIngestionKey(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectExec(`DELETE FROM "ingestion_keys".+ WHERE \(id = 'key-1'\) AND \(org_id = '` + orgID.StringValue() + `'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := prov.DeleteIngestionKey(ctx, orgID, "key-1")
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLimitCreateAndGet(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	keyColumns := []string{"id", "org_id", "name", "value", "tags", "expires_at", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_keys".+ WHERE \(org_id = '` + orgID.StringValue() + `' AND id = 'key-1'\)`).
		WillReturnRows(sqlmock.NewRows(keyColumns).
			AddRow("key-1", orgID.StringValue(), "test-key", "abc123", nil, int64(0), int64(1000), int64(1000)))

	mock.ExpectExec(`INSERT INTO "ingestion_key_limits"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sizeVal := int64(1024)
	cfg := gatewaytypes.LimitConfig{Day: &gatewaytypes.LimitValue{Size: &sizeVal}}

	created, err := prov.CreateIngestionKeyLimit(ctx, orgID, "key-1", "traces", cfg, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLimitUpdateAndDelete(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectExec(`UPDATE "ingestion_key_limits".+ WHERE \(.+\.id = 'limit-1'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	countVal := int64(100)
	cfg := gatewaytypes.LimitConfig{Second: &gatewaytypes.LimitValue{Count: &countVal}}
	err := prov.UpdateIngestionKeyLimit(ctx, orgID, "limit-1", cfg, nil)
	require.NoError(t, err)

	mock.ExpectExec(`DELETE FROM "ingestion_key_limits".+ WHERE \(.+\.id = 'limit-1'\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = prov.DeleteIngestionKeyLimit(ctx, orgID, "limit-1")
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetIngestionKey_NotFound(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_keys".+ WHERE \(org_id = '` + orgID.StringValue() + `' AND id = 'nonexistent'\)`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "org_id", "name", "value", "tags", "expires_at", "created_at", "updated_at"}))

	_, err := prov.GetIngestionKey(ctx, orgID, "nonexistent")
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchIngestionKeysByName(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	keyColumns := []string{"id", "org_id", "name", "value", "tags", "expires_at", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_keys".+ WHERE \(org_id = '` + orgID.StringValue() + `'.+\)`).
		WillReturnRows(sqlmock.NewRows(keyColumns).
			AddRow("key-1", orgID.StringValue(), "my-key", "val-a", nil, int64(0), int64(1000), int64(1000)))

	result, err := prov.SearchIngestionKeysByName(ctx, orgID, "my", 1, 10)
	require.NoError(t, err)
	assert.Len(t, result.Keys, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetIngestionKeyLimits(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	keyColumns := []string{"id", "org_id", "name", "value", "tags", "expires_at", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_keys".+ WHERE \(org_id = '` + orgID.StringValue() + `' AND id = 'key-1'\)`).
		WillReturnRows(sqlmock.NewRows(keyColumns).
			AddRow("key-1", orgID.StringValue(), "test-key", "abc123", nil, int64(0), int64(1000), int64(1000)))

	limitColumns := []string{"id", "key_id", "signal", "config", "tags", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT (.+) FROM "ingestion_key_limits"`).
		WillReturnRows(sqlmock.NewRows(limitColumns).
			AddRow("lim-1", "key-1", "metrics", []byte(`{}`), nil, int64(1000), int64(1000)))

	limits, err := prov.GetIngestionKeyLimits(ctx, orgID, "key-1")
	require.NoError(t, err)
	assert.Len(t, limits, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteIngestionKey_NotFound(t *testing.T) {
	prov, mock := newTestProvider()
	ctx := context.Background()
	orgID := valuer.GenerateUUID()

	mock.ExpectExec(`DELETE FROM "ingestion_keys".+ WHERE \(id = 'nonexistent'\) AND \(org_id = '` + orgID.StringValue() + `'\)`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := prov.DeleteIngestionKey(ctx, orgID, "nonexistent")
	require.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}