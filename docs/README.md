# dcx-gateway 文档

本文档是工程说明的统一入口。首次阅读建议按“架构概览 → 接口说明 → 本地运行与部署 → 联调验收”的顺序查看；项目启动步骤见[项目 README](../readme.md)。

## 目录分类

```text
docs/
├── README.md                 文档入口、阅读顺序与维护规范
├── api/                      接口参考、OpenAPI 与离线 Swagger UI
├── architecture/             系统边界、分层与数据流转
├── operations/               部署、数据库检查及公开发布
├── testing/                  联调与验收操作
└── archive/                  历史分析和设计提案
```

## 按任务查阅

| 目标 | 文档 | 说明 |
|---|---|---|
| 理解工程 | [架构概览](architecture/overview.md) | 组件、职责、请求链路与启动过程 |
| 理解结构体与协议 | [数据流转与结构体契约](architecture/data-flow.md) | 中文主版本；[英文版本](architecture/data-flow.en.md)作为补充 |
| 使用 UE 配置接口 | [UE 配置接口](api/ue-config.md) | 配置读取、写入、模型视图及鉴权 |
| 操作模型配置 | [模型配置接口使用指南](api/model-config.md) | 模型类型、实例、placement 的使用示例 |
| 使用补充配置接口 | [补充配置接口](api/aux-config.md) | 锚点、设备、Agent 目录与授权注册表 |
| 查看完整接口结构 | [OpenAPI](api/openapi.yaml) | 机器可读契约；[Swagger UI 入口](api/swagger-ui/index.html) |
| 构建与部署 | [部署说明](operations/deployment.md) | 配置、ARM64 打包、systemd 和文档服务 |
| 检查数据库 | [只读 SQL 检查清单](operations/database-checks.sql) | 使用目标库执行，只包含查询语句 |
| 准备公开展示 | [公开版本清理与发布检查](operations/public-release.md) | 演示数据边界、凭据处理及公开前检查 |
| 联调与验收 | [联调验收清单](testing/integration-checklist.md) | 功能验证、运行状态与预热操作 |
| 面试现场演示 | [五分钟演示](testing/demo-walkthrough.md) | 独立环境启动、模型修改、预热与实际推送验证 |
| 查阅历史决策 | [历史归档索引](archive/README.md) | 保留原阶段分析，不作为当前部署依据 |

当前接口行为以源码和 `api/openapi.yaml` 为依据；示例数据以根目录 `migrations/` 为依据。操作示例中的模型 ID、账号、主机和端口需与自己的演示环境一致。

## 文档状态

- `api/`、`operations/`、`testing/` 为使用和运维资料，接口或部署流程变化时同步更新。
- `architecture/` 说明模块边界与数据流转，详细协议仍以接口文档和源码为准。
- `archive/` 保存历史分析、阶段方案和迁移提案。归档内容可能包含旧版本判断，不代表当前已实现或待实现功能。
- Swagger UI 的第三方 JS/CSS 文件保留原名及版权信息，不按工程文档命名规则改写。

## 命名与引用规范

1. 自建目录和文件采用小写英文 `kebab-case`，例如 `model-config.md`、`database-checks.sql`。导航入口统一使用 `README.md`。
2. 中文为默认语言；同一文档的英文补充使用 `.en.md` 后缀，不并列维护含义不明的重复文件。
3. 当前文档不在文件名中追加版本号。版本快照只放入 `archive/`，必要时使用 `-v12` 等后缀。
4. 每份 Markdown 只有一个一级标题，标题描述内容用途；章节使用二级或三级标题。
5. Markdown 链接相对于当前文件所在目录。代码、配置和脚本示例中的 `internal/`、`configs/`、`migrations/` 等路径均相对于仓库根目录。
6. 移动文档时同步更新 README、脚本、打包路径、静态资源路径和相关链接。上游未分发的历史资料用普通文字注明，不保留失效链接。
7. 文档仅保存脱敏示例与可维护源文件，不加入现场配置、数据库导出、PDF 或压缩包。

## 本地查看与维护检查

在仓库根目录执行：

```bash
bash scripts/serve_openapi_docs.sh
```

服务从 `docs/api/` 提供接口规范及静态资源，默认访问地址保持为 `http://127.0.0.1:8090/openapi.yaml` 和 `http://127.0.0.1:8090/swagger-ui/`。Markdown 文档可直接在编辑器或代码托管平台阅读。

修改后执行：

```bash
python scripts/check_docs.py
python scripts/check_public_release.py
```

文档检查覆盖命名、一级标题、本地 Markdown 链接及锚点、Swagger UI 静态资源引用。发布包保留完整 `docs/` 目录结构，避免跨目录链接丢失。
