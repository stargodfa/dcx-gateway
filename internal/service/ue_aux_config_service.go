package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/repository"
)

var ErrInvalidAuxConfig = errors.New("invalid auxiliary config")

type UEAuxConfigService interface {
	Get(ctx context.Context, resource domain.AuxConfigResource) (map[string]any, error)
	Upsert(ctx context.Context, resource domain.AuxConfigResource, id string, record map[string]any) (map[string]any, error)
	Replace(ctx context.Context, resource domain.AuxConfigResource, document map[string]any) (map[string]any, error)
	Delete(ctx context.Context, resource domain.AuxConfigResource, request map[string]any) error
}

type ueAuxConfigService struct {
	repo repository.UEAuxConfigRepository
}

func NewUEAuxConfigService(repo repository.UEAuxConfigRepository) UEAuxConfigService {
	return &ueAuxConfigService{repo: repo}
}

func (s *ueAuxConfigService) Get(ctx context.Context, resource domain.AuxConfigResource) (map[string]any, error) {
	if resource.IDField() == "" {
		return nil, invalidAuxConfig("unsupported resource")
	}
	rows, err := s.repo.List(ctx, resource)
	if err != nil {
		return nil, err
	}
	return auxConfigDocument(resource, rows)
}

func auxConfigDocument(resource domain.AuxConfigResource, rows []entity.UEAuxConfigRecord) (map[string]any, error) {
	document := map[string]any{"version": 1}
	items := make([]map[string]any, 0, len(rows))
	agents := map[string]any{}
	whitelist := []string{}
	var updatedAt float64
	for _, row := range rows {
		record, err := decodeAuxPayload(row.Payload)
		if err != nil {
			return nil, err
		}
		items = append(items, record)
		agents[row.ID] = record
		if row.Enabled {
			whitelist = append(whitelist, row.ID)
		}
		if !row.UpdatedAt.IsZero() {
			ts := float64(row.UpdatedAt.UnixNano()) / 1e9
			if ts > updatedAt {
				updatedAt = ts
			}
		}
	}
	switch resource {
	case domain.ModelAnchors, domain.SensorDevices:
		document[resource.Collection()] = items
	case domain.ClusterAgentsCatalog:
		document["agents"] = agents
		document["updatedAt"] = updatedAt
	case domain.ClusterAgentRegistry:
		sort.Strings(whitelist)
		document["agents"] = agents
		document["authorizedWhitelist"] = whitelist
	}
	return document, nil
}

func (s *ueAuxConfigService) Upsert(ctx context.Context, resource domain.AuxConfigResource, id string, record map[string]any) (map[string]any, error) {
	if err := validateAuxID(resource, id); err != nil {
		return nil, err
	}
	if record == nil {
		return nil, invalidAuxConfig("body must be object")
	}
	if resource == domain.ModelAnchors || resource == domain.SensorDevices {
		if bodyID, _ := record[resource.IDField()].(string); bodyID != id {
			return nil, invalidAuxConfig("path %s does not match body %s", resource.IDField(), resource.IDField())
		}
	}
	var saved map[string]any
	err := s.repo.UpdateAtomic(ctx, resource, []string{id}, false, func(existing map[string]entity.UEAuxConfigRecord) ([]entity.UEAuxConfigRecord, error) {
		row, result, err := prepareAuxRecord(resource, id, record, existing, true)
		saved = result
		return []entity.UEAuxConfigRecord{row}, err
	})
	if err != nil {
		return nil, err
	}
	if resource == domain.ClusterAgentRegistry {
		document, err := s.Get(ctx, resource)
		if err != nil {
			return nil, err
		}
		return map[string]any{"agents": map[string]any{id: saved}, "authorizedWhitelist": document["authorizedWhitelist"]}, nil
	}
	return saved, nil
}

