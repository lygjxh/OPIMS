# OPIMS — Oversea Project Integrated Management System

海外项目综合管理系统，面向海外运营中心项目管理部，提供项目全生命周期信息的标准化管理、数据联动与可视化。

## 技术栈

| 层 | 选型 |
|---|---|
| 后端 | Go（标准库 net/http，无 Web 框架） |
| 前端 | Vue 3 + TypeScript + Element Plus + Vite |
| 数据库 | SQLite（modernc.org/sqlite，pure-Go 免 CGO） |
| 地图 | Leaflet |
| 打包 | 单个 opims.exe（~20MB，内含前端产物） |

## 启动

```bash
# 双击或命令行运行
opims.exe

# 浏览器访问
http://localhost:8080
```

> 重复双击不会重复启动，仅打开浏览器（通过检测 8080 端口占用实现）。
> 在命令行窗口按 **Enter** 停止服务。

**首次运行**：文件根目录默认为 exe 所在目录，需在「项目文件 → 设置根目录」中指向实际的项目文件目录。该配置存入数据库，换电脑或移动文件夹后重新设置即可。

## 目录结构

```
opims/
├── backend/                    # Go 后端源码
│   ├── main.go                 # 入口：端口检测、DB 初始化、路由注册、启动浏览器
│   ├── handlers/
│   │   ├── helpers.go          # Handler 结构、scanProject、列名/占位符、GPS 解析
│   │   ├── projects.go         # 项目 CRUD、Excel 导入导出
│   │   ├── files.go            # 文件树、合同文件识别、打开文件、根目录设置
│   │   ├── blacklist.go        # 分包商黑名单
│   │   └── dashboard.go        # 看板聚合、数据库备份/恢复
│   ├── database/db.go          # SQLite 初始化与 schema（migrate）
│   ├── models/project.go       # 数据模型（Project 82 字段宽表）
│   ├── services/
│   │   ├── excel.go            # Excel 解析（按 sheet 名推断状态 + 列号映射）
│   │   ├── country.go          # 国别关键词表（有序，见下方「注意事项」）
│   │   ├── country_test.go     # 国别识别测试
│   │   └── rootdir.go          # 线程安全保存「文件根目录」配置
│   └── frontend-dist/          # 前端构建产物（gitignore，由 npm run build 生成）
├── frontend/                   # Vue 3 前端源码（唯一前端目录）
│   └── src/
│       ├── App.vue             # 外壳：侧边栏导航 + 顶栏
│       ├── theme.css           # 设计令牌 + Element Plus 主题覆盖
│       ├── data/countryCoords.ts  # 国别→经纬度表（地图近似定位用）
│       ├── views/              # 页面组件
│       ├── i18n/               # 中英双语
│       └── router/             # 路由（hash 模式）
├── docs/                       # 功能介绍与操作说明
└── .gitignore
```

## 功能板块

| 板块 | 状态 |
|---|---|
| 首页看板 | ✅ 已实现（KPI、全球分布地图、合同额分布、快捷入口） |
| 项目清单 | ✅ 已实现（筛选、排序、分页、Excel 导入导出、详情） |
| 项目文件 | ✅ 已实现 |
| 分包商黑名单 | ✅ 已实现 |
| 人员黑名单 | 📅 待开发 |
| 项目进度 | 📅 待开发 |
| 项目质量 | 📅 待开发 |
| 项目分包 | 📅 待开发 |
| 项目人员 | 📅 待开发 |

待开发模块统一指向 `views/Placeholder.vue`，侧边栏标记「待建」，点击给出提示。

