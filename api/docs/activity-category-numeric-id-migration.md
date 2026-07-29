# 活动分类数字 ID 迁移

> **长期维护提醒：** 当前存在旧版 App 兼容层。不能在数字 ID 上线后立即删除
> `activity_category_legacy_ids`、`ResolveCategoryID` 的英文 ID 解析或旧版字符串响应转换。

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
3. 将旧英文 ID 写入临时兼容映射表。
4. 确认没有未关联活动后删除分类业务表中的旧英文 ID。
5. 将数字 ID 设置为自增主键。

## 老版本兼容

- 新版 App 公共请求头携带：
  - `X-App-Platform`
  - `X-App-Version`
  - `X-App-Version-Code`
  - `X-App-Category-ID-Version: 2`
- 服务端根据分类协议版本返回数字 ID；App 版本头可用于统计老版本使用情况。
- 未携带该请求头的老版本，接口返回数字字符串，保持原有字符串校验可用。
- 发布、编辑和筛选接口兼容数字、数字字符串及临时英文旧 ID。
- 英文旧 ID 仅保存在 `activity_category_legacy_ids` 迁移映射表，不再进入分类业务表。

### 兼容层清理条件

必须同时满足以下条件，才能开始清理：

1. 已通过版本统计确认老版本 App 停止使用，或已完成强制升级。
2. 服务端请求日志经过观察期后，不再出现英文分类 ID。
3. 新版 App 均携带 `X-App-Category-ID-Version: 2`。
4. 已备份 `activity_category_legacy_ids`。

清理时先删除并发布服务端英文 ID 解析和旧版字符串响应转换，验证无异常后，再执行
`docs/sql/20260729_remove_activity_category_legacy_compat.sql`。禁止把清理脚本与迁移脚本放在同一发布批次执行。


## 发布顺序

1. 备份 `activity_categories` 和 `activities`。
2. 暂停活动发布、编辑及分类管理。
3. 执行迁移 SQL。
4. 部署服务端和后台管理。
5. 发布使用数字分类 ID 的 App 版本。
6. 验证分类列表、活动筛选、发布和编辑后恢复写入。

服务端与旧 App 的分类参数类型不兼容，应在同一发布窗口完成切换。
