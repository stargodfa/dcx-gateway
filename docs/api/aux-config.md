# 补充配置接口

dcx-gateway 已整合辅助网关的四类配置、16 个接口。沿用当前工程的 Gin、GORM、Zap、YAML 配置、鉴权、请求追踪及 `bootstrap → handler → service → repository` 分层。

## 功能与存储

| 配置域 | 作用 | MySQL 表 | 写入规则 |
|---|---|---|---|
| `model-anchors` | 模型锚点及传感器绑定信息 | `model_anchors` | revision 乐观锁；相同内容重复写入不递增版本 |
| `sensor-devices` | 传感器设备清单及状态 | `sensor_devices` | 最后一次写入生效，单条写入覆盖该记录 |
| `cluster-agents-catalog` | 集群 Agent 最后已知快照 | `cluster_agents_catalog` | 单条写入合并字段，全量写入替换快照 |
| `cluster-agent-registry` | Agent 授权注册信息 | `cluster_agent_registry` | 单条写入覆盖该记录；由 `enabled=true` 派生白名单 |

所有记录保留未知 JSON 字段和嵌套内容，JSON 解码使用 `json.Number`，避免把大整数强制转换为 float64。锚点与设备、模型实例的引用不额外做外键校验；Gateway 不直接控制 UE 或 Agent，也不接管 Server 的运行态广播。

## 部署前准备

在当前 `configs/config.dev.yaml` 的 `mysql.dsn` 所指数据库执行增量脚本：

```bash
mysql -h <host> -P <port> -u <user> -p <current_database> < migrations/014_init_ue_aux_configs.sql
```

脚本只通过 `CREATE TABLE IF NOT EXISTS` 创建四张新表，可以重复执行。沿用当前工程的手动 SQL migration 方式，应用启动不会自动修改数据库结构。ARM64 打包脚本会自动包含该 migration。

继续使用 `go run ./cmd/dcx-gateway`、现有 YAML、MySQL/Redis、HTTP 端口及 systemd 部署，不需要额外运行源网关的 `main.go`、`.env` 或独立 Compose 服务。

## 鉴权与响应

- 路径前缀为 `/api/ue/config`，与源网关一致。
- 成功直接返回 JSON 对象，不包 `code/data`；失败返回 `{"error":"..."}`。
- 沿用 `security.enabled`、`security.protect_read`、Bearer Session 和 `X-Gateway-Token`。启用鉴权时，PUT/DELETE 必须提供有效凭据；GET 是否鉴权由 `protect_read` 决定。
- 沿用 `X-Request-Id` / `X-Trace-Id` 请求追踪。
- 请求体大小使用 `server.max_body_bytes`，超出返回 413。
- 无数据时返回 `version:1` 及空数组/对象，白名单始终返回数组。

源 Server 的 `server/infrastructure/aux_gateway_client.py` 当前不发送鉴权凭据。因此切换到本工程前，需要在该客户端添加有效的 `Authorization: Bearer ...` 或 `X-Gateway-Token` 请求头，然后将 `DATA_AUX_GATEWAY_URL` 指向当前网关；仅修改 URL 会使受保护的写接口返回 401。客户端属于源工程，本次融合未修改它。

## 接口清单

以下路径均带上述前缀。请求与响应详情也已登记到 [OpenAPI](openapi.yaml)，可通过现有 Swagger UI 查看。

| 方法 | 路径 | 功能 |
|---|---|---|
| GET | `/model-anchors` | 全量读取锚点 |
| PUT | `/model-anchors/{anchor_id}` | 新建或更新单个锚点 |
| PUT | `/model-anchors` | 原子替换全部锚点 |
| DELETE | `/model-anchors/delete` | 单个或批量删除锚点 |
| GET | `/sensor-devices` | 全量读取设备 |
| PUT | `/sensor-devices/{device_id}` | 新建或更新单个设备 |
| PUT | `/sensor-devices` | 原子替换全部设备 |
| DELETE | `/sensor-devices/delete` | 单个或批量删除设备 |
| GET | `/cluster-agents-catalog` | 读取发现目录及更新时间 |
| PUT | `/cluster-agents-catalog/agents/{node_id}` | 合并单个 Agent 快照 |
| PUT | `/cluster-agents-catalog` | 原子替换全部目录 |
| DELETE | `/cluster-agents-catalog/agents/delete` | 单个或批量删除目录节点 |
| GET | `/cluster-agent-registry` | 读取注册表及授权白名单 |
| PUT | `/cluster-agent-registry/agents/{node_id}` | 覆盖单个授权条目 |
| PUT | `/cluster-agent-registry` | 原子替换全部注册表 |
| DELETE | `/cluster-agent-registry/agents/delete` | 单个或批量删除授权条目 |

## 模型锚点

`anchor_id` 必须非空且不超过 64 字符，单条 PUT 的路径与 body 中 ID 必须一致。`instance_id` 至多 128 字符。其他字段原样存储，包括 `local_position`、`local_normal`、`title`、`device_id` 等。

