-- APP 用户封禁能力（后台管理操作，作用于 APP 端账号，与 admin 管理员无关）
-- status: 1=正常 0=封禁
-- banned_until: 限时解封时间；status=0 且本字段为 NULL 表示永久封禁
-- ban_reason: 封禁原因/备注，可展示给 APP 用户
ALTER TABLE users
  ADD COLUMN banned_until datetime NULL DEFAULT NULL AFTER status,
  ADD COLUMN ban_reason varchar(255) NOT NULL DEFAULT '' AFTER banned_until;
