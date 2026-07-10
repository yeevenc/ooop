ALTER TABLE users
  ADD COLUMN harmony_push_token varchar(2048) NOT NULL DEFAULT '' AFTER registration_id;