```bash
curl -X PUT http://127.0.0.1:8080/api/ue/config/model-anchors/anchor_1 \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Token: <configured-token>" \
  -d '{"anchor_id":"anchor_1","instance_id":"hall_1","revision":0,"title":"探照灯","local_position":{"x":1,"y":2,"z":3},"local_normal":{"x":0,"y":0,"z":1}}'
```

| 场景 | 请求 revision | 结果 |
|---|---|---|
| 新建 | 缺失、null、0 或 1 | 保存为 1 |
| 新建 | 其他非负整数 | 409 |
| 更新 | 等于数据库当前版本 | 内容变化时版本加 1 |
| 更新 | 等于当前版本，内容相同 | 幂等返回，版本不变 |
| 更新 | 缺失、null 或版本过期 | 409 |
| 任意写入 | 非整数、负数、超过有符号 INT 范围 | 400 |

单条 PUT 返回保存后的记录。GET 和全量 PUT 返回 `{"version":1,"anchors":[...]}`。全量请求为 `{"anchors":[...]}`，逐条执行同样的版本校验，冲突时整个事务回滚。全量写入不是绕过版本规则的数据库导入接口。

## 设备、集群目录与注册表

设备单条请求至少包含与路径一致的 `device_id`，ID 非空且至多 128 字符。GET/全量 PUT 使用 `{"version":1,"devices":[...]}`；单条 PUT 返回保存后的记录。

```json
{"device_id":"light_1","display_name":"报告厅灯光","status":"fault","fault_message":"离线","vendor_extension":{"channel":3}}
```

两个集群域的节点 ID 非空且至多 128 字符。单条请求的路径决定 `nodeId`，覆盖 body 中的同名字段。全量请求均使用 `{"agents":{"node_1":{...}}}`，对象键决定各条记录的 `nodeId`。

- 目录单条 PUT 保留未提交字段，仅覆盖提交字段；全量 PUT 完全替换记录和节点集合。GET 返回 `{"version":1,"updatedAt":<最新数据库更新时间的秒时间戳>,"agents":{...}}`。全量 PUT 的响应包含 `version`、`agents`；请求中的 `updatedAt` 不写入数据库。
- 注册表单条 PUT 覆盖该条记录，`enabled` 为布尔值 true 时授权，缺失或 false 时不授权。GET/全量 PUT 返回 `{"version":1,"agents":{...},"authorizedWhitelist":[...]}`。单条 PUT 返回本次保存的 `agents` 条目和完整白名单。白名单按节点 ID 排序，由数据库中的启用状态派生，请求中的白名单不参与授权计算。

## 删除与全量替换

删除请求支持单个或批量 ID，两者同时提供时合并去重，不存在的 ID 忽略：

```json
{"anchor_id":"a1","anchor_ids":["a1","a2"]}
```

设备改用 `device_id` / `device_ids`，集群改用 `node_id` / `node_ids`。成功统一返回 `{"ok":true}`，没有有效 ID 则返回 400。

四类全量 PUT 都会删除请求集合中未出现的记录，并将整次修改放入事务。清空需要显式传入 `{"anchors":[]}`、`{"devices":[]}` 或 `{"agents":{}}`。遗漏集合字段、null、错误类型、重复 ID 或无效 ID 返回 400，避免把错误请求当作清空操作。数据库错误返回 500，锚点版本冲突返回 409。

## 既有数据迁移与切换

四张表保留源网关表名和列定义，便于在暂停源端写入后通过数据库备份/恢复迁移。建议先对当前目标库备份，再将源库的四张表以 data-only 方式导出到文件，经审核后导入当前库的空目标表；保留 `revision`、JSON payload、enabled 及更新时间。先用四个 GET 接口逐项回读核对，再切换 Server 地址和鉴权请求头。

源端锚点若已有 revision > 1，直接向空库调用全量 PUT 会返回 409。此类迁移应使用数据库数据导入以保留原始版本；普通新增仍遵守上述 revision 规则。当前实现未自动连接源数据库或搬迁现场数据。

## 验证

```powershell
go test ./...
go vet ./...
go build ./...
```

真实 MySQL 的 16 接口集成测试通过可选环境变量启用：

```powershell
$env:DCX_GATEWAY_TEST_MYSQL_DSN = 'root@tcp(127.0.0.1:<test-port>)/?parseTime=true'
go test ./internal/api -run TestAuxConfigMySQLHTTPIntegration -v -count=1
Remove-Item Env:DCX_GATEWAY_TEST_MYSQL_DSN
```

只针对可丢弃的测试 MySQL 使用此命令，账号需有创建数据库权限。测试自动创建并清理独立的 `dcx-gateway-aux-test-*` 数据库，验证 SQL 可重复执行、16 个接口、鉴权、字段透传、并发版本冲突、原子回滚、合并、删除及白名单派生。
