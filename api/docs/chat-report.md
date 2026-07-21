# 聊天举报功能

## 审核目标

- 聊天页面始终展示文字“举报”入口，不依赖长按、侧滑或隐藏菜单。
- 举报提交后由后台统一查看和处理。
- 后台处理结果通过站内消息通知举报人。
- 举报成立后限制被举报用户的聊天功能，默认 24 小时，后台可选择任意未来解除时间。
- 聊天消息仅保留 3～7 天，因此提交举报时固化最近 50 条当前用户可见消息作为证据快照。

## App 接口

```http
POST /api/v1/chat/conversations/{id}/reports
Authorization: Bearer <access_token>
Content-Type: application/json
```

```json
{
  "reason": "harassment",
  "description": "对方持续发送侮辱性内容"
}
```

举报原因支持：

- `spam`：垃圾广告
- `harassment`：骚扰辱骂
- `pornography`：色情低俗
- `fraud`：诈骗行为
- `illegal`：违法违规
- `other`：其他问题，必须填写补充说明

同一用户、同一会话只能存在一条待处理举报，避免重复提交。

聊天页面进入时查询当前用户的聊天权限：

```http
GET /api/v1/chat/access-status
Authorization: Bearer <access_token>
```

限制期间服务端拒绝发送文字和图片消息，返回 HTTP `403`、业务码 `403003`。App 在活动详情“问问”和消息列表两个入口进入聊天页面时都会显示不可关闭的限制提示，并在解除时间到达后自动向服务端复核。

## 后台接口

```http
GET /api/v1/admin/chat-reports?page=1&page_size=20&status=pending&keyword=
GET /api/v1/admin/chat-reports/{id}
PUT /api/v1/admin/chat-reports/{id}/resolve
```

处理请求：

```json
{
  "status": "resolved",
  "result": "经核查存在骚扰行为，平台已对被举报账号进行处理。",
  "restriction_until": "2026-07-21T16:00:00+08:00"
}
```

处理状态支持：

- `resolved`：举报成立并限制被举报用户聊天，必须提供未来的 `restriction_until`
- `dismissed`：经核查未发现违规

举报状态更新、被举报用户聊天限制与站内消息写入在同一数据库事务中完成。若用户已有更晚的聊天限制，新处理不会缩短原限制。处理成功后，举报人会收到标题为“举报处理结果”的站内消息；第三方 Push 失败不会回滚已经完成的处理和站内消息。
