ALTER TABLE activity_participants
  ADD COLUMN contact_info VARCHAR(64) NOT NULL DEFAULT '' COMMENT '报名联系方式' AFTER remark;
