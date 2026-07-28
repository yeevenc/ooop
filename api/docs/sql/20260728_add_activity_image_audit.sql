ALTER TABLE activities
  ADD COLUMN review_source varchar(32) NOT NULL DEFAULT '' COMMENT '审核来源：admin/image_audit',
  ADD COLUMN review_reason varchar(500) NOT NULL DEFAULT '' COMMENT '审核原因',
  ADD COLUMN reviewed_at datetime(3) NULL COMMENT '审核完成时间';

ALTER TABLE user_messages
  ADD COLUMN idempotency_key varchar(100) NULL COMMENT '可重试业务通知幂等标识',
  ADD UNIQUE KEY uniq_user_messages_idempotency_key (idempotency_key);

CREATE TABLE IF NOT EXISTS activity_image_audit_tasks (
  id bigint NOT NULL AUTO_INCREMENT,
  activity_id bigint NOT NULL,
  image_urls_json text NOT NULL,
  status varchar(24) NOT NULL,
  decision varchar(16) NOT NULL DEFAULT '',
  attempts int NOT NULL DEFAULT 0,
  next_retry_at datetime(3) NOT NULL,
  locked_at datetime(3) NULL,
  result_json longtext NULL,
  reject_reason varchar(500) NOT NULL DEFAULT '',
  notification_done tinyint(1) NOT NULL DEFAULT 0,
  last_error varchar(500) NOT NULL DEFAULT '',
  completed_at datetime(3) NULL,
  created_at datetime(3) NULL,
  updated_at datetime(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_activity_image_audit_tasks_activity_id (activity_id),
  KEY idx_activity_image_audit_schedule (status, next_retry_at),
  KEY idx_activity_image_audit_tasks_locked_at (locked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
