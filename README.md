# OPIMS — Oversea Project Integrated Management System

海外项目综合管理系统，面向海外运营中心项目管理部，提供项目全生命周期信息的标准化管理、数据联动与可视化。

## 技术栈

| 层 | 选型 |
|---|---|
| 后端 | Go |
| 前端 | Vue 3 + Element Plus |
| 数据库 | SQLite（pure-Go） |
| 打包 | 单个 opims.exe（~20MB） |

## 启动

```bash
# 方式一：双击运行
opims.exe

# 方式二：命令行
opims.exe

# 浏览器访问
http://localhost:8080
```

> 重复双击不会重复启动，仅打开浏览器。
> 在命令行窗口按 **Enter** 停止服务。

## 目录结构

```
opims/
├── backend/              # Go 后端源码
│   ├── main.go           # 入口，路由注册
│   ├── handlers/         # HTTP handler
│   ├── database/         # SQLite 数据库初始化与 schema
│   ├── models/           # 数据模型
│   └── services/         # Excel 解析、文件监控
├── frontend/             # Vue 3 前端源码
│   └── src/
│       ├── views/        # 页面组件
│       ├── i18n/         # 中英双语
│       └── router/       # 路由
├── frontend-dist/        # 前端构建产物（与 exe 同目录）
├── docs/                 # 文档
│   ├── OPIMS_系统功能介绍.md
│   └── OPIMS_操作说明.md
└── .gitignore
```

## 功能板块

| 板块 | 状态 |
|---|---|
| 首页看板 | ✅ 已实现 |
| 项目清单 | ✅ 已实现 |
| 项目文件 | ✅ 已实现 |
| 分包商黑名单 | ✅ 已实现 |
| 人员黑名单 | 📅 待开发 |
| 项目进度 | 📅 待开发 |
| 项目质量 | 📅 待开发 |
| 项目分包 | 📅 待开发 |
| 项目人员 | 📅 待开发 |

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
- 项目简称和合同编号从 `海外项目简称.xlsx` 按项目名称自动匹配
- 冲突处理：跳过 / 覆盖 / 保留两者

### 备份恢复

- 备份：`opims_backup_YYYYMMDD_HHmm.db`
- 恢复：选择备份文件 → 确认覆盖 → 替换当前数据库

## 开发

```bash
# 后端
cd backend
go run .

# 前端
cd frontend
npm install
npm run dev

# 构建
cd frontend && npm run build    # 产物到 ../backend/frontend-dist/
cd ../backend && go build -o opims.exe .
```

## 许可

内部使用。
