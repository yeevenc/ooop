-- 用户主页封面：空字符串表示未上传，由 App 继续展示现有默认封面。
ALTER TABLE users
  ADD COLUMN cover_url VARCHAR(500) NOT NULL DEFAULT '' AFTER avatar;
