ALTER TABLE users
  ADD COLUMN real_name varchar(64) NOT NULL DEFAULT '' AFTER password_hash,
  ADD COLUMN id_card_mask varchar(32) NOT NULL DEFAULT '' AFTER real_name,
  ADD COLUMN is_real_name_verified tinyint(1) NOT NULL DEFAULT 0 AFTER id_card_mask,
  ADD COLUMN real_name_verified_at datetime(3) NULL AFTER is_real_name_verified;
