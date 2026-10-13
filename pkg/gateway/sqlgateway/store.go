package sqlgateway

import (
	"context"
	"encoding/json"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/sqlstore"
	"github.com/uptrace/bun"
)

var (
	ErrCodeIngestionKeyNotFound      = errors.MustNewCode("ingestion_key_not_found")
	ErrCodeIngestionKeyLimitNotFound = errors.MustNewCode("ingestion_key_limit_not_found")
)

type store struct {
	sqlstore sqlstore.SQLStore
}

func newStore(sqlstore sqlstore.SQLStore) *store {
	return &store{sqlstore: sqlstore}
}

type ingestionKey struct {
	bun.BaseModel `bun:"table:ingestion_keys"`

	ID        string `bun:"id,pk,type:text"`
	OrgID     string `bun:"org_id,notnull,type:text"`
	Name      string `bun:"name,notnull,type:text"`
	Value     string `bun:"value,notnull,type:text"`
	Tags      []byte `bun:"tags,type:blob"`
	ExpiresAt int64  `bun:"expires_at,type:integer"`
	CreatedAt int64  `bun:"created_at,notnull,type:integer"`
	UpdatedAt int64  `bun:"updated_at,notnull,type:integer"`
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

func (s *store) createKey(ctx context.Context, key *ingestionKey) error {
	_, err := s.sqlstore.BunDB().NewInsert().Model(key).Exec(ctx)
	if err != nil {
		return s.sqlstore.WrapAlreadyExistsErrf(err, errors.CodeAlreadyExists, "ingestion key with id %s already exists", key.ID)
	}
	return nil
}

func (s *store) getKey(ctx context.Context, orgID, keyID string) (*ingestionKey, error) {
	var key ingestionKey
	err := s.sqlstore.BunDB().NewSelect().Model(&key).
		Where("org_id = ? AND id = ?", orgID, keyID).
		Scan(ctx)
	if err != nil {
		return nil, s.sqlstore.WrapNotFoundErrf(err, ErrCodeIngestionKeyNotFound, "ingestion key %s not found", keyID)
	}
	return &key, nil
}

func (s *store) listKeys(ctx context.Context, orgID string, offset, limit int) ([]ingestionKey, int, error) {
	var keys []ingestionKey
	total, err := s.sqlstore.BunDB().NewSelect().Model(&keys).
		Where("org_id = ?", orgID).
		Count(ctx)
	if err != nil {
		return nil, 0, errors.WrapInternalf(err, errors.CodeInternal, "error counting ingestion keys")
	}

	err = s.sqlstore.BunDB().NewSelect().Model(&keys).
		Where("org_id = ?", orgID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, 0, errors.WrapInternalf(err, errors.CodeInternal, "error listing ingestion keys")
	}

	return keys, total, nil
}

func (s *store) searchKeysByName(ctx context.Context, orgID, name string, offset, limit int) ([]ingestionKey, int, error) {
	var keys []ingestionKey
	total, err := s.sqlstore.BunDB().NewSelect().Model(&keys).
		Where("org_id = ?", orgID).
		Where("name LIKE ?", "%"+name+"%").
		Count(ctx)
	if err != nil {
		return nil, 0, errors.WrapInternalf(err, errors.CodeInternal, "error counting ingestion keys")
	}

	err = s.sqlstore.BunDB().NewSelect().Model(&keys).
		Where("org_id = ?", orgID).
		Where("name LIKE ?", "%"+name+"%").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, 0, errors.WrapInternalf(err, errors.CodeInternal, "error searching ingestion keys")
	}

	return keys, total, nil
}

func (s *store) updateKey(ctx context.Context, orgID, keyID, name string, tags []byte, expiresAt int64, now int64) error {
	res, err := s.sqlstore.BunDB().NewUpdate().
		Model((*ingestionKey)(nil)).
		Set("name = ?, tags = ?, expires_at = ?, updated_at = ?", name, tags, expiresAt, now).
		Where("id = ?", keyID).
		Where("org_id = ?", orgID).
		Exec(ctx)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error updating ingestion key")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error verifying updated ingestion key")
	}
	if rowsAffected == 0 {
		return errors.NewNotFoundf(ErrCodeIngestionKeyNotFound, "ingestion key %s not found", keyID)
	}

	return nil
}

func (s *store) deleteKey(ctx context.Context, orgID, keyID string) error {
	tx, err := s.sqlstore.BunDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error beginning transaction to delete ingestion key")
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.NewDelete().
		Model((*ingestionKeyLimit)(nil)).
		Where("key_id = ?", keyID).
		Exec(ctx)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error deleting ingestion key limits")
	}

	res, err := tx.NewDelete().
		Model((*ingestionKey)(nil)).
		Where("id = ?", keyID).
		Where("org_id = ?", orgID).
		Exec(ctx)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error deleting ingestion key")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error verifying deleted ingestion key")
	}
	if rowsAffected == 0 {
		return errors.NewNotFoundf(ErrCodeIngestionKeyNotFound, "ingestion key %s not found", keyID)
	}

	return tx.Commit()
}

