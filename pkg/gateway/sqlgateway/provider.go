package sqlgateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/gateway"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/SigNoz/signoz/pkg/types/gatewaytypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type provider struct {
	store *store
}

func NewProviderFactory(sqlstore sqlstore.SQLStore) factory.ProviderFactory[gateway.Gateway, gateway.Config] {
	return factory.NewProviderFactory(factory.MustNewName("sql"), func(ctx context.Context, ps factory.ProviderSettings, c gateway.Config) (gateway.Gateway, error) {
		return New(ctx, ps, c, sqlstore)
	})
}

func New(_ context.Context, _ factory.ProviderSettings, _ gateway.Config, sqlstore sqlstore.SQLStore) (gateway.Gateway, error) {
	return &provider{store: newStore(sqlstore)}, nil
}

func (p *provider) GetIngestionKeys(ctx context.Context, orgID valuer.UUID, page, perPage int) (*gatewaytypes.GettableIngestionKeys, error) {
	offset := (page - 1) * perPage
	keys, total, err := p.store.listKeys(ctx, orgID.StringValue(), offset, perPage)
	if err != nil {
		return nil, err
	}

	pages := total / perPage
	if total%perPage != 0 {
		pages++
	}

	result := make([]gatewaytypes.IngestionKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, p.toGatewayIngestionKey(&k))
	}

	return &gatewaytypes.GettableIngestionKeys{
		Keys: result,
		Pagination: gatewaytypes.Pagination{
			Page:    page,
			PerPage: perPage,
			Pages:   pages,
			Total:   total,
		},
	}, nil
}

func (p *provider) SearchIngestionKeysByName(ctx context.Context, orgID valuer.UUID, name string, page, perPage int) (*gatewaytypes.GettableIngestionKeys, error) {
	offset := (page - 1) * perPage
	keys, total, err := p.store.searchKeysByName(ctx, orgID.StringValue(), name, offset, perPage)
	if err != nil {
		return nil, err
	}

	pages := total / perPage
	if total%perPage != 0 {
		pages++
	}

	result := make([]gatewaytypes.IngestionKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, p.toGatewayIngestionKey(&k))
	}

	return &gatewaytypes.GettableIngestionKeys{
		Keys: result,
		Pagination: gatewaytypes.Pagination{
			Page:    page,
			PerPage: perPage,
			Pages:   pages,
			Total:   total,
		},
	}, nil
}

func (p *provider) GetIngestionKey(ctx context.Context, orgID valuer.UUID, keyID string) (*gatewaytypes.IngestionKey, error) {
	key, err := p.store.getKey(ctx, orgID.StringValue(), keyID)
	if err != nil {
		return nil, err
	}

	limits, err := p.store.listLimits(ctx, orgID.StringValue(), keyID)
	if err != nil {
		return nil, err
	}

	result := p.toGatewayIngestionKey(key)
	result.Limits = p.toGatewayLimits(limits)

	return &result, nil
}

func (p *provider) CreateIngestionKey(ctx context.Context, orgID valuer.UUID, name string, tags []string, expiresAt time.Time) (*gatewaytypes.GettableCreatedIngestionKey, error) {
	id := valuer.GenerateUUID().StringValue()
	keyValue, err := generateKeyValue()
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "failed to generate ingestion key value")
	}

	now := nowMillis()
	key := &ingestionKey{
		ID:        id,
		OrgID:     orgID.StringValue(),
		Name:      name,
		Value:     keyValue,
		Tags:      encodeTags(tags),
		ExpiresAt: expiresAt.UnixMilli(),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := p.store.createKey(ctx, key); err != nil {
		return nil, err
	}

	return &gatewaytypes.GettableCreatedIngestionKey{
		ID:    id,
		Value: keyValue,
	}, nil
}

func (p *provider) UpdateIngestionKey(ctx context.Context, orgID valuer.UUID, keyID string, name string, tags []string, expiresAt time.Time) error {
	return p.store.updateKey(ctx, orgID.StringValue(), keyID, name, encodeTags(tags), expiresAt.UnixMilli(), nowMillis())
}

func (p *provider) DeleteIngestionKey(ctx context.Context, orgID valuer.UUID, keyID string) error {
	return p.store.deleteKey(ctx, orgID.StringValue(), keyID)
}

