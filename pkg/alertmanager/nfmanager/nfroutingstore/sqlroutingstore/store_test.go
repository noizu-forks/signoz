package sqlroutingstore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SigNoz/signoz/pkg/factory/factorytest"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/sqlstore/sqlitesqlstore"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/alertmanagertypes"
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
		Model((*alertmanagertypes.RoutePolicy)(nil)).
		IfNotExists().
		Exec(t.Context())
	require.NoError(t, err)

	return NewStore(sqlstoreInstance).(*store)
}

func newRoutePolicy(orgID string, name string, kind alertmanagertypes.ExpressionKind, channels []string) *alertmanagertypes.RoutePolicy {
	return &alertmanagertypes.RoutePolicy{
		Identifiable:   types.Identifiable{ID: valuer.GenerateUUID()},
		TimeAuditable:  types.TimeAuditable{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Expression:     "alertstate == \"alerting\"",
		ExpressionKind: kind,
		Channels:       channels,
		Name:           name,
		Enabled:        true,
		OrgID:          orgID,
	}
}

func channelReferences(t *testing.T, store *store, orgID string, channelName string) []*alertmanagertypes.RoutePolicy {
	t.Helper()
	routes, err := store.GetAll(t.Context(), orgID)
	require.NoError(t, err)

	refs := make([]*alertmanagertypes.RoutePolicy, 0)
	for _, route := range routes {
		for _, channel := range route.Channels {
			if channel == channelName {
				refs = append(refs, route)
				break
			}
		}
	}

	return refs
}

func TestRoutePolicyDeleteReleasesChannelRefs(t *testing.T) {
	orgID := valuer.GenerateUUID().StringValue()
	otherOrgID := valuer.GenerateUUID().StringValue()

	testCases := []struct {
		name string
		kind alertmanagertypes.ExpressionKind
	}{
		{
			name: "PolicyBased_DeleteByID_ReleasesChannel",
			kind: alertmanagertypes.PolicyBasedExpression,
		},
		{
			name: "RuleBased_DeleteByName_ReleasesChannel",
			kind: alertmanagertypes.RuleBasedExpression,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newTestStore(t)

			policy := newRoutePolicy(orgID, "critical-route", testCase.kind, []string{"pager", "email"})
			require.NoError(t, store.Create(t.Context(), policy))

			foreignPolicy := newRoutePolicy(otherOrgID, "foreign-route", testCase.kind, []string{"pager"})
			require.NoError(t, store.Create(t.Context(), foreignPolicy))

			require.Len(t, channelReferences(t, store, orgID, "pager"), 1)

			if testCase.kind == alertmanagertypes.PolicyBasedExpression {
				require.NoError(t, store.Delete(t.Context(), orgID, policy.ID.StringValue()))
			} else {
				require.NoError(t, store.DeleteRouteByName(t.Context(), orgID, policy.Name))
			}

			assert.Empty(t, channelReferences(t, store, orgID, "pager"))
			assert.Empty(t, channelReferences(t, store, orgID, "email"))

			foreign := channelReferences(t, store, otherOrgID, "pager")
			require.Len(t, foreign, 1)
			assert.Equal(t, "foreign-route", foreign[0].Name)

			_, err := store.GetByID(t.Context(), orgID, policy.ID.StringValue())
			assert.Error(t, err)
		})
	}
}
