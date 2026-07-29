# 活动分类数字 ID 生产迁移手册

> **长期维护提醒：** 当前存在旧版 App 兼容层。不能在数字 ID 上线后立即删除
> `activity_category_legacy_ids`、`ResolveCategoryID` 的英文 ID 解析或旧版字符串响应转换。

## 一、迁移目标

- `activity_categories.id`：由英文字符串主键改为 `BIGINT AUTO_INCREMENT`。
- `activities.category_id`：由英文字符串改为 `BIGINT`。
- 分类业务表不再保存英文标识，分类名称继续使用 `label`。
- 旧英文 ID 仅保存在 `activity_category_legacy_ids`，用于兼容旧版 App。
- App、后台和服务端统一使用数字分类 ID。

## 二、备份数据分析

分析文件：`/Users/vencyyee/Downloads/ooop.sql`

- MySQL 版本：`8.0.45`。
- `activity_categories`：11 条分类，`id` 类型为 `VARCHAR(32)`。
- `activities`：13 条活动，`category_id` 类型为 `VARCHAR(32)`。
- 所有活动分类均能匹配到 `activity_categories`，未发现孤立数据。
- 两张表之间没有外键，`activities.category_id` 只有普通索引。
- 分类 `label` 没有重复。

按 `sort、created_at、旧 ID` 排序后，预期映射如下：

| 数字 ID | 旧英文 ID | label |
| ---: | --- | --- |
| 1 | `clock` | 打卡 |
| 2 | `outdoor` | 户外 |
| 3 | `games` | 游戏 |
| 4 | `food` | 美食 |
| 5 | `sports` | 运动 |
| 6 | `music` | 音乐 |
| 7 | `photo` | 摄影 |
| 8 | `art` | 艺术 |
| 9 | `hiking` | 登山 |
| 10 | `citywalk` | 城市漫步 |
| 11 | `movie` | 电影 |

该映射仅是本次备份的预期结果。生产执行时以迁移脚本最终输出为准。

## 三、迁移原则

1. 不能先运行 GORM 自动迁移。
2. 不能直接执行 `ALTER TABLE activity_categories MODIFY id BIGINT`。
3. 必须通过映射表将英文 ID 转成数字 ID。
4. 必须先增加临时数字列并完成数据校验，再删除旧英文字段。
5. MySQL DDL 会自动提交，脚本中任何断言失败后都必须立即停止。
6. 迁移 SQL、GORM 迁移和新版服务发布必须在同一个停写窗口完成。
7. 本次不增加活动分类外键，避免扩大迁移风险。

## 四、相关文件

- 可执行迁移 SQL：
  `docs/sql/20260729_migrate_activity_category_numeric_id.sql`
- 兼容层清理 SQL：
  `docs/sql/20260729_remove_activity_category_legacy_compat.sql`
- GORM 迁移保护：
  `internal/database/mysql.go`

迁移 SQL 是唯一可执行版本，本手册不重复维护完整 SQL，避免后续两份内容不一致。

## 五、执行前准备

必须同时满足以下条件：

- 已完整备份生产数据库，并确认备份文件可以读取。
- 已暂停活动发布、活动编辑和后台分类管理。
- 旧服务保持运行或进入维护状态，但不能启动使用数字 ID 的新版服务。
- 已准备新版服务端和后台生产构建产物。
- 已确认当前数据库为 MySQL 8.0。
- 使用支持 `DELIMITER` 的 MySQL 客户端，并按完整脚本执行，不要拆开存储过程逐行执行。
- 已确认没有其他迁移任务同时修改 `activity_categories` 或 `activities`。
- 已确认 `MYSQL_AUTO_MIGRATE=false`，避免服务启动时提前触发 GORM。

执行前记录：

```sql
SELECT COUNT(*) AS category_count
FROM activity_categories;

SELECT COUNT(*) AS activity_count
FROM activities;

SELECT COUNT(*) AS orphan_activity_count
FROM activities AS activity
LEFT JOIN activity_categories AS category
  ON category.id = activity.category_id
WHERE category.id IS NULL;
```

`orphan_activity_count` 必须为 `0`。

## 六、正式执行流程

### 1. 进入维护窗口

1. 暂停活动发布、编辑和分类管理入口。
2. 确认没有正在执行的活动写请求。
3. 再次确认数据库备份时间和文件大小。

### 2. 执行专用迁移 SQL

执行：

```text
docs/sql/20260729_migrate_activity_category_numeric_id.sql
```

脚本共五个阶段：

1. 校验旧字段类型、重复分类、孤立活动和部分迁移状态。
2. 使用 `ROW_NUMBER()` 生成稳定数字 ID，并建立兼容映射表。
3. 填充分类和活动临时数字列，删除旧字段前再次强制校验。
4. 将临时数字列切换为正式字段。
5. 校验最终字段类型、自动递增属性和活动分类关联。

执行要求：

