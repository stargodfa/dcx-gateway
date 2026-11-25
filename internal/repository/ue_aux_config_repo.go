package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/model/entity"

	drivermysql "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAuxConfigConflict = errors.New("anchor revision conflict")

type UEAuxConfigRepository interface {
	List(ctx context.Context, resource domain.AuxConfigResource) ([]entity.UEAuxConfigRecord, error)
	UpdateAtomic(ctx context.Context, resource domain.AuxConfigResource, ids []string, replace bool, mutate func(map[string]entity.UEAuxConfigRecord) ([]entity.UEAuxConfigRecord, error)) error
	Delete(ctx context.Context, resource domain.AuxConfigResource, ids []string) error
}

type ueAuxConfigRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

func NewUEAuxConfigRepository(db *gorm.DB, log *zap.Logger) UEAuxConfigRepository {
	return &ueAuxConfigRepository{db: db, log: log}
}

// Table and column names only come from this allowlist, never from HTTP input.
func auxConfigTable(resource domain.AuxConfigResource) (string, string, error) {
	switch resource {
	case domain.ModelAnchors:
		return "model_anchors", "anchor_id", nil
	case domain.SensorDevices:
		return "sensor_devices", "device_id", nil
	case domain.ClusterAgentsCatalog:
		return "cluster_agents_catalog", "node_id", nil
	case domain.ClusterAgentRegistry:
		return "cluster_agent_registry", "node_id", nil
	default:
		return "", "", fmt.Errorf("unsupported auxiliary config: %s", resource)
	}
}

func auxConfigQuery(db *gorm.DB, resource domain.AuxConfigResource) (*gorm.DB, string, error) {
	table, key, err := auxConfigTable(resource)
	if err != nil {
		return nil, "", err
	}
	columns := key + " AS id, payload, updated_at"
	if resource == domain.ModelAnchors {
		columns += ", instance_id, revision"
	}
	if resource == domain.ClusterAgentRegistry {
		columns += ", enabled"
	}
	return db.Table(table).Select(columns).Order(key), key, nil
}

func (r *ueAuxConfigRepository) List(ctx context.Context, resource domain.AuxConfigResource) ([]entity.UEAuxConfigRecord, error) {
	query, _, err := auxConfigQuery(r.db.WithContext(ctx), resource)
	if err != nil {
		return nil, err
	}
	var rows []entity.UEAuxConfigRecord
	if err := query.Find(&rows).Error; err != nil {
		r.logError(resource, err)
		return nil, err
	}
	return rows, nil
}

func (r *ueAuxConfigRepository) UpdateAtomic(
	ctx context.Context,
	resource domain.AuxConfigResource,
	ids []string,
	replace bool,
	mutate func(map[string]entity.UEAuxConfigRecord) ([]entity.UEAuxConfigRecord, error),
) error {
	table, key, err := auxConfigTable(resource)
	if err != nil {
		return err
	}
	if !replace && len(ids) == 0 {
		return nil
	}
	update := func(tx *gorm.DB) error {
		query, _, err := auxConfigQuery(tx, resource)
		if err != nil {
			return err
		}
		if !replace {
			query = query.Where(key+" IN ?", ids)
		}
		var rows []entity.UEAuxConfigRecord
		if err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&rows).Error; err != nil {
			return err
		}
		existing := make(map[string]entity.UEAuxConfigRecord, len(rows))
		for _, row := range rows {
			existing[row.ID] = row
		}
		next, err := mutate(existing)
		if err != nil {
			return err
		}
		// Consistent write order reduces deadlocks between batch requests.
		next = append([]entity.UEAuxConfigRecord(nil), next...)
		sort.Slice(next, func(i, j int) bool { return next[i].ID < next[j].ID })
		for _, row := range next {
			cur, exists := existing[row.ID]
			if exists && row.Payload == cur.Payload && row.Revision == cur.Revision && row.Enabled == cur.Enabled && row.InstanceID == cur.InstanceID {
				continue
			}
			values := map[string]any{"payload": row.Payload}
			if resource == domain.ModelAnchors {
				values["instance_id"] = row.InstanceID
				values["revision"] = row.Revision
			}
			if resource == domain.ClusterAgentRegistry {
				values["enabled"] = row.Enabled
			}
			if exists {
				err = tx.Table(table).Where(key+" = ?", row.ID).Updates(values).Error
			} else {
				values[key] = row.ID
				err = tx.Table(table).Create(values).Error
			}
			if err != nil {
				return err
			}
		}
		if replace {
			deleteQuery := tx.Table(table)
			if len(ids) > 0 {
				deleteQuery = deleteQuery.Where(key+" NOT IN ?", ids)
			} else {
				// Only an explicitly supplied empty collection may clear a domain.
				deleteQuery = deleteQuery.Where("1 = 1")
			}
			return deleteQuery.Delete(&entity.UEAuxConfigRecord{}).Error
		}
		return nil
	}
	for attempt := 0; attempt < 3; attempt++ {
		err = r.db.WithContext(ctx).Transaction(update)
		var transient *drivermysql.MySQLError
		if resource == domain.ModelAnchors || !errors.As(err, &transient) ||
			(transient.Number != 1062 && transient.Number != 1213 && transient.Number != 1205) || attempt == 2 {
			break
		}
		// Two first writers can contend on a missing row. Retry the rolled-back
		// transaction so last-write-wins and catalog merge semantics still hold.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 10 * time.Millisecond):
		}
	}
	// Concurrent creation can race on a missing row's gap lock. Surface the same
	// conflict as a stale revision rather than exposing a database error.
	var mysqlErr *drivermysql.MySQLError
	if resource == domain.ModelAnchors && errors.As(err, &mysqlErr) && (mysqlErr.Number == 1062 || mysqlErr.Number == 1213) {
		return ErrAuxConfigConflict
	}
	if err != nil && !errors.Is(err, ErrAuxConfigConflict) {
		r.logError(resource, err)
	}
	return err
}

func (r *ueAuxConfigRepository) Delete(ctx context.Context, resource domain.AuxConfigResource, ids []string) error {
	table, key, err := auxConfigTable(resource)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	err = r.db.WithContext(ctx).Table(table).Where(key+" IN ?", ids).Delete(&entity.UEAuxConfigRecord{}).Error
	if err != nil {
		r.logError(resource, err)
	}
	return err
}

func (r *ueAuxConfigRepository) logError(resource domain.AuxConfigResource, err error) {
	if r.log != nil {
		r.log.Error("auxiliary UE config database operation failed", zap.String("resource", string(resource)), zap.Error(err))
	}
}