func (s *ueAuxConfigService) Replace(ctx context.Context, resource domain.AuxConfigResource, document map[string]any) (map[string]any, error) {
	records, ids, err := auxReplacementRecords(resource, document)
	if err != nil {
		return nil, err
	}
	var saved []entity.UEAuxConfigRecord
	err = s.repo.UpdateAtomic(ctx, resource, ids, true, func(existing map[string]entity.UEAuxConfigRecord) ([]entity.UEAuxConfigRecord, error) {
		saved = make([]entity.UEAuxConfigRecord, 0, len(ids))
		for i, id := range ids {
			row, _, err := prepareAuxRecord(resource, id, records[i], existing, false)
			if err != nil {
				return nil, err
			}
			saved = append(saved, row)
		}
		return saved, nil
	})
	if err != nil {
		return nil, err
	}
	result, err := auxConfigDocument(resource, saved)
	if resource == domain.ClusterAgentsCatalog {
		// The source bulk response does not include updatedAt; GET derives it from MySQL.
		delete(result, "updatedAt")
	}
	return result, err
}

func auxReplacementRecords(resource domain.AuxConfigResource, document map[string]any) ([]map[string]any, []string, error) {
	collection := resource.Collection()
	if collection == "" {
		return nil, nil, invalidAuxConfig("unsupported resource")
	}
	records := []map[string]any{}
	ids := []string{}
	if collection == "agents" {
		agents, ok := document[collection].(map[string]any)
		if !ok || agents == nil {
			return nil, nil, invalidAuxConfig("agents must be an object")
		}
		for id := range agents {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			record, ok := agents[id].(map[string]any)
			if !ok || record == nil {
				return nil, nil, invalidAuxConfig("agents[%s] must be an object", id)
			}
			records = append(records, record)
		}
	} else {
		items, ok := document[collection].([]any)
		if !ok || items == nil {
			return nil, nil, invalidAuxConfig("%s must be an array", collection)
		}
		for _, item := range items {
			record, ok := item.(map[string]any)
			if !ok || record == nil {
				return nil, nil, invalidAuxConfig("%s entries must be objects", collection)
			}
			id, _ := record[resource.IDField()].(string)
			ids = append(ids, id)
			records = append(records, record)
		}
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if err := validateAuxID(resource, id); err != nil {
			return nil, nil, err
		}
		if seen[id] {
			return nil, nil, invalidAuxConfig("duplicate %s: %s", resource.IDField(), id)
		}
		seen[id] = true
	}
	return records, ids, nil
}

func prepareAuxRecord(resource domain.AuxConfigResource, id string, input map[string]any, existing map[string]entity.UEAuxConfigRecord, merge bool) (entity.UEAuxConfigRecord, map[string]any, error) {
	// Round trip to clone nested fields and normalize numbers without losing precision.
	payload, err := json.Marshal(input)
	if err != nil {
		return entity.UEAuxConfigRecord{}, nil, invalidAuxConfig("invalid JSON record")
	}
	record, err := decodeAuxPayload(string(payload))
	if err != nil {
		return entity.UEAuxConfigRecord{}, nil, err
	}
	cur, exists := existing[id]
	if resource == domain.ClusterAgentsCatalog && exists && merge {
		previous, err := decodeAuxPayload(cur.Payload)
		if err != nil {
			return entity.UEAuxConfigRecord{}, nil, err
		}
		for key, value := range record {
			previous[key] = value
		}
		record = previous
	}
	record[resource.IDField()] = id
	row := entity.UEAuxConfigRecord{ID: id}
	if resource == domain.ModelAnchors {
		revision, specified, err := auxRevision(record["revision"])
		if err != nil {
			return row, nil, err
		}
		if !exists {
			if specified && revision != 0 && revision != 1 {
				return row, nil, repository.ErrAuxConfigConflict
			}
			row.Revision = 1
		} else {
			if !specified || revision != cur.Revision {
				return row, nil, repository.ErrAuxConfigConflict
			}
			previous, err := decodeAuxPayload(cur.Payload)
			if err != nil {
				return row, nil, err
			}
			if auxJSONEqual(record, previous) {
				return cur, previous, nil
			}
			if revision == 2147483647 {
				return row, nil, repository.ErrAuxConfigConflict
			}
			row.Revision = revision + 1
		}
		record["revision"] = row.Revision
		row.InstanceID, _ = record["instance_id"].(string)
		if utf8.RuneCountInString(row.InstanceID) > 128 {
			return row, nil, invalidAuxConfig("instance_id exceeds 128 characters")
		}
	}
	if resource == domain.ClusterAgentRegistry {
		row.Enabled, _ = record["enabled"].(bool)
	}
	payload, err = json.Marshal(record)
	row.Payload = string(payload)
	return row, record, err
}

