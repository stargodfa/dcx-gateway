package mapper

import (
	"testing"

	"dcx-gateway/internal/model/entity"
)

func TestUESceneDesiredModelsToConfig(t *testing.T) {
	boundsX, boundsY, boundsZ := 1.0, 2.0, 3.0
	cfg := UESceneDesiredModelsToConfig([]entity.UESceneDesiredModel{
		{
			InstanceID:    "discussion_room_1",
			ModelID:       "discussion_room",
			URI:           "file://model.udatasmith",
			Format:        "udatasmith",
			SourcePath:    "./model.udatasmith",
			DisplayName:   "discussion room",
			Visible:       true,
			BoundsX:       &boundsX,
			BoundsY:       &boundsY,
			BoundsZ:       &boundsZ,
			PositionX:     10,
			PositionY:     20,
			PositionZ:     30,
			RotationYaw:   90,
			RotationPitch: 1,
			RotationRoll:  2,
			ScaleX:        1,
			ScaleY:        1,
			ScaleZ:        1,
			ConfigJSON:    `{"custom_field":"keep"}`,
		},
	})

	if cfg.Version != 1 || len(cfg.SceneModelsDesired) != 1 {
		t.Fatalf("cfg = %#v", cfg)
	}
	model := cfg.SceneModelsDesired[0]
	if model.InstanceID != "discussion_room_1" || model.BoundsSize == nil || model.BoundsSize.X != 1 {
		t.Fatalf("model = %#v", model)
	}
	if model.Transform == nil || model.Transform.Position == nil || model.Transform.Position.X != 10 {
		t.Fatalf("transform = %#v", model.Transform)
	}
	if model.Extra["custom_field"] != "keep" {
		t.Fatalf("extra = %#v", model.Extra)
	}
}