## API 一览

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/projects` | 列表。**不带 status 参数时只返回「在建+未开工」**；`status=all` 返回全部；也支持 `type`/`country`/`domestic_overseas`/`keyword` |
| POST/PUT | `/api/projects` | 新建 / 更新 |
| GET/PUT/DELETE | `/api/projects/{id}` | 单条查询 / 更新 / 软删除（`is_deleted=1`） |
| POST | `/api/projects/import` | Excel 导入（`conflict=skip\|overwrite\|keep_both`） |
| GET | `/api/projects/export` | 导出 xlsx |
| GET | `/api/dashboard` | 看板聚合（总数、各状态计数、有 GPS 的地图标记） |
| GET | `/api/files/scan` | 扫描根目录 |
| GET/POST | `/api/files/root` | 读取 / 设置文件根目录 |
| GET | `/api/files/contracts` | 某项目的合同文件列表 |
| GET | `/api/files/view` | 用系统默认程序打开文件 |
| GET | `/api/files/{project}` | 某项目的文件树 |
| GET/POST | `/api/blacklist/subcontractor` | 分包商黑名单列表 / 新增 |
| PUT/DELETE | `/api/blacklist/subcontractor/{id}` | 修改 / 删除（「已拉出」记录不可改删） |
| POST | `/api/config/backup` `/api/config/restore` | 数据库备份 / 恢复 |

## 关键设计约定

### 项目关联

整个系统以**项目简称**为关联键。文件夹名必须与项目简称一致，否则文件无法关联到对应项目。

### 合同文件命名

`01.Contract` 文件夹内的文件按文件名自动分类：

| 文件名规则 | 显示标签 |
|---|---|
| 含 `contract`（不含 `supplement`） | 主合同 |
| 含 `supplement` | 补充合同 |
| 含 `supplement` + 末尾数字 | 补充合同一/二/三… |
| 以上均不匹配 | 合同文件 |

### 数据导入

- 仅导入**境外**项目（境内/境外=境外）
- 项目简称和合同编号从 `海外项目简称.xlsx` 按项目名称自动匹配（在文件根目录下查找）
- 冲突处理：跳过 / 覆盖 / 保留两者
- Excel 解析按 **sheet 名**推断项目状态（含"未开工"/"停工"/"完工"/"在建"），列位置在 `services/excel.go` 中按列号硬编码，表格列序变化会导致解析错位

### 国别识别（services/country.go）

导入时按 `countryRules` **有序**匹配项目名称+地址中的关键词。

⚠️ **顺序敏感，改动时务必注意**：
- 长国名必须排在其包含的短国名**之前**，否则会被抢先匹配
  （`印度尼西亚` 必须在 `印度` 前；`白俄罗斯` 必须在 `俄罗斯` 前）
- 项目专有缩写（ADNOC/DBN/PLF）排在所有国名**之后**，避免抢先命中
- **不要**把工艺缩写（如 `LNG`）当地名 —— 它会错误命中任何含 LNG 的项目
- 改完请运行 `go test ./services/` 验证（`country_test.go` 已覆盖上述场景）

### 地图定位

- 有 GPS 坐标的项目按真实坐标显示（实线标记）
- 无 GPS 但国别可识别的项目，落到 `frontend/src/data/countryCoords.ts` 中该国代表坐标，**虚线标记**表示近似，同国多项目按黄金角散开
- 完全无可定位项目时显示空状态提示
- 状态用**颜色 + 形状**双重编码（圆点/空心圈/方块/小圆点），因为红绿在红绿色盲下无法靠色相区分；tooltip 始终带状态文字

### 备份恢复

- 备份：`opims_backup_YYYYMMDD_HHmm.db`
- 恢复：选择备份文件 → 确认覆盖 → 替换当前数据库

## 开发

```bash
# 后端
cd backend
go run .
go vet ./...
go test ./...

# 前端（开发模式，代理 /api 到 8080）
cd frontend
npm install
npm run dev

# 构建（前端产物 → backend/frontend-dist/，再编进 exe 同目录）
cd frontend && npm run build
cd ../backend && go build -o opims.exe .
```

## 注意事项（易踩的坑）

1. **前端只有一个目录 `frontend/`**。历史上曾短暂存在 `frontend-v2/`，两者构建到同一个 `frontend-dist/` 会互相覆盖，已合并删除。不要再新建平行的前端目录。
2. **国别是导入时写入数据库的**。修改 `country.go` 的关键词表后，已入库的旧数据不会自动更新，需要**重新导入 Excel** 才会生效。
3. **`/api/projects` 默认只返回「在建+未开工」**。需要全量时必须显式传 `status=all`，否则统计会少数据。
4. **`Project` 是 82 字段宽表**。新增字段需同步改动：`database/db.go` 的 schema、`models/project.go`、`handlers/helpers.go` 的 `projectsInsertCols`/`projectsInsertVals`/`placeholders(82)` 这个数字、以及 `services/excel.go` 的各 `parseXXX` 列映射。**极易漏改**，改完务必实际跑一遍导入和新建。
5. **Excel 列映射是硬编码列号**。上游表格列序调整会导致数据错位，且不会报错。
6. **备份前会执行 `PRAGMA wal_checkpoint(TRUNCATE)`** 以避免 WAL 中未落盘的数据丢失。

## 许可

内部使用。
