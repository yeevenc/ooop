# 本地密钥目录

本目录用于存放 **不进 Git** 的本地密钥文件，避免放在项目外路径导致换机/清理磁盘时丢失。

## 鸿蒙推送 Service Account

1. 从 AppGallery Connect 下载服务账号 JSON  
   （项目设置 → API 密钥 / 服务账号）
2. 保存为本目录下固定文件名：

```text
api/secrets/harmony-push-service-account.json
```

3. `api/.env` 配置（相对 `api` 工作目录）：

```env
HARMONY_PUSH_SERVICE_ACCOUNT_FILE=secrets/harmony-push-service-account.json
HARMONY_PUSH_URL=https://push-api.cloud.huawei.com
HARMONY_PUSH_TEST_MESSAGE=true
```

JSON 中需包含：`project_id`、`key_id`、`private_key`、`sub_account`、`token_uri`。

> `*.json` 密钥已在仓库 `.gitignore` 中忽略，请勿强制 `git add -f` 提交。
