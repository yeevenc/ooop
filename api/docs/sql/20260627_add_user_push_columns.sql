ALTER TABLE users
  ADD COLUMN push_platform varchar(32) NOT NULL DEFAULT '' AFTER device_no,
  ADD COLUMN registration_id varchar(128) NOT NULL DEFAULT '' AFTER push_platform;
