ALTER TABLE users
  ADD COLUMN chat_restricted_until datetime(3) NULL COMMENT '聊天功能限制解除时间',
  ADD COLUMN chat_restriction_reason varchar(500) NOT NULL DEFAULT '' COMMENT '聊天功能限制原因',
  ADD KEY idx_users_chat_restricted_until (chat_restricted_until);
