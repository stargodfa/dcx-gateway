package repository

import (
	"database/sql"
	"strings"
	"testing"

	"dcx-gateway/internal/model/entity"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestPlacementUpsertMapsIncludeFalseAndZeroValues(t *testing.T) {
	model := entity.UEModelInstancePlacement{
		InstanceID:    "discussion_room_1",
		Visible:       false,
		PositionX:     0,
		PositionY:     0,
		PositionZ:     0,
		RotationYaw:   0,
		RotationPitch: 0,
		RotationRoll:  0,
		ScaleX:        0,
		ScaleY:        0,
		ScaleZ:        0,
	}

	insert := placementInsertMap(model)
	assertMapValue(t, insert, "instance_id", "discussion_room_1")
	assertMapValue(t, insert, "visible", false)
	assertMapValue(t, insert, "position_x", float64(0))
	assertMapValue(t, insert, "position_y", float64(0))
	assertMapValue(t, insert, "position_z", float64(0))
	assertMapValue(t, insert, "rotation_yaw", float64(0))
	assertMapValue(t, insert, "rotation_pitch", float64(0))
	assertMapValue(t, insert, "rotation_roll", float64(0))
	assertMapValue(t, insert, "scale_x", float64(0))
	assertMapValue(t, insert, "scale_y", float64(0))
	assertMapValue(t, insert, "scale_z", float64(0))

	update := placementUpdateMap(model)
	if _, ok := update["instance_id"]; ok {
		t.Fatalf("update map should not mutate primary key")
	}
	assertMapValue(t, update, "visible", false)
	assertMapValue(t, update, "position_x", float64(0))
	assertMapValue(t, update, "rotation_yaw", float64(0))
	assertMapValue(t, update, "scale_x", float64(0))
}

func TestDefaultPlacementRowUsesLockableDefaults(t *testing.T) {
	model := defaultPlacementRow("discussion_room_1")
	insert := placementInsertMap(model)

	assertMapValue(t, insert, "instance_id", "discussion_room_1")
	assertMapValue(t, insert, "visible", true)
	assertMapValue(t, insert, "position_x", float64(0))
	assertMapValue(t, insert, "position_y", float64(0))
	assertMapValue(t, insert, "position_z", float64(0))
	assertMapValue(t, insert, "rotation_yaw", float64(0))
	assertMapValue(t, insert, "rotation_pitch", float64(0))
	assertMapValue(t, insert, "rotation_roll", float64(0))
	assertMapValue(t, insert, "scale_x", float64(1))
	assertMapValue(t, insert, "scale_y", float64(1))
	assertMapValue(t, insert, "scale_z", float64(1))
}

func TestDefaultPlacementConflictClauseMySQLDryRunHasAssignment(t *testing.T) {
	conn, err := sql.Open("mysql", "gorm:gorm@tcp(127.0.0.1:9910)/gorm?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		t.Fatalf("sql.Open returned error: %v", err)
	}
	defer conn.Close()

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      conn,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("gorm.Open mysql dry run returned error: %v", err)
	}

	result := db.Table(entity.UEModelInstancePlacement{}.TableName()).
		Clauses(defaultPlacementConflictClause()).
		Create(placementInsertMap(defaultPlacementRow("discussion_room_1")))
	if result.Error != nil {
		t.Fatalf("dry run create returned error: %v", result.Error)
	}
	stmt := result.Statement
	sql := strings.TrimSpace(stmt.SQL.String())
	duplicate := "ON DUPLICATE KEY UPDATE"
	if !strings.Contains(sql, duplicate) {
		t.Fatalf("sql = %s, want %s", sql, duplicate)
	}
	if strings.HasSuffix(sql, duplicate) {
		t.Fatalf("sql has empty duplicate-key update clause: %s", sql)
	}
	after := strings.TrimSpace(sql[strings.Index(sql, duplicate)+len(duplicate):])
	if after == "" {
		t.Fatalf("sql has no duplicate-key assignment: %s", sql)
	}
	if !strings.Contains(after, "instance_id") {
		t.Fatalf("duplicate-key assignment = %q, want instance_id no-op", after)
	}
}

func assertMapValue[T comparable](t *testing.T, values map[string]any, key string, want T) {
	t.Helper()
	got, ok := values[key]
	if !ok {
		t.Fatalf("missing key %q in %#v", key, values)
	}
	if got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
