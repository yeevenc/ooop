-- 活动分类 ID：英文字符串主键迁移为数字自增主键。
-- 适用数据库：MySQL 8.0。
-- 执行要求：
-- 1. 已完整备份 activity_categories、activities；
-- 2. 已暂停活动发布、编辑和分类管理；
-- 3. 必须先执行本脚本，再运行 go run ./cmd/migrate；
-- 4. 禁止让 GORM 直接转换英文 ID 字段；
-- 5. activity_category_legacy_ids 是旧版 App 兼容表，兼容期结束前禁止删除。
--
-- MySQL 的 DDL 会自动提交，本脚本通过“先填充、先校验、后切换”降低半迁移风险。
-- 任一断言失败后立即停止，不要跳过错误继续执行，也不要直接重复运行本脚本。

-- ============================================================
-- 阶段一：迁移前断言
-- ============================================================

DROP PROCEDURE IF EXISTS assert_activity_category_numeric_precheck;

DELIMITER $$

CREATE PROCEDURE assert_activity_category_numeric_precheck()
BEGIN
  DECLARE category_count BIGINT DEFAULT 0;
  DECLARE orphan_activity_count BIGINT DEFAULT 0;
  DECLARE duplicate_label_count BIGINT DEFAULT 0;
  DECLARE category_id_type VARCHAR(64) DEFAULT '';
  DECLARE activity_category_id_type VARCHAR(64) DEFAULT '';
  DECLARE legacy_mapping_table_count BIGINT DEFAULT 0;
  DECLARE temporary_column_count BIGINT DEFAULT 0;

  SELECT COALESCE(MAX(DATA_TYPE), '')
  INTO category_id_type
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'activity_categories'
    AND COLUMN_NAME = 'id';

  SELECT COALESCE(MAX(DATA_TYPE), '')
  INTO activity_category_id_type
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'activities'
    AND COLUMN_NAME = 'category_id';

  SELECT COUNT(*)
  INTO legacy_mapping_table_count
  FROM information_schema.TABLES
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'activity_category_legacy_ids';

  SELECT COUNT(*)
  INTO temporary_column_count
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND (
      (TABLE_NAME = 'activity_categories' AND COLUMN_NAME = 'numeric_id')
      OR
      (TABLE_NAME = 'activities' AND COLUMN_NAME = 'category_id_numeric')
    );

  IF category_id_type NOT IN ('char', 'varchar') THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：activity_categories.id 不是旧版字符串类型';
  END IF;

  IF activity_category_id_type NOT IN ('char', 'varchar') THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：activities.category_id 不是旧版字符串类型';
  END IF;

  IF legacy_mapping_table_count > 0 OR temporary_column_count > 0 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：检测到映射表或临时数字列，请先确认是否存在未完成迁移';
  END IF;

  SELECT COUNT(*)
  INTO category_count
  FROM activity_categories;

  IF category_count = 0 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：activity_categories 没有分类数据';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM activity_categories
    WHERE TRIM(id) = ''
  ) THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：存在空的英文分类 ID';
  END IF;

  SELECT COUNT(*)
  INTO duplicate_label_count
  FROM (
    SELECT label
    FROM activity_categories
    GROUP BY label
    HAVING COUNT(*) > 1
  ) AS duplicate_labels;

  IF duplicate_label_count > 0 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：存在重复分类 label';
  END IF;

  SELECT COUNT(*)
  INTO orphan_activity_count
  FROM activities AS activity
  LEFT JOIN activity_categories AS category
    ON category.id = activity.category_id
  WHERE category.id IS NULL;

  IF orphan_activity_count > 0 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：存在无法匹配分类的历史活动';
  END IF;
END$$

DELIMITER ;

CALL assert_activity_category_numeric_precheck();
DROP PROCEDURE assert_activity_category_numeric_precheck;

-- ============================================================
-- 阶段二：建立稳定映射并填充临时数字列
-- ============================================================

CREATE TEMPORARY TABLE temporary_activity_category_id_map (
  legacy_id VARCHAR(32) NOT NULL,
  category_id BIGINT NOT NULL,
  PRIMARY KEY (legacy_id),
  UNIQUE INDEX idx_temporary_activity_category_id (category_id)
) ENGINE = InnoDB;

INSERT INTO temporary_activity_category_id_map (legacy_id, category_id)
SELECT
  id,
  ROW_NUMBER() OVER (
    ORDER BY sort ASC, created_at ASC, id ASC
  ) AS category_id
FROM activity_categories;

