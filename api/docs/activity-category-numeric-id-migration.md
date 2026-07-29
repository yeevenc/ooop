# 活动分类数字 ID 迁移

## 目标

活动分类只保留数据库自增数字 `id`，不再维护英文标识。分类展示名称继续使用现有 `label`。

## 接口变化

- `activity_categories.id`：`string` 改为 `number`
- `activities.category_id`：`string` 改为 `number`
- App 发布、编辑、分类筛选参数 `category_id`：改为 JSON 数字或数字查询参数
- 后台新增分类：不再提交 `id`，由数据库创建后返回

## 历史数据

执行 `docs/sql/20260729_migrate_activity_category_numeric_id.sql`：

1. 为现有分类生成数字 ID。
2. 根据旧英文 ID 更新所有历史活动的分类关联。
3. 确认没有未关联活动后删除旧英文 ID。
4. 将数字 ID 设置为自增主键。

## 发布顺序

1. 备份 `activity_categories` 和 `activities`。
2. 暂停活动发布、编辑及分类管理。
3. 执行迁移 SQL。
4. 部署服务端和后台管理。
5. 发布使用数字分类 ID 的 App 版本。
6. 验证分类列表、活动筛选、发布和编辑后恢复写入。

服务端与旧 App 的分类参数类型不兼容，应在同一发布窗口完成切换。
