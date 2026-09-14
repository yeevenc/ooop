# 后台系统通知广播

按当前 App 用户列表发送系统通知。不改 App 端：沿用已有 `type=system` 站内信，以及用户 ID（极光 alias）/ `harmony_push_token` 双通道 Push。

## 目标用户

- 账号状态为正常的 App 用户
- 排除保留用户名 `admin`
- 已注销用户不在用户表中，不会被选中
- 封禁用户不发送
- 关闭系统通知权限的用户：仍写入站内信，不发 Push

## 发送方式

1. 后台创建一条 `system_broadcasts` 记录后立即返回。
2. Worker 按用户 ID 游标分批处理。
3. 每个用户写入一条 `user_messages`（`type=system`），幂等键为 `system-broadcast:{广播ID}:{用户ID}`。
4. 复用现有 `pushToUser`：极光走用户 ID alias / registration_id，鸿蒙走 `harmony_push_token`。
5. Push extras 只带 `messageId` 和 `type=system`，不带 `activityId`。客户端点击后进入消息列表。

## 管理端接口

创建：

```http
POST /api/v1/admin/system-broadcasts
Authorization: Bearer <admin token>
```

```json
{
  "title": "系统维护通知",
  "content": "今晚 23:00 进行维护，预计 30 分钟。"
}
```

标题最多 80 字，正文最多 500 字。

列表：

```http
GET /api/v1/admin/system-broadcasts?page=1&page_size=10&status=running
```

`status` 可选：`pending` / `running` / `completed` / `failed`。

## 迁移

```bash
go run ./cmd/migrate
```

或执行 `docs/sql/20260914_add_system_broadcasts.sql`。
