ALTER TABLE users
  ADD COLUMN hide_region tinyint(1) NOT NULL DEFAULT 0 AFTER registration_id,
  ADD COLUMN notification_disabled tinyint(1) NOT NULL DEFAULT 0 AFTER hide_region;