func (p *provider) CreateIngestionKeyLimit(ctx context.Context, orgID valuer.UUID, keyID string, signal string, limitConfig gatewaytypes.LimitConfig, tags []string) (*gatewaytypes.GettableCreatedIngestionKeyLimit, error) {
	_, err := p.store.getKey(ctx, orgID.StringValue(), keyID)
	if err != nil {
		return nil, err
	}

	id := valuer.GenerateUUID().StringValue()
	now := nowMillis()

	config := limitConfigData{
		Day:    toStoreLimitValue(limitConfig.Day),
		Second: toStoreLimitValue(limitConfig.Second),
	}

	limit := &ingestionKeyLimit{
		ID:        id,
		KeyID:     keyID,
		Signal:    signal,
		Config:    encodeLimitConfig(config),
		Tags:      encodeTags(tags),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := p.store.createLimit(ctx, limit); err != nil {
		return nil, err
	}

	return &gatewaytypes.GettableCreatedIngestionKeyLimit{
		ID: id,
	}, nil
}

func (p *provider) GetIngestionKeyLimit(ctx context.Context, orgID valuer.UUID, limitID string) (*gatewaytypes.Limit, error) {
	limit, err := p.store.getLimit(ctx, orgID.StringValue(), limitID)
	if err != nil {
		return nil, err
	}

	result := p.toGatewayLimit(limit)
	return &result, nil
}

func (p *provider) GetIngestionKeyLimits(ctx context.Context, orgID valuer.UUID, keyID string) ([]gatewaytypes.Limit, error) {
	_, err := p.store.getKey(ctx, orgID.StringValue(), keyID)
	if err != nil {
		return nil, err
	}

	limits, err := p.store.listLimits(ctx, orgID.StringValue(), keyID)
	if err != nil {
		return nil, err
	}

	return p.toGatewayLimits(limits), nil
}

func (p *provider) UpdateIngestionKeyLimit(ctx context.Context, orgID valuer.UUID, limitID string, limitConfig gatewaytypes.LimitConfig, tags []string) error {
	config := limitConfigData{
		Day:    toStoreLimitValue(limitConfig.Day),
		Second: toStoreLimitValue(limitConfig.Second),
	}

	return p.store.updateLimit(ctx, orgID.StringValue(), limitID, encodeLimitConfig(config), encodeTags(tags), nowMillis())
}

func (p *provider) DeleteIngestionKeyLimit(ctx context.Context, orgID valuer.UUID, limitID string) error {
	return p.store.deleteLimit(ctx, orgID.StringValue(), limitID)
}

func (p *provider) toGatewayIngestionKey(key *ingestionKey) gatewaytypes.IngestionKey {
	return gatewaytypes.IngestionKey{
		ID:          key.ID,
		Name:        key.Name,
		Value:       key.Value,
		ExpiresAt:   fromMillis(key.ExpiresAt),
		Tags:        decodeTags(key.Tags),
		CreatedAt:   fromMillis(key.CreatedAt),
		UpdatedAt:   fromMillis(key.UpdatedAt),
		WorkspaceID: key.OrgID,
	}
}

func (p *provider) toGatewayLimit(limit *ingestionKeyLimit) gatewaytypes.Limit {
	return gatewaytypes.Limit{
		ID:        limit.ID,
		Signal:    limit.Signal,
		Config:    toGatewayLimitConfig(decodeLimitConfig(limit.Config)),
		Tags:      decodeTags(limit.Tags),
		CreatedAt: fromMillis(limit.CreatedAt),
		UpdatedAt: fromMillis(limit.UpdatedAt),
		KeyID:     limit.KeyID,
	}
}

func (p *provider) toGatewayLimits(limits []ingestionKeyLimit) []gatewaytypes.Limit {
	result := make([]gatewaytypes.Limit, 0, len(limits))
	for _, l := range limits {
		result = append(result, p.toGatewayLimit(&l))
	}
	return result
}

func toStoreLimitValue(v *gatewaytypes.LimitValue) *limitValueData {
	if v == nil {
		return nil
	}
	return &limitValueData{Size: v.Size, Count: v.Count}
}

func toGatewayLimitValue(v *limitValueData) *gatewaytypes.LimitValue {
	if v == nil {
		return nil
	}
	return &gatewaytypes.LimitValue{Size: v.Size, Count: v.Count}
}

func toGatewayLimitConfig(cfg limitConfigData) gatewaytypes.LimitConfig {
	return gatewaytypes.LimitConfig{
		Day:    toGatewayLimitValue(cfg.Day),
		Second: toGatewayLimitValue(cfg.Second),
	}
}

func generateKeyValue() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}