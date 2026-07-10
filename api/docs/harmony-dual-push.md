# HarmonyOS 双通道推送改造说明

## 目标

保留极光长连接的前台实时能力，并直接接入 HarmonyOS Push Kit，覆盖应用在后台、锁屏和进程不存在时的系统通知。

## 通道职责

- 极光通道：发送自定义消息，`distribution` 固定为 `jpush`，用于应用进程存活时刷新站内消息。
- 鸿蒙通道：发送 `foregroundShow: false` 的通知消息，负责后台、锁屏和进程不存在时的系统通知栏展示。
- 两个通道使用同一个站内消息 ID 作为 `messageId`，客户端按该字段去重。

极光不再发送通知栏消息，也不再使用免费厂商辅助通道，避免与鸿蒙官方通知重复展示。

客户端需要在唯一声明 `action.ohos.push.listener` 的 `PushMessageAbility` 中注册 `pushService.receiveMessage('DEFAULT', ...)`。应用在前台时，Push Kit 会通过该回调传递被抑制展示的通知数据；该数据与极光自定义消息仍按 `messageId` 去重。

发送通知前，客户端需要调用 `notificationManager.requestEnableNotification()` 引导用户开启系统通知权限。点击通知进入首页时，在入口 Ability 的 `onCreate()` 和 `onNewWant()` 中读取 `messageId`、`activityId` 和 `type`。

## 鸿蒙通知消息体（对齐官方 push-send-alert）

```http
POST https://push-api.cloud.huawei.com/v3/{projectId}/messages:send
Content-Type: application/json; charset=UTF-8
Authorization: Bearer <JWT>
push-type: 0
```

```json
{
  "payload": {
    "notification": {
      "category": "WORK",
      "title": "活动审核通知",
      "body": "您发布的活动审核成功。",
      "clickAction": {
        "actionType": 0,
        "data": {
          "messageId": "88",
          "activityId": "99",
          "type": "activity_review"
        }
      },
      "foregroundShow": false,
      "badge": { "addNum": 1 },
      "notifyId": 88
    }
  },
  "target": {
    "token": ["PushToken"]
  },
  "pushOptions": {
    "ttl": 86400,
    "testMessage": false
  }
}
```

### category 映射

当前 AGC 已开通 **WORK** 自分类权益，服务端**全部业务消息**统一发送：

| 业务类型 | category | 说明 |
|---------|----------|------|
| 审核 / 报名 / 互动 / 系统 | `WORK` | 工作事项、业务流程、审核进度提醒 |

空或非法 category 会在发送层归一为 `MARKETING`（兜底）。

`category` 必须使用华为官方枚举，**不要**使用 `SYSTEM_REMINDER`、`SOCIAL_DYNAMICS` 等自定义字符串。

后续若开通 `SUBSCRIPTION` 等权益，再按 `messageType` 细分映射。调测可开 `HARMONY_PUSH_TEST_MESSAGE=true`。

## Push Token

- 客户端每次启动时调用 `pushService.getToken()`，获取成功后上报服务端。
- Push Kit 6.1.0(23) 及以上版本同时监听 `tokenUpdate`，Token 更新后重新上报。
- Token 长度不固定，服务端使用 `varchar(2048)` 保存，不校验固定长度。
- 退出登录只解绑服务端关联，不调用 `deleteToken()`。
- 极光与 Push Kit 共用唯一声明 `action.ohos.push.listener` 的 `PushMessageAbility`。

## 服务端鉴权

服务端读取 API Console 下载的 Service Account JSON，使用其中的私钥按 PS256 生成一小时有效的 JWT，并缓存至过期前十秒。发送地址为：

```text
POST https://push-api.cloud.huawei.com/v3/{project_id}/messages:send
```

Service Account 文件仅通过环境变量配置路径，不写入仓库，不输出私钥和完整 Push Token。

## 接口约定

绑定或更新推送标识：

```http
PUT /api/v1/user/push-registration
```

```json
{
  "platform": "hmos",
  "registration_id": "极光 Registration ID",
  "harmony_push_token": "HarmonyOS Push Token"
}
```

三个字段均支持按需提交，`registration_id` 和 `harmony_push_token` 至少提交一个。

解绑当前用户推送标识：

```http
DELETE /api/v1/user/push-registration
```

## 配置项

```text
HARMONY_PUSH_SERVICE_ACCOUNT_FILE=/secure/path/service_account.json
HARMONY_PUSH_URL=https://push-api.cloud.huawei.com
HARMONY_PUSH_TEST_MESSAGE=false
```

`HARMONY_PUSH_TEST_MESSAGE` 仅用于测试环境。生产环境应关闭测试消息。
