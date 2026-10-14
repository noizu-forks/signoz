package sqlauthzstore

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
		Model((*authtypes.Role)(nil)).
		IfNotExists().
		Exec(t.Context())
	require.NoError(t, err)

	_, err = sqlstoreInstance.BunDB().NewCreateIndex().
		IfNotExists().
		Unique().
		Index("role_name_org_id_key").
		Table("role").
		Column("name", "org_id").
		Exec(t.Context())
	require.NoError(t, err)

	return NewSqlAuthzStore(sqlstoreInstance).(*store)
}

func newOrgID(t *testing.T) valuer.UUID {
	t.Helper()
	orgID, err := valuer.NewUUID(valuer.GenerateUUID().StringValue())
	require.NoError(t, err)
	return orgID
}

func TestRoleCrud(t *testing.T) {
	orgID := newOrgID(t)
	otherOrgID := newOrgID(t)
	transactionGroups, err := authtypes.NewTransactionGroups([]byte(`[{"relation":"read","objectGroup":{"resource":{"type":"organization","kind":"organization"},"selectors":["*"]}}]`))
	require.NoError(t, err)

	testCases := []struct {
		name string
		run  func(t *testing.T, store *store)
	}{
		{
			name: "Create_Read_Back",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "read everything", authtypes.RoleTypeCustom, orgID, transactionGroups)
				require.NoError(t, store.Create(t.Context(), role))

				stored, err := store.Get(t.Context(), orgID, role.ID)
				require.NoError(t, err)
				assert.Equal(t, "reader", stored.Name)
				assert.Equal(t, authtypes.RoleTypeCustom, stored.Type)
				assert.Equal(t, orgID, stored.OrgID)
				assert.Len(t, stored.TransactionGroups, 1)

				byName, err := store.GetByOrgIDAndName(t.Context(), orgID, "reader")
				require.NoError(t, err)
				assert.Equal(t, role.ID, byName.ID)
			},
		},
		{
			name: "Create_DuplicateName_SameOrg_AlreadyExists",
			run: func(t *testing.T, store *store) {
				require.NoError(t, store.Create(t.Context(), authtypes.NewRole("reader", "first", authtypes.RoleTypeCustom, orgID, nil)))

				err := store.Create(t.Context(), authtypes.NewRole("reader", "second", authtypes.RoleTypeCustom, orgID, nil))
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeAlreadyExists))
			},
		},
		{
			name: "Create_SameName_OtherOrg_Allowed",
			run: func(t *testing.T, store *store) {
				require.NoError(t, store.Create(t.Context(), authtypes.NewRole("reader", "org one", authtypes.RoleTypeCustom, orgID, nil)))
				require.NoError(t, store.Create(t.Context(), authtypes.NewRole("reader", "org two", authtypes.RoleTypeCustom, otherOrgID, nil)))

				roles, err := store.List(t.Context(), orgID)
				require.NoError(t, err)
				require.Len(t, roles, 1)
				assert.Equal(t, "org one", roles[0].Description)
			},
		},
		{
			name: "Get_OtherOrg_NotFound",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "org one", authtypes.RoleTypeCustom, orgID, nil)
				require.NoError(t, store.Create(t.Context(), role))

				_, err := store.Get(t.Context(), otherOrgID, role.ID)
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))

				_, err = store.GetByOrgIDAndName(t.Context(), otherOrgID, "reader")
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			},
		},
		{
			name: "Get_Missing_NotFound",
			run: func(t *testing.T, store *store) {
				_, err := store.Get(t.Context(), orgID, valuer.GenerateUUID())
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			},
		},
		{
			name: "Update_Existing_Persists",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "before", authtypes.RoleTypeCustom, orgID, nil)
				require.NoError(t, store.Create(t.Context(), role))

				require.NoError(t, role.Update("after", transactionGroups))
				require.NoError(t, store.Update(t.Context(), orgID, role))

				stored, err := store.Get(t.Context(), orgID, role.ID)
				require.NoError(t, err)
				assert.Equal(t, "after", stored.Description)
				assert.Len(t, stored.TransactionGroups, 1)
			},
		},
		{
			name: "Update_Missing_NotFound",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "missing", authtypes.RoleTypeCustom, orgID, nil)
				role.ID = valuer.GenerateUUID()

				err := store.Update(t.Context(), orgID, role)
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			},
		},
		{
			name: "Update_OtherOrg_NotFound_RoleUntouched",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "org one", authtypes.RoleTypeCustom, orgID, nil)
				require.NoError(t, store.Create(t.Context(), role))

				require.NoError(t, role.Update("org two", nil))
				err := store.Update(t.Context(), otherOrgID, role)
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))

				stored, err := store.Get(t.Context(), orgID, role.ID)
				require.NoError(t, err)
				assert.Equal(t, "org one", stored.Description)
			},
		},
		{
			name: "Delete_Existing_GetReturnsNotFound",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "org one", authtypes.RoleTypeCustom, orgID, nil)
				require.NoError(t, store.Create(t.Context(), role))

				require.NoError(t, store.Delete(t.Context(), orgID, role.ID))

				_, err := store.Get(t.Context(), orgID, role.ID)
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			},
		},
		{
			name: "Delete_Missing_NotFound",
			run: func(t *testing.T, store *store) {
				err := store.Delete(t.Context(), orgID, valuer.GenerateUUID())
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))
			},
		},
		{
			name: "Delete_OtherOrg_NotFound_RoleRemains",
			run: func(t *testing.T, store *store) {
				role := authtypes.NewRole("reader", "org one", authtypes.RoleTypeCustom, orgID, nil)
				require.NoError(t, store.Create(t.Context(), role))

				err := store.Delete(t.Context(), otherOrgID, role.ID)
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeNotFound))

				_, err = store.Get(t.Context(), orgID, role.ID)
				require.NoError(t, err)
			},
		},
		{
			name: "ListByOrgIDAndNames_OrgScoped",
			run: func(t *testing.T, store *store) {
				require.NoError(t, store.Create(t.Context(), authtypes.NewRole("reader", "org one", authtypes.RoleTypeCustom, orgID, nil)))
				require.NoError(t, store.Create(t.Context(), authtypes.NewRole("reader", "org two", authtypes.RoleTypeCustom, otherOrgID, nil)))
				require.NoError(t, store.Create(t.Context(), authtypes.NewRole("writer", "org one", authtypes.RoleTypeCustom, orgID, nil)))

				roles, err := store.ListByOrgIDAndNames(t.Context(), orgID, []string{"reader", "writer"})
				require.NoError(t, err)
				require.Len(t, roles, 2)
				for _, role := range roles {
					assert.Equal(t, orgID, role.OrgID)
				}

				_, err = store.ListByOrgIDAndNames(t.Context(), orgID, []string{"missing"})
				require.Error(t, err)
				assert.True(t, errors.Ast(err, errors.TypeInvalidInput))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.run(t, newTestStore(t))
		})
	}
}
