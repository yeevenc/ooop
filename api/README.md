# OoopAdmin API

Golang RESTful JSON API 服务，第一版提供用户登录注册能力，供 APP 项目调用。

## 技术栈

- Go 1.22
- Gin
- GORM
- MySQL
- JWT access token / refresh token
- 阿里云号码认证与短信服务

## 启动前准备

1. 安装 Go 1.22 或以上版本。
2. 创建 MySQL 数据库。

```sql
CREATE DATABASE ooop_admin DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

3. 按 `.env.example` 配置环境变量。

4. 执行数据表迁移。

```bash
go run ./cmd/migrate
```

## 启动服务

```bash
go mod tidy
go run ./cmd/api
```

服务默认监听 `http://127.0.0.1:8080`。

## APP 调用接口

完整 APP 用户接口文档见：

```text
docs/app-user-api.md
```

所有接口返回统一结构：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 阿里云手机号一键登录

`POST /api/v1/auth/aliyun-mobile-login`

```json
{
  "access_token": "APP 端从阿里云号码认证 SDK 获取的一键登录凭证"
}
```

首次登录时，手机号不存在会自动创建用户；手机号存在则直接登录。

### 发送手机号验证码

`POST /api/v1/auth/send-code`

```json
{
  "phone": "13800138000"
}
```

### 手机号验证码登录

`POST /api/v1/auth/mobile-code-login`

```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

### 账号密码登录

`POST /api/v1/auth/password-login`

`account` 支持手机号或用户设置的用户名。

```json
{
  "account": "13800138000",
  "password": "password123"
}
```

### 设置密码

`POST /api/v1/auth/set-password`

请求头：

```http
Authorization: Bearer <access_token>
```

```json
{
  "username": "test_user",
  "password": "password123"
}
```

### 刷新令牌

`POST /api/v1/auth/refresh-token`

```json
{
  "refresh_token": "<refresh_token>"
}
```

### 获取当前用户信息

`GET /api/v1/user/profile`

请求头：

```http
Authorization: Bearer <access_token>
```

## 本地验证

```bash
go test ./...
```

数据表迁移请执行 `go run ./cmd/migrate`。
