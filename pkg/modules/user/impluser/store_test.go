package impluser

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
	"github.com/SigNoz/signoz/pkg/types"
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

	_, err = sqlstoreInstance.BunDB().NewCreateTable().
		Model((*types.User)(nil)).
		IfNotExists().
		Exec(t.Context())
	require.NoError(t, err)

	return NewStore(sqlstoreInstance, factorytest.NewSettings()).(*store)
}

func TestGetByOrgIDAndIDExcludesDeleted(t *testing.T) {
	testCases := []struct {
		name   string
		status valuer.String
	}{
		{
			name:   "DeletedUser_NotFound",
			status: types.UserStatusDeleted,
		},
		{
			name:   "ActiveUser_Visible",
			status: types.UserStatusActive,
		},
		{
			name:   "PendingInviteUser_Visible",
			status: types.UserStatusPendingInvite,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newTestStore(t)

			orgID := valuer.GenerateUUID()
			email, err := valuer.NewEmail("invitee" + valuer.GenerateUUID().StringValue()[:8] + "@signoz.io")
			require.NoError(t, err)

			user, err := types.NewUser("invitee", email, orgID, testCase.status)
			require.NoError(t, err)
			require.NoError(t, store.CreateUser(t.Context(), user))

			fetched, err := store.GetByOrgIDAndID(t.Context(), orgID, user.ID)
			if testCase.status == types.UserStatusDeleted {
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, user.ID, fetched.ID)
		})
	}
}
