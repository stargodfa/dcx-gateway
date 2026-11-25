package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"dcx-gateway/internal/model/domain"
	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/repository"
)

type memoryAuxConfigRepository struct {
	rows map[domain.AuxConfigResource]map[string]entity.UEAuxConfigRecord
}

func (r *memoryAuxConfigRepository) List(_ context.Context, resource domain.AuxConfigResource) ([]entity.UEAuxConfigRecord, error) {
	rows := []entity.UEAuxConfigRecord{}
	for _, row := range r.rows[resource] {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}

func (r *memoryAuxConfigRepository) UpdateAtomic(_ context.Context, resource domain.AuxConfigResource, _ []string, replace bool, mutate func(map[string]entity.UEAuxConfigRecord) ([]entity.UEAuxConfigRecord, error)) error {
	current := map[string]entity.UEAuxConfigRecord{}
	for id, row := range r.rows[resource] {
		current[id] = row
	}
	next, err := mutate(current)
	if err != nil {
		return err
	}
	if r.rows == nil {
		r.rows = map[domain.AuxConfigResource]map[string]entity.UEAuxConfigRecord{}
	}
	if replace || r.rows[resource] == nil {
		r.rows[resource] = map[string]entity.UEAuxConfigRecord{}
	}
	for _, row := range next {
		r.rows[resource][row.ID] = row
	}
	return nil
}

func (r *memoryAuxConfigRepository) Delete(_ context.Context, resource domain.AuxConfigResource, ids []string) error {
	for _, id := range ids {
		delete(r.rows[resource], id)
	}
	return nil
}

func auxObject(t *testing.T, value string) map[string]any {
	t.Helper()
	result, err := decodeAuxPayload(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestAuxAnchorRevisionAndUnknownFields(t *testing.T) {
	svc := NewUEAuxConfigService(&memoryAuxConfigRepository{})
	ctx := context.Background()
	input := auxObject(t, `{"anchor_id":"a","instance_id":"i","extra":{"large":9007199254740993,"value":1.0}}`)
	saved, err := svc.Upsert(ctx, domain.ModelAnchors, "a", input)
	if err != nil || saved["revision"] != 1 {
		t.Fatalf("create = %v, %v", saved, err)
	}
	if _, exists := input["revision"]; exists {
		t.Fatal("service mutated caller record")
	}
	idempotent := auxObject(t, `{"anchor_id":"a","instance_id":"i","revision":1,"extra":{"large":9007199254740993,"value":1e0}}`)
	saved, err = svc.Upsert(ctx, domain.ModelAnchors, "a", idempotent)
	if err != nil || saved["revision"] != json.Number("1") {
		t.Fatalf("idempotent = %v, %v", saved, err)
	}
	changed := auxObject(t, `{"anchor_id":"a","instance_id":"i","revision":1,"extra":{"large":9007199254740993,"value":2}}`)
	saved, err = svc.Upsert(ctx, domain.ModelAnchors, "a", changed)
	if err != nil || saved["revision"] != 2 {
		t.Fatalf("update = %v, %v", saved, err)
	}
	if saved["extra"].(map[string]any)["large"] != json.Number("9007199254740993") {
		t.Fatal("unknown integer field lost precision")
	}
	for _, body := range []string{
		`{"anchor_id":"a","instance_id":"i"}`,
		`{"anchor_id":"a","instance_id":"i","revision":1}`,
	} {
		_, err := svc.Upsert(ctx, domain.ModelAnchors, "a", auxObject(t, body))
		if !errors.Is(err, repository.ErrAuxConfigConflict) {
			t.Fatalf("expected conflict: %v", err)
		}
	}
	for _, revision := range []string{`"1"`, `1.5`, `-1`, `true`, `2147483648`} {
		_, err := svc.Upsert(ctx, domain.ModelAnchors, "new", auxObject(t, `{"anchor_id":"new","revision":`+revision+`}`))
		if !errors.Is(err, ErrInvalidAuxConfig) {
			t.Fatalf("revision %s: %v", revision, err)
		}
	}
}

func TestAuxReplacementRequiresExplicitCollection(t *testing.T) {
	svc := NewUEAuxConfigService(&memoryAuxConfigRepository{})
	ctx := context.Background()
	for _, resource := range []domain.AuxConfigResource{domain.ModelAnchors, domain.SensorDevices, domain.ClusterAgentsCatalog, domain.ClusterAgentRegistry} {
		t.Run(string(resource), func(t *testing.T) {
			for _, body := range []string{`{}`, `{"` + resource.Collection() + `":null}`} {
				if _, err := svc.Replace(ctx, resource, auxObject(t, body)); !errors.Is(err, ErrInvalidAuxConfig) {
					t.Fatalf("body %s: %v", body, err)
				}
			}
			empty := `[]`
			if resource.Collection() == "agents" {
				empty = `{}`
			}
			result, err := svc.Replace(ctx, resource, auxObject(t, `{"`+resource.Collection()+`":`+empty+`}`))
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(result)
			if !strings.Contains(string(encoded), `"`+resource.Collection()+`":`+empty) {
				t.Fatalf("empty response = %s", encoded)
			}
		})
	}
	for _, body := range []string{
		`{"devices":[{"device_id":"a"},{"device_id":"a"}]}`,
		`{"devices":[{}]}`,
		`{"devices":[null]}`,
	} {
		if _, err := svc.Replace(ctx, domain.SensorDevices, auxObject(t, body)); !errors.Is(err, ErrInvalidAuxConfig) {
			t.Fatalf("body %s: %v", body, err)
		}
	}
}

func TestAuxCatalogMergeAndRegistryWhitelist(t *testing.T) {
	svc := NewUEAuxConfigService(&memoryAuxConfigRepository{})
	ctx := context.Background()
	_, err := svc.Upsert(ctx, domain.ClusterAgentsCatalog, "n", auxObject(t, `{"nodeId":"wrong","ip":"192.0.2.10","extension":{"x":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := svc.Upsert(ctx, domain.ClusterAgentsCatalog, "n", auxObject(t, `{"status":"offline"}`))
	if err != nil || saved["ip"] != "192.0.2.10" || saved["nodeId"] != "n" || saved["extension"] == nil {
		t.Fatalf("merge = %v, %v", saved, err)
	}
	result, err := svc.Replace(ctx, domain.ClusterAgentsCatalog, auxObject(t, `{"agents":{"n":{"status":"online"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if result["agents"].(map[string]any)["n"].(map[string]any)["ip"] != nil {
		t.Fatal("full replacement retained an omitted field")
	}
	result, err = svc.Replace(ctx, domain.ClusterAgentRegistry, auxObject(t, `{"agents":{"b":{"enabled":true},"a":{"enabled":true},"c":{"enabled":false}},"authorizedWhitelist":["c"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := result["authorizedWhitelist"].([]string); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("whitelist = %v", got)
	}
	result, err = svc.Upsert(ctx, domain.ClusterAgentRegistry, "a", auxObject(t, `{"enabled":false,"extra":42}`))
	if err != nil || len(result["authorizedWhitelist"].([]string)) != 1 {
		t.Fatalf("disable = %v, %v", result, err)
	}
	if err := svc.Delete(ctx, domain.ClusterAgentRegistry, auxObject(t, `{"node_id":"b","node_ids":["b","missing"]}`)); err != nil {
		t.Fatal(err)
	}
	result, err = svc.Get(ctx, domain.ClusterAgentRegistry)
	if err != nil || len(result["authorizedWhitelist"].([]string)) != 0 {
		t.Fatalf("delete = %v, %v", result, err)
	}
}
