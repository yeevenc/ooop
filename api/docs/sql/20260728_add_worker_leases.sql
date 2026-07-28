CREATE TABLE IF NOT EXISTS worker_leases (
  name varchar(100) NOT NULL,
  owner_id varchar(128) NOT NULL,
  expires_at datetime(3) NOT NULL,
  created_at datetime(3) NULL,
  updated_at datetime(3) NULL,
  PRIMARY KEY (name),
  KEY idx_worker_leases_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