CREATE TABLE activity_category_legacy_ids (
  legacy_id VARCHAR(32) NOT NULL,
  category_id BIGINT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (legacy_id),
  UNIQUE INDEX idx_activity_category_legacy_ids_category_id (category_id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci;

INSERT INTO activity_category_legacy_ids (legacy_id, category_id)
SELECT legacy_id, category_id
FROM temporary_activity_category_id_map;

ALTER TABLE activity_categories
  ADD COLUMN numeric_id BIGINT NULL AFTER id;

UPDATE activity_categories AS category
INNER JOIN activity_category_legacy_ids AS mapping
  ON mapping.legacy_id = category.id
SET category.numeric_id = mapping.category_id;

ALTER TABLE activities
  ADD COLUMN category_id_numeric BIGINT NULL AFTER category_id;

UPDATE activities AS activity
INNER JOIN activity_category_legacy_ids AS mapping
  ON mapping.legacy_id = activity.category_id
SET activity.category_id_numeric = mapping.category_id;

-- ============================================================
-- 阶段三：删除旧字段前的强制断言
-- ============================================================

DROP PROCEDURE IF EXISTS assert_activity_category_numeric_fill;

DELIMITER $$

CREATE PROCEDURE assert_activity_category_numeric_fill()
BEGIN
  DECLARE category_count BIGINT DEFAULT 0;
  DECLARE mapping_count BIGINT DEFAULT 0;
  DECLARE activity_count BIGINT DEFAULT 0;
  DECLARE mapped_activity_count BIGINT DEFAULT 0;

  SELECT COUNT(*) INTO category_count
  FROM activity_categories;

  SELECT COUNT(*) INTO mapping_count
  FROM activity_category_legacy_ids;

  SELECT COUNT(*) INTO activity_count
  FROM activities;

  SELECT COUNT(category_id_numeric) INTO mapped_activity_count
  FROM activities;

  IF category_count <> mapping_count THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：分类数量与兼容映射数量不一致';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM activity_categories
    WHERE numeric_id IS NULL
  ) THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：存在未生成数字 ID 的分类';
  END IF;

  IF activity_count <> mapped_activity_count THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：存在未映射到数字分类 ID 的活动';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM activity_category_legacy_ids AS mapping
    LEFT JOIN activity_categories AS category
      ON category.numeric_id = mapping.category_id
    WHERE category.numeric_id IS NULL
  ) THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移已停止：兼容映射与分类数字 ID 不一致';
  END IF;
END$$

DELIMITER ;

CALL assert_activity_category_numeric_fill();
DROP PROCEDURE assert_activity_category_numeric_fill;

-- ============================================================
-- 阶段四：切换正式字段
-- ============================================================

ALTER TABLE activity_categories
  MODIFY COLUMN numeric_id BIGINT NOT NULL;

ALTER TABLE activities
  MODIFY COLUMN category_id_numeric BIGINT NOT NULL;

ALTER TABLE activities
  DROP INDEX idx_activities_category_id,
  DROP COLUMN category_id,
  CHANGE COLUMN category_id_numeric category_id BIGINT NOT NULL,
  ADD INDEX idx_activities_category_id (category_id);

ALTER TABLE activity_categories
  DROP PRIMARY KEY,
  DROP COLUMN id,
  CHANGE COLUMN numeric_id id BIGINT NOT NULL,
  ADD PRIMARY KEY (id),
  ADD UNIQUE INDEX idx_activity_categories_label (label);

ALTER TABLE activity_categories
  MODIFY COLUMN id BIGINT NOT NULL AUTO_INCREMENT;

DROP TEMPORARY TABLE temporary_activity_category_id_map;

-- ============================================================
-- 阶段五：迁移结果断言与结果输出
-- ============================================================

DROP PROCEDURE IF EXISTS assert_activity_category_numeric_result;

DELIMITER $$

CREATE PROCEDURE assert_activity_category_numeric_result()
BEGIN
  DECLARE category_id_type VARCHAR(64) DEFAULT '';
  DECLARE activity_category_id_type VARCHAR(64) DEFAULT '';
  DECLARE category_id_extra VARCHAR(255) DEFAULT '';
  DECLARE orphan_activity_count BIGINT DEFAULT 0;

  SELECT
    COALESCE(MAX(DATA_TYPE), ''),
    COALESCE(MAX(EXTRA), '')
  INTO category_id_type, category_id_extra
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'activity_categories'
    AND COLUMN_NAME = 'id';

  SELECT COALESCE(MAX(DATA_TYPE), '')
  INTO activity_category_id_type
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'activities'
    AND COLUMN_NAME = 'category_id';

  IF category_id_type <> 'bigint'
    OR category_id_extra NOT LIKE '%auto_increment%' THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移结果异常：activity_categories.id 不是 BIGINT AUTO_INCREMENT';
  END IF;

  IF activity_category_id_type <> 'bigint' THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移结果异常：activities.category_id 不是 BIGINT';
  END IF;

  SELECT COUNT(*)
  INTO orphan_activity_count
  FROM activities AS activity
  LEFT JOIN activity_categories AS category
    ON category.id = activity.category_id
  WHERE category.id IS NULL;

  IF orphan_activity_count > 0 THEN
    SIGNAL SQLSTATE '45000'
      SET MESSAGE_TEXT = '迁移结果异常：存在无对应分类的活动';
  END IF;
END$$

DELIMITER ;

CALL assert_activity_category_numeric_result();
DROP PROCEDURE assert_activity_category_numeric_result;

SELECT
  mapping.legacy_id,
  mapping.category_id,
  category.label
FROM activity_category_legacy_ids AS mapping
INNER JOIN activity_categories AS category
  ON category.id = mapping.category_id
ORDER BY mapping.category_id ASC;

SELECT
  (SELECT COUNT(*) FROM activity_categories) AS category_count,
  (SELECT COUNT(*) FROM activity_category_legacy_ids) AS mapping_count,
  (SELECT COUNT(*) FROM activities) AS activity_count,
  (
    SELECT COUNT(*)
    FROM activities AS activity
    LEFT JOIN activity_categories AS category
      ON category.id = activity.category_id
    WHERE category.id IS NULL
  ) AS orphan_activity_count;
