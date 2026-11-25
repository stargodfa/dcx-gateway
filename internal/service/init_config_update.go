package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dcx-gateway/internal/model/protocol/ue"
)

var ErrInvalidInitConfigPatch = errors.New("invalid init_config patch")

type InitConfigUpdate struct {
	CameraID *string
	Position *ue.Vector3
	Rotation *ue.Rotation
	AutoLoad *bool
}

type initConfigPatch struct {
	cameraIDSet  bool
	cameraID     string
	positionSet  bool
	position     ue.Vector3
	rotationSet  bool
	rotation     ue.Rotation
	autoLoadSet  bool
	autoLoad     bool
	instanceMode initInstanceMode
	instanceIDs  []string
}

type initInstanceMode string

const (
	initInstanceModeNone    initInstanceMode = ""
	initInstanceModeAdd     initInstanceMode = "add"
	initInstanceModeReplace initInstanceMode = "replace"
	initInstanceModeDelete  initInstanceMode = "delete"
)

func (s *ueConfigService) UpdateInitConfig(ctx context.Context, patch InitConfigUpdate) (json.RawMessage, error) {
	if patch.CameraID == nil && patch.Position == nil && patch.Rotation == nil && patch.AutoLoad == nil {
		return nil, invalidInitPatch("no init_config fields to update")
	}
	internalPatch := initConfigPatch{}
	if patch.CameraID != nil {
		internalPatch.cameraIDSet = true
		internalPatch.cameraID = *patch.CameraID
	}
	if patch.Position != nil {
		internalPatch.positionSet = true
		internalPatch.position = *patch.Position
	}
	if patch.Rotation != nil {
		internalPatch.rotationSet = true
		internalPatch.rotation = *patch.Rotation
	}
	if patch.AutoLoad != nil {
		internalPatch.autoLoadSet = true
		internalPatch.autoLoad = *patch.AutoLoad
	}

	raw, err := s.rawConfigRepo.UpdateContentJSONAtomic(ctx, "init_config", func(raw []byte) ([]byte, error) {
		return updateInitConfigJSON(raw, internalPatch)
	})
	return json.RawMessage(raw), err
}

func (s *ueConfigService) AddInitInstanceIDs(ctx context.Context, ids []string) (json.RawMessage, error) {
	normalized, err := normalizeInstanceIDs(ids)
	if err != nil {
		return nil, err
	}
	raw, err := s.rawConfigRepo.UpdateContentJSONAtomic(ctx, "init_config", func(raw []byte) ([]byte, error) {
		return updateInitConfigJSON(raw, initConfigPatch{
			instanceMode: initInstanceModeAdd,
			instanceIDs:  normalized,
		})
	})
	return json.RawMessage(raw), err
}

func (s *ueConfigService) DeleteInitInstanceIDs(ctx context.Context, ids []string) (json.RawMessage, error) {
	raw, err := s.rawConfigRepo.UpdateContentJSONAtomic(ctx, "init_config", func(raw []byte) ([]byte, error) {
		return updateInitConfigJSON(raw, initConfigPatch{
			instanceMode: initInstanceModeDelete,
			instanceIDs:  ids,
		})
	})
	return json.RawMessage(raw), err
}

func updateInitConfigJSON(raw []byte, patch initConfigPatch) ([]byte, error) {
	root, err := decodeJSONObject(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid init_config json: %w", err)
	}
	if patch.cameraIDSet || patch.positionSet || patch.rotationSet {
		if err := applyInitCameraPatch(root, patch); err != nil {
			return nil, err
		}
	}
	if patch.autoLoadSet {
		root["auto_load_on_startup"] = patch.autoLoad
	}
	if patch.instanceMode != initInstanceModeNone {
		ids, err := normalizeInstanceIDs(patch.instanceIDs)
		if err != nil {
			return nil, err
		}
		applyInitInstanceIDsPatch(root, patch.instanceMode, ids)
	}
	next, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	return next, nil
}

func decodeJSONObject(raw []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	if root == nil {
		return nil, errors.New("root must be object")
	}
	return root, nil
}

func applyInitCameraPatch(root map[string]any, patch initConfigPatch) error {
	camera := ensureObject(root, "camera")
	if patch.cameraIDSet {
		cameraID := strings.TrimSpace(patch.cameraID)
		if cameraID == "" {
			return invalidInitPatch("camera_id is required")
		}
		camera["camera_id"] = cameraID
	}
	if patch.positionSet {
		camera["position"] = map[string]any{
			"x": patch.position.X,
			"y": patch.position.Y,
			"z": patch.position.Z,
		}
	}
	if patch.rotationSet {
		camera["rotation"] = map[string]any{
			"yaw":   patch.rotation.Yaw,
			"pitch": patch.rotation.Pitch,
			"roll":  patch.rotation.Roll,
		}
	}
	return nil
}

func ensureObject(root map[string]any, key string) map[string]any {
	if obj, ok := root[key].(map[string]any); ok {
		return obj
	}
	obj := make(map[string]any)
	root[key] = obj
	return obj
}

func applyInitInstanceIDsPatch(root map[string]any, mode initInstanceMode, ids []string) {
	current := readStringArray(root["instance_ids"])
	switch mode {
	case initInstanceModeAdd:
		seen := make(map[string]struct{}, len(current)+len(ids))
		next := make([]string, 0, len(current)+len(ids))
		for _, id := range current {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			next = append(next, id)
		}
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			next = append(next, id)
		}
		root["instance_ids"] = next
	case initInstanceModeReplace:
		root["instance_ids"] = ids
	case initInstanceModeDelete:
		remove := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			remove[id] = struct{}{}
		}
		next := make([]string, 0, len(current))
		for _, id := range current {
			if _, drop := remove[id]; drop {
				continue
			}
			next = append(next, id)
		}
		root["instance_ids"] = next
	}
}

func readStringArray(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func normalizeInstanceIDs(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, invalidInitPatch("instance_ids is required")
	}
	return out, nil
}

func invalidInitPatch(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInitConfigPatch, message)
}
