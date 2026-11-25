# dcx-gateway

基于 Go、Gin、GORM、MySQL 和 Redis 的数据网关，提供 UE 场景配置管理、业务数据组装、缓存预热，以及到 FastAPI 的 WebSocket 推送。

本仓库用于项目展示与本地演示。种子配置中的场景、路径、模型及坐标已替换为虚构样例；UE 模型资产、现场数据库导出、部署凭据和运行环境配置不随仓库分发。

## 主要功能

- 配置管理：启动配置、图表、模型类型、模型实例、摆放、路径、渲染及汇报脚本。
- 补充配置：模型锚点、传感器设备、Agent 发现目录和授权注册表，共 16 个接口。
- 一致性处理：锚点 revision 乐观锁、幂等写入、批量替换事务与失败回滚。
- 数据组装：通过 Adapter 对接安防、能耗和教学数据，默认使用 Mock 数据。
- 缓存及预热：Redis 热数据、脚本预热与定时刷新。
- 实时推送：WebSocket 长连接、心跳、断线重连和运行状态查询。
- 鉴权与可观测性：Bearer Session、可选静态 token、请求 ID、结构化日志和深度健康检查。

## 工程结构

| 目录 | 职责 |
|---|---|
| `cmd/dcx-gateway` | 应用入口 |
| `internal/bootstrap` | 依赖装配与生命周期 |
| `internal/api` | Gin 路由、参数处理和 HTTP 响应 |
| `internal/service` | 业务编排、配置校验及数据组装 |
| `internal/repository` | MySQL 持久化与事务 |
| `internal/cache` | Redis 缓存与 Session |
| `internal/adapter` | 外部 HTTP 接口和 Mock 数据源 |
| `internal/model` | 实体、DTO、领域对象及协议 |
| `internal/websocket`、`internal/scheduler` | 实时推送与后台任务 |
| `migrations`、`docs` | 数据库结构、虚构演示种子及接口文档 |

## 五分钟演示

准备 Go 和已启动的 Docker，在仓库根目录执行：

```bash
go run ./cmd/dcx-gateway-demo
```

命令会创建临时 MySQL/Redis、导入演示种子，使用正式 bootstrap 与 Gin 路由启动 HTTP 服务，并完成“启动 → 修改模型并回读 → 预热写缓存并推送 → 查询连接与推送状态”。WebSocket 接收端为本地模拟 FastAPI，实际验证 `chart_data_push` 已被接收。

演示使用随机端口和临时凭据，不读取或覆盖 `config.dev.yaml`；完成后自动清理容器和临时配置。首次使用可能需要下载 Docker 镜像。演示步骤、讲解要点与预期输出见[面试演示](docs/testing/demo-walkthrough.md)。

## 本地运行

依赖 Go（版本见 `go.mod`）、MySQL 8.0+ 和 Redis。查看 API 与 Mock 数据不需要运行 UE 或 FastAPI。

1. 在本地创建空数据库 `dcx-gateway`，按文件名顺序执行 `migrations/001_*.sql` 至 `014_*.sql`。这些脚本面向新建演示库，不用于清理或迁移已有现场数据库。
2. 将 `configs/config.example.yaml` 复制为 `configs/config.dev.yaml`，填入自己的 MySQL/Redis 配置。
3. 运行 `go run ./cmd/dcx-gateway`，访问 `http://127.0.0.1:8080/api/health` 或 `/api/ue/config/bootstrap`。

数据库名包含连字符，在 SQL 中需要使用反引号：

```sql
CREATE DATABASE IF NOT EXISTS `dcx-gateway` CHARACTER SET utf8mb4;
```

模板默认使用 Mock Adapter，不连接 FastAPI WebSocket。模型 URI 与 nDisplay 文档是占位示例；实际 UE 展示需自行提供资产和集群配置。视频地址使用保留示例域名，不提供实际视频服务。

公开版本不创建默认登录账号。写接口默认受鉴权保护；可自行创建 bcrypt 用户，或在私有配置中设置随机 `security.static_token` 并启用 `static_token_enabled`，使用 `X-Gateway-Token` 请求头进行本地演示。不要把配置、真实 token 或数据库备份提交到仓库。

## 验证与公开前检查

```bash
go test ./...
go vet ./...
go build ./...
python scripts/check_public_release.py
python scripts/check_docs.py
```

扫描器会检查当前目录中的本地配置、内网地址、个人路径、密钥、密码 hash、数据库导出及压缩文件；只输出位置与类别，不输出匹配内容。它是补充检查，不能判断所有业务内容是否允许公开。

## 文档

- [文档首页与目录规范](docs/README.md)
- [架构概览](docs/architecture/overview.md)
- [UE Config API](docs/api/ue-config.md)
- [补充配置接口与迁移说明](docs/api/aux-config.md)
- [OpenAPI](docs/api/openapi.yaml)
- [部署说明](docs/operations/deployment.md)
- [公开版本清理说明](docs/operations/public-release.md)
- [联调验收清单](docs/testing/integration-checklist.md)
- [启动到实时推送的面试演示](docs/testing/demo-walkthrough.md)

阶段性分析与设计提案统一放在 `docs/archive/`；实际演示数据以当前 `migrations` 为准。
