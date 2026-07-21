ALTER TABLE chat_reports
  ADD COLUMN restriction_until datetime(3) NULL COMMENT '本次举报处理设置的聊天限制解除时间',
  ADD KEY idx_chat_reports_restriction_until (restriction_until);
