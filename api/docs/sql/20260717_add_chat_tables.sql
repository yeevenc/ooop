CREATE TABLE IF NOT EXISTS chat_conversations (
  id bigint NOT NULL AUTO_INCREMENT,
  user_a_id bigint NOT NULL,
  user_b_id bigint NOT NULL,
  last_message_id bigint NOT NULL DEFAULT 0,
  last_message_content varchar(2000) NOT NULL DEFAULT '',
  last_message_at datetime(3) NULL,
  user_a_unread int NOT NULL DEFAULT 0,
  user_b_unread int NOT NULL DEFAULT 0,
  user_a_last_read_message_id bigint NOT NULL DEFAULT 0,
  user_b_last_read_message_id bigint NOT NULL DEFAULT 0,
  created_at datetime(3) NULL,
  updated_at datetime(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_chat_conversation_users (user_a_id, user_b_id),
  KEY idx_chat_conversations_user_a_id (user_a_id),
  KEY idx_chat_conversations_user_b_id (user_b_id),
  KEY idx_chat_conversations_last_message_at (last_message_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS chat_messages (
  id bigint NOT NULL AUTO_INCREMENT,
  conversation_id bigint NOT NULL,
  sender_id bigint NOT NULL,
  recipient_id bigint NOT NULL,
  client_message_id varchar(64) NOT NULL,
  type varchar(16) NOT NULL DEFAULT 'text',
  content varchar(2000) NOT NULL,
  expires_at datetime(3) NOT NULL,
  created_at datetime(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_chat_sender_client_message (sender_id, client_message_id),
  KEY idx_chat_message_conversation_id (conversation_id, created_at),
  KEY idx_chat_messages_sender_id (sender_id),
  KEY idx_chat_messages_recipient_id (recipient_id),
  KEY idx_chat_messages_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS chat_push_tasks (
  id bigint NOT NULL AUTO_INCREMENT,
  message_id bigint NOT NULL,
  recipient_id bigint NOT NULL,
  channel varchar(16) NOT NULL,
  status varchar(16) NOT NULL,
  attempts int NOT NULL DEFAULT 0,
  next_retry_at datetime(3) NOT NULL,
  locked_at datetime(3) NULL,
  last_error varchar(500) NOT NULL DEFAULT '',
  created_at datetime(3) NULL,
  updated_at datetime(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uniq_chat_message_push_channel (message_id, channel),
  KEY idx_chat_push_tasks_message_id (message_id),
  KEY idx_chat_push_tasks_recipient_id (recipient_id),
  KEY idx_chat_push_schedule (status, next_retry_at),
  KEY idx_chat_push_tasks_locked_at (locked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
