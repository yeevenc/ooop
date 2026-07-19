ALTER TABLE chat_conversations
  ADD COLUMN user_a_deleted_before_id bigint NOT NULL DEFAULT 0 AFTER user_b_last_read_message_id,
  ADD COLUMN user_b_deleted_before_id bigint NOT NULL DEFAULT 0 AFTER user_a_deleted_before_id;
