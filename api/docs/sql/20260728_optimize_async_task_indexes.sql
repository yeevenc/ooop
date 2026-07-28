ALTER TABLE chat_push_tasks
  ADD KEY idx_chat_push_recovery (status, locked_at);

ALTER TABLE activity_image_audit_tasks
  ADD KEY idx_activity_image_audit_recovery (status, locked_at);
