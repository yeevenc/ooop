ALTER TABLE users
  ADD COLUMN is_official tinyint(1) NOT NULL DEFAULT 0 AFTER ban_reason;
