# 模型配置接口使用指南

本文档面向 UE 工程师、Server/FastAPI 工程师和现场联调人员，说明当前模型配置接口如何使用。

## 三层模型关系

模型配置被拆成三层：

| 层级 | 管理对象 | 主要接口 | 数据表 |
|---|---|---|---|
| model-types | `model_id` 资产、URI、格式、基础 bounds | `/api/ue/config/model-types` | `ue_model_types` |
| model-table | `instance_id` 目录、显示名、排序、启用状态 | `/api/ue/config/model-table` | `ue_model_instances` |
| model-config placements | `visible / position / rotation / scale` | `/api/ue/config/model-config/placements` | `ue_model_instance_placements` |

`scene-desired` 是 generated 视图，由三表 JOIN 生成，不直接写。

## 数据库表对应关系

- `ue_model_types`: model_id 资产权威。
- `ue_model_instances`: instance_id 目录权威。
- `ue_model_instance_placements`: 位姿和显隐权威。

`bounds_size` 不在 placement 表保存，输出时按：

```text
bounds_size = base_bounds_size * abs(scale)
```

计算。

## 典型操作

新库中的模型类型为 `demo_model`，实例为 `demo_model_1`。下面的新增示例使用 `new_type` / `new_instance_1`，需要按顺序先创建；后文鉴权验证使用已经初始化的 `demo_model_1`。

写接口需要鉴权。先在私有配置中设置随机静态 token 并启用 `security.static_token_enabled`，再在同一个 Bash 会话中设置：

```bash
export GATEWAY_URL="http://127.0.0.1:8080"
read -r -s -p '输入私有配置中的 Gateway token: ' GATEWAY_TOKEN
export GATEWAY_TOKEN
```

如采用 Bearer token，先自行创建登录用户，再将静态 token 请求头替换为 `Authorization: Bearer <token>`。公开种子不创建默认账号。

### 新增模型类型

```bash
curl -X POST "${GATEWAY_URL}/api/ue/config/model-types/create" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"model_id":"new_type","uri":"file://./models/new/new.udatasmith","format":"udatasmith","source_path":"./models/new/new.udatasmith","base_bounds_size":{"x":1,"y":2,"z":3}}'
```

### 新增实例

```bash
curl -X POST "${GATEWAY_URL}/api/ue/config/model-table/entries/create" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1","model_id":"new_type","display_name":"新实例 01","sort_order":100}'
```

新增实例不会自动加入 `init_config.instance_ids`。

### 修改实例显示名

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-table/entries" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1","display_name":"新实例 01 已更新"}'
```

### 修改实例位置

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/new_instance_1" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"transform":{"position":{"x":100,"y":200,"z":300}}}'
```

### 修改 visible=false

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/new_instance_1" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"visible":false}'
```

### 重置 placement

```bash
curl -X DELETE "${GATEWAY_URL}/api/ue/config/model-config/placements/delete" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1"}'
```

这只删除 placement 行。如果 instance 仍 enabled，查询时会补默认 placement。

### 禁用实例

```bash
curl -X DELETE "${GATEWAY_URL}/api/ue/config/model-table/entries/delete" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1"}'
```

禁用后，该 instance 不再出现在 model-table、model-config generated/runtime、scene-desired generated 中。

### 恢复实例 enabled=true

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-table/entries" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1","enabled":true}'
```

### 加入 init_config 启动白名单

```bash
curl -X POST "${GATEWAY_URL}/api/ue/config/init-config/instance_ids/create" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1"}'
```

### 移出 init_config 启动白名单

```bash
curl -X DELETE "${GATEWAY_URL}/api/ue/config/init-config/instance_ids/delete" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"new_instance_1"}'
```

### 读取 scene-desired

```bash
curl "${GATEWAY_URL}/api/ue/config/scene-desired?mode=generated"
```

## 删除接口区别表

| 接口 | 含义 |
|---|---|
| `DELETE /api/ue/config/init-config/instance_ids/delete` | 删除启动白名单项。 |
| `DELETE /api/ue/config/model-config/placements/delete` | 删除 placement 行，查询时仍可能默认补位姿。 |
| `DELETE /api/ue/config/model-table/entries/delete` | 禁用实例，让实例从 generated 结果中消失。 |
| `DELETE /api/ue/config/model-types/delete` | 禁用模型类型；如果仍有启用实例引用则返回 409。 |

## 验证 placement 修改

不要用：

```http
GET /api/ue/config/model-config
```

验证 placement，因为默认可能是 raw/legacy 文档。

应该用：

```http
GET /api/ue/config/model-config?mode=structured
GET /api/ue/config/model-config?mode=runtime
GET /api/ue/config/scene-desired?mode=generated
```
## 鉴权请求示例

P1-1 起，模型配置写接口需要鉴权。推荐使用登录后返回的 Bearer token：

```bash
curl -X POST "${GATEWAY_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"operator","password":"<your-password>"}'
```

然后调用写接口：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"visible":false}'
```

内网联调兜底也可以使用静态 token：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Token: <your-private-token>" \
  -d '{"visible":false}'
```

`security.protect_read=false` 时读取接口仍可无 token 调用，便于 UE/Server 联调；生产如需收紧，可改为 `protect_read=true`。

写接口联调时建议带 `X-Request-Id`：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-Id: ue-placement-test-001" \
  -d '{"visible":false}'
```

响应头会回传同一个 `X-Request-Id`，网关请求日志也会带该字段，方便现场排查。