func (s *ueAuxConfigService) Delete(ctx context.Context, resource domain.AuxConfigResource, request map[string]any) error {
	single := resource.IDField()
	if single == "nodeId" {
		single = "node_id"
	}
	if single == "" {
		return invalidAuxConfig("unsupported resource")
	}
	ids := []string{}
	if value, exists := request[single]; exists {
		id, ok := value.(string)
		if !ok {
			return invalidAuxConfig("%s must be a string", single)
		}
		if id != "" {
			ids = append(ids, id)
		}
	}
	if value, exists := request[single+"s"]; exists {
		items, ok := value.([]any)
		if !ok {
			return invalidAuxConfig("%ss must be an array", single)
		}
		for _, item := range items {
			id, ok := item.(string)
			if !ok {
				return invalidAuxConfig("%ss must contain strings", single)
			}
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	seen := map[string]bool{}
	unique := []string{}
	for _, id := range ids {
		if err := validateAuxID(resource, id); err != nil {
			return err
		}
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	if len(unique) == 0 {
		return invalidAuxConfig("%s or %ss required", single, single)
	}
	return s.repo.Delete(ctx, resource, unique)
}

func validateAuxID(resource domain.AuxConfigResource, id string) error {
	maxLength := 128
	if resource == domain.ModelAnchors {
		maxLength = 64
	}
	if resource.IDField() == "" || strings.TrimSpace(id) == "" || utf8.RuneCountInString(id) > maxLength {
		return invalidAuxConfig("%s must be nonempty and at most %d characters", resource.IDField(), maxLength)
	}
	return nil
}

func auxRevision(value any) (int, bool, error) {
	if value == nil {
		return 0, false, nil
	}
	number, ok := value.(json.Number)
	if !ok {
		return 0, true, invalidAuxConfig("revision must be a nonnegative integer")
	}
	n, err := number.Int64()
	if err != nil || n < 0 || n > 2147483647 {
		return 0, true, invalidAuxConfig("revision must be an integer between 0 and 2147483647")
	}
	return int(n), true, nil
}

func decodeAuxPayload(payload string) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	var record map[string]any
	if err := decoder.Decode(&record); err != nil {
		return nil, fmt.Errorf("decode auxiliary config payload: %w", err)
	}
	if record == nil {
		return nil, errors.New("stored auxiliary config payload must be an object")
	}
	return record, nil
}

func auxJSONEqual(left, right any) bool {
	switch l := left.(type) {
	case json.Number:
		r, ok := right.(json.Number)
		ln, lok := new(big.Rat).SetString(string(l))
		rn, rok := new(big.Rat).SetString(string(r))
		return ok && lok && rok && ln.Cmp(rn) == 0
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for key, value := range l {
			other, exists := r[key]
			if !exists || !auxJSONEqual(value, other) {
				return false
			}
		}
		return true
	case []any:
		r, ok := right.([]any)
		if !ok || len(l) != len(r) {
			return false
		}
		for i, value := range l {
			if !auxJSONEqual(value, r[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(left, right)
	}
}

func invalidAuxConfig(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidAuxConfig, fmt.Sprintf(format, args...))
}
