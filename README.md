# OoopAdmin

当前目录是后台系统工作区，按项目职责拆分为两个独立子项目。

## 目录说明

```text
api/        Golang API 接口服务
web-admin/  Vue 后台管理项目
scripts/    本地启动与打包脚本
dist/       打包输出目录
```

## 一键启动

```bash
npm run dev
```

该命令会同时启动：

```text
API 服务：http://127.0.0.1:8080
后台管理：http://127.0.0.1:5173
```

停止时按 `Ctrl+C`。

首次启动前如需建表，先执行：

```bash
npm run api:migrate
```

## 独立启动

```bash
npm run api:dev
npm run web:dev
```

## 数据库迁移

```bash
npm run api:migrate
```

该命令只执行 API 项目的 GORM 数据表迁移。

## 独立打包

```bash
npm run build:api
npm run build:web
```

API 打包产物输出到：

```text
dist/api/
```

后台管理打包产物输出到：

```text
web-admin/dist/
```

## 全量打包

```bash
npm run build
```

该命令会依次构建 API 和后台管理项目，两个项目仍然保持独立产物。