func (s *store) createLimit(ctx context.Context, limit *ingestionKeyLimit) error {
	_, err := s.sqlstore.BunDB().NewInsert().Model(limit).Exec(ctx)
	if err != nil {
		return s.sqlstore.WrapAlreadyExistsErrf(err, errors.CodeAlreadyExists, "ingestion key limit with id %s already exists", limit.ID)
	}
	return nil
}

func (s *store) getLimit(ctx context.Context, orgID, limitID string) (*ingestionKeyLimit, error) {
	var limit ingestionKeyLimit
	err := s.sqlstore.BunDB().NewSelect().
		Model(&limit).
		Join("JOIN ingestion_keys ON ingestion_key_limit.key_id = ingestion_keys.id").
		Where("ingestion_keys.org_id = ?", orgID).
		Where("ingestion_key_limit.id = ?", limitID).
		Scan(ctx)
	if err != nil {
		return nil, s.sqlstore.WrapNotFoundErrf(err, ErrCodeIngestionKeyLimitNotFound, "ingestion key limit %s not found", limitID)
	}
	return &limit, nil
}

func (s *store) listLimits(ctx context.Context, orgID, keyID string) ([]ingestionKeyLimit, error) {
	var limits []ingestionKeyLimit
	err := s.sqlstore.BunDB().NewSelect().
		Model(&limits).
		Join("JOIN ingestion_keys ON ingestion_key_limit.key_id = ingestion_keys.id").
		Where("ingestion_keys.org_id = ?", orgID).
		Where("ingestion_key_limit.key_id = ?", keyID).
		Order("ingestion_key_limit.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "error listing ingestion key limits")
	}
	return limits, nil
}

func (s *store) getKeyLimitsByKeyIDs(ctx context.Context, orgID string, keyIDs []string) (map[string][]ingestionKeyLimit, error) {
	limitsByKeyIDs := make(map[string][]ingestionKeyLimit, len(keyIDs))
	if len(keyIDs) == 0 {
		return limitsByKeyIDs, nil
	}

	var limits []ingestionKeyLimit
	err := s.sqlstore.BunDB().NewSelect().
		Model(&limits).
		Join("JOIN ingestion_keys ON ingestion_key_limit.key_id = ingestion_keys.id").
		Where("ingestion_keys.org_id = ?", orgID).
		Where("ingestion_key_limit.key_id IN (?)", bun.In(keyIDs)).
		Order("ingestion_key_limit.created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, errors.WrapInternalf(err, errors.CodeInternal, "error listing ingestion key limits")
	}

	for _, limit := range limits {
		limitsByKeyIDs[limit.KeyID] = append(limitsByKeyIDs[limit.KeyID], limit)
	}

	return limitsByKeyIDs, nil
}

func (s *store) updateLimit(ctx context.Context, orgID, limitID string, config []byte, tags []byte, now int64) error {
	res, err := s.sqlstore.BunDB().NewUpdate().
		Model((*ingestionKeyLimit)(nil)).
		Set("config = ?, tags = ?, updated_at = ?", config, tags, now).
		Where("ingestion_key_limit.id = ?", limitID).
		Where("ingestion_key_limit.key_id IN (SELECT id FROM ingestion_keys WHERE org_id = ?)", orgID).
		Exec(ctx)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error updating ingestion key limit")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error verifying updated ingestion key limit")
	}
	if rowsAffected == 0 {
		return errors.NewNotFoundf(ErrCodeIngestionKeyLimitNotFound, "ingestion key limit %s not found", limitID)
	}

	return nil
}

func (s *store) deleteLimit(ctx context.Context, orgID, limitID string) error {
	res, err := s.sqlstore.BunDB().NewDelete().
		Model((*ingestionKeyLimit)(nil)).
		Where("ingestion_key_limit.id = ?", limitID).
		Where("ingestion_key_limit.key_id IN (SELECT id FROM ingestion_keys WHERE org_id = ?)", orgID).
		Exec(ctx)
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error deleting ingestion key limit")
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.WrapInternalf(err, errors.CodeInternal, "error verifying deleted ingestion key limit")
	}
	if rowsAffected == 0 {
		return errors.NewNotFoundf(ErrCodeIngestionKeyLimitNotFound, "ingestion key limit %s not found", limitID)
	}

	return nil
}

func encodeTags(tags []string) []byte {
	if tags == nil {
		return nil
	}
	data, _ := json.Marshal(tags)
	return data
}

func decodeTags(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	var tags []string
	_ = json.Unmarshal(data, &tags)
	return tags
}

func encodeLimitConfig(cfg limitConfigData) []byte {
	data, _ := json.Marshal(cfg)
	return data
}

func decodeLimitConfig(data []byte) limitConfigData {
	var cfg limitConfigData
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

type limitConfigData struct {
	Day    *limitValueData `json:"day,omitempty"`
	Second *limitValueData `json:"second,omitempty"`
}

type limitValueData struct {
	Size  *int64 `json:"size,omitempty"`
	Count *int64 `json:"count,omitempty"`
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

func fromMillis(ms int64) time.Time {
	return time.UnixMilli(ms)
}
