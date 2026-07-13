-- 已报名人数不再默认计入发起人：新建为 0，并修正历史数据
-- 1) 列默认值改为 0（MySQL 8+）
ALTER TABLE activities
  MODIFY COLUMN current_count int NOT NULL DEFAULT 0;

-- 2) 按「已通过」报名名额重算 current_count（无通过记录则为 0）
UPDATE activities a
SET current_count = COALESCE((
  SELECT SUM(p.count)
  FROM activity_participants p
  WHERE p.activity_id = a.id
    AND p.status = 'approved'
), 0);
