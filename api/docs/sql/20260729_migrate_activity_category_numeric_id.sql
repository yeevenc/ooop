-- 活动分类 ID：英文字符串主键迁移为数字自增主键。
-- 执行前请备份 activity_categories、activities，并在停写窗口执行。
-- 重要：activity_category_legacy_ids 是旧版 App 兼容表，不能在迁移完成后立即删除。
-- 只有确认旧版 App 已停止使用、请求日志不再出现英文分类 ID，且服务端兼容代码已下线后才能清理。

-- 1. 为现有分类按“排序、创建时间、旧 ID”生成稳定的数字 ID。
ALTER TABLE activity_categories
  ADD COLUMN numeric_id BIGINT NULL AFTER id;

SET @next_activity_category_id := 0;
UPDATE activity_categories
SET numeric_id = (@next_activity_category_id := @next_activity_category_id + 1)
ORDER BY sort ASC, created_at ASC, id ASC;

ALTER TABLE activity_categories
  MODIFY COLUMN numeric_id BIGINT NOT NULL;

-- 2. 保留兼容期旧 ID 映射；业务分类表不再保留英文标识。
-- 注意：该映射用于解析旧版 App 已缓存或仍提交的英文分类 ID。
CREATE TABLE activity_category_legacy_ids (
  legacy_id VARCHAR(32) NOT NULL,
  category_id BIGINT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (legacy_id),
  UNIQUE INDEX idx_activity_category_legacy_ids_category_id (category_id)
);

INSERT INTO activity_category_legacy_ids (legacy_id, category_id)
SELECT id, numeric_id
FROM activity_categories;

-- 3. 通过旧英文 ID 将已有活动关联迁移到数字 ID。
ALTER TABLE activities
  ADD COLUMN category_id_numeric BIGINT NULL AFTER category_id;

UPDATE activities AS activity
INNER JOIN activity_categories AS category
  ON category.id = activity.category_id
SET activity.category_id_numeric = category.numeric_id;

-- 若存在无法关联的历史活动，此处会因 NULL 数据终止，请先补齐分类后重试。
ALTER TABLE activities
  MODIFY COLUMN category_id_numeric BIGINT NOT NULL;

ALTER TABLE activities
  DROP COLUMN category_id;

ALTER TABLE activities
  CHANGE COLUMN category_id_numeric category_id BIGINT NOT NULL,
  ADD INDEX idx_activities_category_id (category_id);

-- 4. 删除旧英文主键，将已生成的数字 ID 设为自增主键。
ALTER TABLE activity_categories
  DROP PRIMARY KEY,
  DROP COLUMN id,
  CHANGE COLUMN numeric_id id BIGINT NOT NULL AUTO_INCREMENT,
  ADD PRIMARY KEY (id);