- 必须完整查看执行日志。
- 遇到任何 `SQLSTATE 45000` 后立即停止。
- 不得跳过断言继续执行后续语句。
- 不得在失败后直接重复运行完整脚本。
- 失败后根据当前表结构决定清理临时列或恢复备份。

### 3. 核对迁移输出

脚本末尾会输出完整映射以及统计结果，必须满足：

```text
category_count = mapping_count
orphan_activity_count = 0
```

同时检查：

```sql
SHOW CREATE TABLE activity_categories;
SHOW CREATE TABLE activities;
SHOW CREATE TABLE activity_category_legacy_ids;
```

预期结果：

- `activity_categories.id` 为 `BIGINT NOT NULL AUTO_INCREMENT`。
- `activities.category_id` 为 `BIGINT NOT NULL`。
- `activity_categories.label` 存在唯一索引。
- `activities.category_id` 存在普通索引。
- `activity_category_legacy_ids.legacy_id` 为主键。
- `activity_category_legacy_ids.category_id` 为唯一索引。

### 4. 运行 GORM 迁移

专用 SQL成功后才能执行：

```bash
cd /www/wwwroot/ooop/api
go run ./cmd/migrate
```

如果仍提示检测到英文分类结构，禁止绕过保护，应重新检查专用 SQL 的执行结果。

### 5. 发布服务端

1. 替换生产 API 二进制或按现有方式重新构建。
2. 确认生产环境 `MYSQL_AUTO_MIGRATE=false`。
3. 重启 API 服务。
4. 检查启动日志中没有数据库迁移、字段扫描或分类初始化错误。
5. 验证健康检查接口。

### 6. 发布后台

1. 上传后台 `dist` 生产产物。
2. 验证分类列表能够显示数字 ID。
3. 验证新增分类由数据库返回自增 ID。
4. 验证分类编辑不再提交英文标识。

### 7. 接口验证

新版协议请求：

```http
X-App-Category-ID-Version: 2
X-App-Version: <当前版本>
X-App-Version-Code: <当前版本号>
X-App-Platform: harmony
```

检查：

- 分类列表中的 `id` 为 JSON 数字。
- 活动列表和详情中的 `categoryId` 为 JSON 数字。
- 发布、编辑和筛选可以提交数字 `category_id`。

旧版兼容请求不携带 `X-App-Category-ID-Version`，检查：

- 分类 ID 和活动分类 ID 返回数字字符串。
- 旧英文 ID 请求能够通过 `activity_category_legacy_ids` 转换。

### 8. 恢复写入

完成以下验证后才能恢复：

- 分类列表正常。
- 活动列表、详情和筛选正常。
- 新建和编辑活动正常。
- 后台新增、编辑分类正常。
- 老版本兼容请求正常。
- 服务端日志没有分类 ID 转换错误。

## 七、接口变化

- `activity_categories.id`：`string` 改为 `number`。
- `activities.category_id`：`string` 改为 `number`。
- App 发布、编辑和分类筛选参数 `category_id`：使用 JSON 数字或数字查询参数。
- 后台新增分类不提交 `id`，由数据库创建后返回。
- `category_label` 继续保留，不参与分类关联。

## 八、老版本兼容

新版 App 公共请求头：

- `X-App-Platform`
- `X-App-Version`
- `X-App-Version-Code`
- `X-App-Category-ID-Version: 2`

兼容规则：

- 携带分类协议版本 `2` 时，服务端返回数字 ID。
- 未携带分类协议版本时，服务端返回数字字符串。
- 发布、编辑和筛选兼容数字、数字字符串及临时英文旧 ID。
- 英文旧 ID 只保存在 `activity_category_legacy_ids`。

## 九、失败处理与回滚

### 删除旧字段前失败

此时旧 `id/category_id` 仍然存在，旧服务仍可使用。

处理顺序：

1. 立即停止继续执行 SQL。
2. 保存完整错误信息。
3. 检查是否已生成映射表、`numeric_id` 或 `category_id_numeric`。
4. 不要直接重复运行完整脚本。
5. 修正数据后重新制定续跑步骤，或从备份恢复。

### 删除旧字段后失败

不要尝试让旧服务继续写入数字结构，直接进入维护状态。

处理顺序：

1. 停止 API 服务和所有活动写入。
2. 使用已验证的备份恢复 `activity_categories` 和 `activities`。
3. 恢复与旧结构匹配的服务端版本。
4. 校验活动分类关联后再恢复写入。

## 十、兼容层清理

必须同时满足以下条件：

1. 已通过版本统计确认老版本 App 停止使用，或已完成强制升级。
2. 服务端请求日志经过观察期后，不再出现英文分类 ID。
3. 新版 App 均携带 `X-App-Category-ID-Version: 2`。
4. 已备份 `activity_category_legacy_ids`。
5. 已删除并发布服务端英文 ID 解析和旧版字符串响应转换。

验证无异常后，才能执行：

```text
docs/sql/20260729_remove_activity_category_legacy_compat.sql
```

禁止把兼容层清理脚本与本次数字 ID 迁移放在同一发布批次。
