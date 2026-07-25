# 变更记录

本文件记录相对 `v0.1`（master 初始版本）的改动，供后续开发者/AI 模型接手时参考。
**重点看每条的「为什么」和「注意」**，避免改回去或踩同样的坑。

---

## for-claude 分支（待合并进 main）

### 一、代码清理（不改变功能）

| 改动 | 为什么 |
|---|---|
| 合并 `scanProject` / `scanProjectFromRows` 两个近乎相同的 60 行函数为一个 `scanProject(rowScanner)` | 原本两份代码逐字重复，改字段要改两处、极易漏改。`*sql.Row` 和 `*sql.Rows` 都实现了 `Scan(dest ...any) error`，用一个只含 Scan 方法的接口即可统一 —— 这是 Go 的标准做法，**不是"语言限制导致无法合并"** |
| 删除 `services/filewatch.go`，新增 `services/rootdir.go` | 原 FileWatcher 从未接线使用：`onChange` 回调始终为 nil，且 `addRecursive` 名不副实（只监控顶层一层）。它实际承担的唯一职责是「线程安全地保存文件根目录」，故按真实职责改名为 `RootDir`，并从 `go.mod` 移除 `fsnotify` 依赖 |
| 移除误提交的 `opims.exe~` / `stderr.log` / `stdout.log`，`.gitignore` 补 `*.log`、`*~`、`node_modules/` | 加 `.gitignore` 不会移除已跟踪的文件，必须配合 `git rm --cached` |

**注意**：若将来需要「文件变动自动刷新前端」，不要简单恢复 FileWatcher —— 递归监控云盘大目录会在启动时挂载数千个 watch，拖慢启动并可能触及系统上限。应配合 SSE/WebSocket 单独设计，并限制监控深度、去掉逐文件日志。

### 二、Bug 修复

#### 1. 国别判定结果会随机跳变（严重）

- **原因**：`extractCountry` 用 `map[string]string` 存关键词表，而 **Go 的 map 遍历顺序是随机的**。项目名若同时命中多个关键词，返回值会在多次运行间变化。
- **修复**：改为有序切片 `countryRules`（`services/country.go`），按「长国名 → 短国名 → 项目专有缩写」排序；补充越南等 40+ 国别；新增 `country_test.go` 锁定顺序敏感场景。
- **同时移除**：`"LNG" → 阿联酋` 这条规则。LNG 是工艺缩写不是地名，会错误命中任何含 LNG 的项目。
- **注意**：改关键词表后必须跑 `go test ./services/`；且**已入库数据不会自动更新**，需重新导入 Excel。

#### 2. `/api/projects` 无法取全量数据

- **原因**：不带 `status` 参数时后端强制附加 `AND project_status IN ('在建','未开工')`，首页做全量统计时拿不到停工/完工项目。
- **修复**：支持 `status=all` 显式请求全部状态，跳过该默认过滤。原有默认行为保持不变（兼容既有前端调用）。

#### 3. 备份可能丢失最新数据

- **原因**：SQLite 处于 WAL 模式，备份只复制 `.db` 主文件，未落盘的 WAL 数据会丢。
- **修复**：`Backup` 复制前执行 `PRAGMA wal_checkpoint(TRUNCATE)`。

### 三、前端重新设计

**原 `frontend/` 已被新设计替换，`frontend-v2/` 已合并删除。**

- ⚠️ **不要再建平行的前端目录**。曾经 `frontend/` 与 `frontend-v2/` 并存，两者 `vite.config.ts` 都输出到 `../backend/frontend-dist`，谁后构建谁生效 —— 会静默覆盖，无任何报错。
- 旧版 UI 如需查阅：`git show origin/master:frontend/src/App.vue`

视觉方案（Data-Dense Dashboard 风格）：

| 项 | 值 |
|---|---|
| 主色 | `#1E40AF` 深蓝 |
| 强调色 | `#D97706` 琥珀 |
| 字体 | 系统字体栈，**无外部字体依赖**（离线/内网可用，也利于后续鸿蒙移植） |
| 设计令牌 | `frontend/src/theme.css`，同时覆盖 Element Plus 的 CSS 变量 |

主要页面改动：

- **App.vue**：品牌区、分组导航（主业务/扩展模块）、待建模块标记且可点击提示、顶栏面包屑+用户信息
- **Dashboard.vue**：KPI 卡片（可点击跳转并按状态筛选、显示占比、0 值弱化）、全球分布地图（国别近似定位 + 空状态）、合同额按国别条形图
- **ProjectList.vue**：合同额大额转「亿」小额留「万」且可排序、状态列三重编码、统计条与可清除筛选标签、分页、操作列单行紧凑布局
- **Placeholder.vue**：友好的「模块建设中」空状态

#### 状态配色的重要约束

状态色取自 dataviz 规范已验证的 status 配色：
`在建 #2a78d6` / `未开工 #eda100` / `停工 #d03b3b` / `完工 #0ca30c`

- 正常视力下四色两两可辨（最差 ΔE 24.1，通过阈值）
- **但红/绿在红绿色盲下无法靠色相区分（ΔE 仅 4.1，物理限制）**
- 因此**凡使用状态色处必须同时给出文字标签或形状**：KPI 卡带图标+文字、地图标记用四种不同形状且 tooltip 带状态文字、表格状态列带形状+文字、图例带文字
- `未开工 #eda100` 在白底对比度仅 2.11（<3:1），**只可用于色块/描边，不可用于文字**

改配色时请重新校验，不要凭感觉选色。

---

## v0.1（master 初始版本）

首个可用版本：项目清单、项目文件、分包商黑名单、首页看板四个模块，Excel 导入导出，数据库备份恢复。
