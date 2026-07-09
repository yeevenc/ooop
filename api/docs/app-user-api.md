# APP 用户接口文档

基础地址：

```text
http://你的服务域名/api/v1
```

本地开发地址：

```text
http://127.0.0.1:8080/api/v1
```

## 统一返回结构

成功：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

失败：

```json
{
  "code": 400002,
  "message": "手机号格式不正确"
}
```

## 登录成功返回结构

一键登录、注册、手机号验证码登录、账号密码登录成功后，都会返回同一类数据。

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user": {
      "id": 1,
      "phone": "13800138000",
      "username": "test_user",
      "status": 1,
      "register_source": "aliyun_mobile",
      "last_login_at": "2026-06-17T14:00:00+08:00",
      "created_at": "2026-06-17T14:00:00+08:00"
    },
    "tokens": {
      "access_token": "登录访问令牌",
      "access_token_expires_in": 2592000
    }
  }
}
```

APP 端后续请求需要携带：

```http
Authorization: Bearer <access_token>
```

访问令牌有效期为一个月（2592000 秒）。系统不提供刷新令牌：令牌过期后接口返回 401，APP 端应清除本地登录态并引导用户重新登录。

## 1. 阿里云手机号一键登录

默认首次登录入口。

手机号不存在时自动注册，手机号存在时直接登录。

```http
POST /auth/aliyun-mobile-login
Content-Type: application/json
```

请求参数：

```json
{
  "access_token": "APP 端从阿里云号码认证 SDK 获取的一键登录凭证"
}
```

## 2. 手机号密码注册

用于用户主动选择手动注册时调用。

```http
POST /auth/register
Content-Type: application/json
```

请求参数：

```json
{
  "phone": "13800138000",
  "username": "test_user",
  "password": "password123"
}
```

字段说明：

```text
phone：必填，中国大陆手机号
username：选填，用户账号名
password：必填，至少 8 位
```

手机号已存在时返回：

```json
{
  "code": 400002,
  "message": "手机号已注册"
}
```

## 3. 发送登录验证码

```http
POST /auth/send-code
Content-Type: application/json
```

请求参数：

```json
{
  "phone": "13800138000"
}
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "sent": true
  }
}
```

## 4. 手机号验证码登录

手机号不存在时自动注册，手机号存在时直接登录。

```http
POST /auth/mobile-code-login
Content-Type: application/json
```

请求参数：

```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

## 5. 账号密码登录

`account` 支持手机号或用户名。

```http
POST /auth/password-login
Content-Type: application/json
```

请求参数：

```json
{
  "account": "13800138000",
  "password": "password123"
}
```

也可以：

```json
{
  "account": "test_user",
  "password": "password123"
}
```

## 6. 设置密码

用户通过一键登录或验证码登录后，可调用该接口设置密码。

```http
POST /auth/set-password
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求参数：

```json
{
  "username": "test_user",
  "password": "password123"
}
```

## 7. 获取当前用户信息

```http
GET /user/profile
Authorization: Bearer <access_token>
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "phone": "13800138000",
    "username": "test_user",
    "status": 1,
    "register_source": "aliyun_mobile",
    "last_login_at": "2026-06-17T14:00:00+08:00",
    "created_at": "2026-06-17T14:00:00+08:00"
  }
}
```

## 8. 修改当前用户资料

仅支持修改昵称、性别、地区、个性签名、头像；只提交需要修改的字段，未提交的字段保持不变。头像需先调用「上传图片」接口拿到 URL，再把 URL 提交到 `avatar`。

```http
PUT /user/profile
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求参数（均为选填）：

```json
{
  "nickname": "新昵称",
  "gender": "男",
  "region": "上海",
  "bio": "热爱生活",
  "avatar": "https://source.ooopai.cn/images/2026/06/29/1718000000000000000.jpg"
}
```

字段说明：

```text
nickname：提交时不可为空，最长 32 字
gender：最长 16 字
region：最长 64 字
bio：最长 200 字
avatar：图片 URL，最长 255 字符；由「上传图片」接口返回
```

返回更新后的完整用户信息，结构同「获取当前用户信息」。

## 9. 获取隐私和通知设置

```http
GET /user/settings
Authorization: Bearer <access_token>
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "privacy": {
      "show_region": true
    },
    "notification": {
      "notification_permission_enabled": true,
      "activity_reminder_enabled": true,
      "system_message_enabled": true
    }
  }
}
```

字段说明：

```text
show_region：是否在他人查看个人主页时展示常驻地区
notification_permission_enabled：APP 同步的手机系统通知权限状态
activity_reminder_enabled：活动提醒开关展示状态，跟随系统通知权限
system_message_enabled：系统消息开关展示状态，跟随系统通知权限
```

通知管理页面只展示「活动提醒」和「系统消息」，不再展示「互动通知」。用户点击活动提醒或系统消息开关时，APP 端弹窗提示用户前往手机系统设置开启或关闭通知权限；权限状态变化后，再调用通知设置接口同步最新系统权限。

## 10. 修改隐私设置

```http
PUT /user/privacy-settings
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求参数：

```json
{
  "show_region": false
}
```

返回更新后的设置结构，结构同「获取隐私和通知设置」。当 `show_region` 为 `false` 时，他人查看该用户个人主页不返回地区。

## 11. 同步通知设置

```http
PUT /user/notification-settings
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求参数：

```json
{
  "notification_permission_enabled": true
}
```

返回更新后的设置结构，结构同「获取隐私和通知设置」。该接口用于 APP 端同步手机系统通知权限状态，不直接代替系统权限开关。

## 12. 上传图片

用于头像等图片上传，后端会上传到七牛云 `ooop` 空间并返回可公开访问的 CDN URL。

```http
POST /upload/image
Authorization: Bearer <access_token>
Content-Type: multipart/form-data
```

表单字段：

```text
file：图片文件，字段名固定为 file；支持 jpg/jpeg/png/webp，最大 5MB
```

返回：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "url": "https://source.ooopai.cn/images/2026/06/29/1718000000000000000.jpg",
    "path": "/images/2026/06/29/1718000000000000000.jpg"
  }
}
```

## 数据表说明

### users

```text
id：用户 ID
phone：手机号，唯一
username：账号名，唯一，可为空
password_hash：密码哈希，接口不返回
status：用户状态，1 表示启用
register_source：注册来源，aliyun_mobile / mobile_code / password
hide_region：是否隐藏常驻地区，0 表示展示，1 表示隐藏
notification_disabled：是否关闭手机系统通知权限，0 表示开启，1 表示关闭
last_login_at：最后登录时间
created_at：创建时间
updated_at：更新时间
```

### login_codes

```text
id：验证码 ID
phone：手机号
scene：验证码场景，当前为 login
code_hash：验证码哈希，明文验证码不落库
used_at：使用时间
expires_at：过期时间
created_at：创建时间
```
