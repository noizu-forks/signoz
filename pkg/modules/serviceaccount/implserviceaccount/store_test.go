package implserviceaccount

import (
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
	"github.com/SigNoz/signoz/pkg/types/serviceaccounttypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

func newTestStore(t *testing.T) *store {
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

	for _, model := range []any{
		(*serviceaccounttypes.ServiceAccount)(nil),
		(*serviceaccounttypes.ServiceAccountRole)(nil),
		(*authtypes.Role)(nil),
	} {
		_, err = sqlstoreInstance.BunDB().NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(t.Context())
		require.NoError(t, err)
	}

	return NewStore(sqlstoreInstance).(*store)
}

func TestGetExcludesDeleted(t *testing.T) {
	orgID := valuer.GenerateUUID()

	testCases := []struct {
		name   string
		status serviceaccounttypes.ServiceAccountStatus
	}{
		{
			name:   "DeletedServiceAccount_NotFound",
			status: serviceaccounttypes.ServiceAccountStatusDeleted,
		},
		{
			name:   "ActiveServiceAccount_Visible",
			status: serviceaccounttypes.ServiceAccountStatusActive,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newTestStore(t)

			serviceAccount := serviceaccounttypes.NewServiceAccount("automation", "signoz.io", testCase.status, orgID)
			require.NoError(t, store.Create(t.Context(), serviceAccount))

			_, err := store.Get(t.Context(), orgID, serviceAccount.ID)
			if testCase.status == serviceaccounttypes.ServiceAccountStatusDeleted {
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			} else {
				require.NoError(t, err)
			}

			_, err = store.GetWithRoles(t.Context(), orgID, serviceAccount.ID)
			if testCase.status == serviceaccounttypes.ServiceAccountStatusDeleted {
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			} else {
				require.NoError(t, err)
			}
		})
	}
}
