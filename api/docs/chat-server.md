# 单聊服务端设计

## 目标

- 支持文字、Unicode Emoji 与图片单聊。
- 按 1000 名用户同时在线的容量目标设计，发送接口不等待第三方 Push 网络请求。
- 消息默认保存 7 天，可通过配置调整为 3～7 天。
- 复用现有极光自定义消息与 HarmonyOS Push Kit 双通道能力。
- 第三方 Push 暂时不可用时消息仍可恢复，不以 Push 是否到达作为消息是否成功的依据。

## 投递链路

1. 服务端校验接收人、消息长度、客户端消息号和敏感词。
2. 在一个数据库事务内创建消息、更新会话摘要与未读数，并分别创建极光、鸿蒙投递任务。
3. 接口在事务提交后立即返回，不同步等待第三方 Push。
4. 后台工作器按通道领取任务并发投递，失败后退避重试；某一通道成功不会重复投递该通道。
5. 极光自定义消息携带聊天数据，供应用进程存活时实时刷新。
6. 鸿蒙通知使用统一隐私文案“您有新会话”，`foregroundShow=false`，只负责后台、锁屏和进程不存在时的系统通知。
7. 客户端统一按 `messageId` 去重；发现消息游标不连续时通过 `after_id` 补拉数据库消息。
8. 发送侧使用内存令牌桶保护 2 核服务器：单用户 5 条/秒、允许突发 10 条；单节点全局 200 条/秒、允许突发 400 条，超过时返回 HTTP 429。

> 第三方 Push 无法提供物理意义上的绝对不断线保证。本方案保证消息先落库、投递任务可重试、客户端可补拉，从而实现业务层面的可恢复和不丢消息。

## 接口

所有接口均需要 APP Access Token。

### 发送消息

```http
POST /api/v1/chat/messages
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "recipient_id": 3001,
  "client_message_id": "0190f25d-6b71-7b68-bc4f-5ce7962a60c6",
  "type": "text",
  "content": "你好 😊"
}
```

`type` 支持 `text` 与 `image`，省略时按 `text` 处理。图片消息的 `content` 为上传成功后的 HTTP/HTTPS 地址，会话摘要显示为“[图片]”。

`client_message_id` 由客户端生成，同一发送人下必须唯一。网络超时后使用原值重试，服务端返回同一条消息，不重复增加未读数和投递任务。

### 会话列表

```http
GET /api/v1/chat/conversations?page=1&page_size=20
```

### 历史消息

向前翻页：

```http
GET /api/v1/chat/conversations/{id}/messages?before_id=100&page_size=50
```

断线补拉：

```http
GET /api/v1/chat/conversations/{id}/messages?after_id=100&page_size=100
```

`before_id` 与 `after_id` 不能同时使用。`after_id` 按消息 ID 正序返回，客户端可连续请求直到返回数量小于 `page_size`。

### 标记已读

```http
PUT /api/v1/chat/conversations/{id}/read
Content-Type: application/json
```

```json
{
  "last_message_id": 100
}
```

### 未读总数

```http
GET /api/v1/chat/unread-count
```

## Push 数据

两个通道共用以下路由字段：

```json
{
  "messageId": "100",
  "conversationId": "20",
  "senderId": "3000",
  "type": "chat_message",
  "messageType": "text"
}
```

极光 `msg_content` 额外包含完整消息 JSON；鸿蒙通知标题为“新会话”，正文为“您有新会话”，不暴露发送人和正文。

## 配置

```text
CHAT_MESSAGE_RETENTION=168h
CHAT_PUSH_INTERVAL=1s
CHAT_CLEANUP_INTERVAL=1h
CHAT_PUSH_BATCH_SIZE=100
CHAT_PUSH_WORKERS=4
CHAT_PUSH_CATEGORY=WORK
```

`CHAT_MESSAGE_RETENTION` 会限制在 72～168 小时。当前项目 AGC 已开通 `WORK` 分类，因此默认沿用该分类；正式申请即时通讯分类权益后可将 `CHAT_PUSH_CATEGORY` 调整为 `IM`。

## 数据清理

- 定时分批删除 `expires_at` 已过期的聊天消息，避免长事务影响在线请求。
- 删除已无消息的空会话。
- 删除超过消息保留期的投递任务。
- 会话列表只保留仍有有效消息的会话摘要。
