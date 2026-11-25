-- Synthetic public demo configuration only. No customer scene, route, host or media data.
-- nDisplay documents are placeholders and must be configured for a local UE setup.

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('chart_binding', 'chart_binding.json', '{
  "version": 1,
  "bindings": [
    {
      "binding_id": "sim_3",
      "chart_id": "chart_h"
    },
    {
      "binding_id": "sim_4",
      "chart_id": "chart_l"
    },
    {
      "binding_id": "energy.total.today",
      "chart_id": "chart_energy_total"
    },
    {
      "binding_id": "energy.building.snapshot",
      "chart_id": "chart_energy_building"
    },
    {
      "binding_id": "education.classroom.usage",
      "chart_id": "chart_education_usage"
    },
    {
      "binding_id": "education.classroom.schedule",
      "chart_id": "chart_education_schedule"
    }
  ]
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('chart_config', 'chart_config.json', '{
  "version": 1,
  "overlay_hud_max_panel_width": 920,
  "charts": [
    {
      "chart_id": "chart_h",
      "chart_name": "演示访问统计",
      "visible": true,
      "rows": 2,
      "cols": 3,
      "cell_values": [
        [
          "指标",
          "数值",
          "单位"
        ],
        [
          "演示指标",
          "",
          ""
        ]
      ]
    },
    {
      "chart_id": "chart_l",
      "chart_name": "演示告警状态",
      "visible": true,
      "rows": 2,
      "cols": 3,
      "cell_values": [
        [
          "指标",
          "数值",
          "单位"
        ],
        [
          "演示指标",
          "",
          ""
        ]
      ]
    },
    {
      "chart_id": "chart_energy_total",
      "chart_name": "演示能耗",
      "visible": true,
      "rows": 2,
      "cols": 3,
      "cell_values": [
        [
          "指标",
          "数值",
          "单位"
        ],
        [
          "演示指标",
          "",
          ""
        ]
      ]
    },
    {
      "chart_id": "chart_energy_building",
      "chart_name": "演示楼栋排行",
      "visible": true,
      "rows": 2,
      "cols": 3,
      "cell_values": [
        [
          "指标",
          "数值",
          "单位"
        ],
        [
          "演示指标",
          "",
          ""
        ]
      ]
    },
    {
      "chart_id": "chart_education_usage",
      "chart_name": "演示使用率",
      "visible": true,
      "rows": 2,
      "cols": 3,
      "cell_values": [
        [
          "指标",
          "数值",
          "单位"
        ],
        [
          "演示指标",
          "",
          ""
        ]
      ]
    },
    {
      "chart_id": "chart_education_schedule",
      "chart_name": "演示课程状态",
      "visible": true,
      "rows": 2,
      "cols": 3,
      "cell_values": [
        [
          "指标",
          "数值",
          "单位"
        ],
        [
          "演示指标",
          "",
          ""
        ]
      ]
    }
  ]
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('drone_config', 'drone_config.json', '{
  "version": 1,
  "player_camera_view_mode": "drone",
  "proximity": {
    "enabled": false,
    "load_distance_mm": 10000,
    "unload_hysteresis_mm": 1000
  },
  "position_report": {
    "enabled": false,
    "interval_s": 1
  },
  "flight": {
    "move_speed_cm_s": 300,
    "shift_speed_multiplier": 2,
    "ctrl_speed_multiplier": 0.5,
    "collision_probe_radius_cm": 20,
    "collision_skin_cm": 2,
    "map_bounds_half_extent_cm": 10000,
    "map_bounds_max_height_above_ground_cm": 10000,
    "ground_plane_world_z": 0,
    "penetration_wheel_speed_cm": 10,
    "min_clearance_above_cesium_surface_cm": 100
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('init_config', 'init_config.json', '{
  "version": 1,
  "camera": {
    "camera_id": "demo_camera",
    "position": {
      "x": 0,
      "y": 0,
      "z": 200
    },
    "rotation": {
      "yaw": 0,
      "pitch": 0,
      "roll": 0
    }
  },
  "auto_load_on_startup": false,
  "instance_ids": [
    "demo_model_1"
  ]
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('model_config', 'model_config.json', '{
  "version": 1,
  "last_load_by_instance_id": {
    "demo_model_1": {
      "model_id": "demo_model",
      "visible": true,
      "transform": {
        "position": {
          "x": 0,
          "y": 0,
          "z": 0
        },
        "rotation": {
          "yaw": 0,
          "pitch": 0,
          "roll": 0
        },
        "scale": {
          "x": 1,
          "y": 1,
          "z": 1
        }
      }
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('model_table', 'model_table.json', '{
  "version": 1,
  "entries": [
    {
      "instance_id": "demo_model_1",
      "model_id": "demo_model",
      "display_name": "演示模型"
    }
  ]
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('path_config', 'path_config.json', '{
  "version": 1,
  "nav_playback": {
    "use_duration": true,
    "planned_duration_s": 10,
    "planned_speed_cm_s": 100
  },
  "route_display": {
    "ground_offset_below_camera_cm": 0,
    "drone_offset_below_camera_cm": 0
  },
  "plans_by_id": {
    "demo_path": {
      "path_id": "demo_path",
      "name": "演示路径",
      "waypoints": [
        {
          "x": 0,
          "y": 0,
          "z": 0
        },
        {
          "x": 100,
          "y": 100,
          "z": 0
        }
      ]
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('render_config', 'render_config.json', '{
  "version": 1,
  "enabled": false,
  "settings": {
    "shadows": true,
    "ao": true,
    "bloom": false,
    "eye_adaptation": false,
    "extend_luminance_range": false,
    "local_exposure_highlight": 1,
    "local_exposure_shadow": 1,
    "exposure_bias": 0,
    "tonemapper_gamma": 2.2,
    "contrast": 1,
    "saturation": 1,
    "sharpen": 0,
    "vignette": 0
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('report_scripts', 'report_scripts.json', '{
  "version": 1,
  "scripts": [
    {
      "script_id": "security_main",
      "label": "演示汇报",
      "chapters": [
        {
          "chapter_id": "demo_chapter",
          "label": "演示章节",
          "scenes": [
            {
              "scene_id": "demo_scene",
              "label": "演示场景",
              "camera": {
                "camera_id": "demo_camera",
                "position": {
                  "x": 0,
                  "y": 0,
                  "z": 200
                },
                "rotation": {
                  "yaw": 0,
                  "pitch": 0,
                  "roll": 0
                }
              },
              "bindings": [
                {
                  "instance_id": "demo_model_1",
                  "model_id": "demo_model",
                  "visible": true,
                  "transform": {
                    "position": {
                      "x": 0,
                      "y": 0,
                      "z": 0
                    },
                    "rotation": {
                      "yaw": 0,
                      "pitch": 0,
                      "roll": 0
                    },
                    "scale": {
                      "x": 1,
                      "y": 1,
                      "z": 1
                    }
                  }
                }
              ],
              "warmup": {
                "data": [
                  "security_main"
                ],
                "videos": [],
                "charts": [
                  "sim_3",
                  "sim_4"
                ]
              }
            }
          ]
        }
      ]
    }
  ]
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('video_streams', 'video_streams.json', '{
  "version": 1,
  "streams": []
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('ndisplay/cluster_1node_2viewport.ndisplay.template', 'cluster_1node_2viewport.ndisplay.template', '{
  "nDisplay": {
    "version": "5.4",
    "description": "Synthetic placeholder; supply your own local nDisplay configuration.",
    "scene": {
      "cameras": {},
      "screens": {},
      "xforms": {}
    },
    "cluster": {
      "nodes": {}
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('ndisplay/cluster_2node_1x2.ndisplay.template', 'cluster_2node_1x2.ndisplay.template', '{
  "nDisplay": {
    "version": "5.4",
    "description": "Synthetic placeholder; supply your own local nDisplay configuration.",
    "scene": {
      "cameras": {},
      "screens": {},
      "xforms": {}
    },
    "cluster": {
      "nodes": {}
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('ndisplay/cluster_3node_1x2_viewer.ndisplay.template', 'cluster_3node_1x2_viewer.ndisplay.template', '{
  "nDisplay": {
    "version": "5.4",
    "description": "Synthetic placeholder; supply your own local nDisplay configuration.",
    "scene": {
      "cameras": {},
      "screens": {},
      "xforms": {}
    },
    "cluster": {
      "nodes": {}
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('ndisplay/cluster_3node_left.ndisplay.template', 'cluster_3node_left.ndisplay.template', '{
  "nDisplay": {
    "version": "5.4",
    "description": "Synthetic placeholder; supply your own local nDisplay configuration.",
    "scene": {
      "cameras": {},
      "screens": {},
      "xforms": {}
    },
    "cluster": {
      "nodes": {}
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);

INSERT INTO ue_config_documents
(config_key, file_name, content_json, source, enabled, version)
VALUES
('ndisplay/cluster_3node_right.ndisplay.template', 'cluster_3node_right.ndisplay.template', '{
  "nDisplay": {
    "version": "5.4",
    "description": "Synthetic placeholder; supply your own local nDisplay configuration.",
    "scene": {
      "cameras": {},
      "screens": {},
      "xforms": {}
    },
    "cluster": {
      "nodes": {}
    }
  }
}', 'public_demo', 1, 'v1')
ON DUPLICATE KEY UPDATE
file_name=VALUES(file_name), content_json=VALUES(content_json),
source=VALUES(source), enabled=VALUES(enabled), version=VALUES(version);
