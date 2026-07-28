# 活动图片自动审核

## 目标

活动发布接口只完成活动与审核任务落库，不同步等待阿里云图片审核。后台任务完成审核后复用活动原有审核通过或拒绝流程，减少后台人工审核量。

## 处理流程

1. 发布活动时，在同一数据库事务中写入 `activities` 和 `activity_image_audit_tasks`。
2. 发布接口返回成功后，后台任务领取待处理记录并调用阿里云 `ScanImage`。
3. 所有图片场景均返回 `pass` 时，活动自动审核通过。
4. 任一场景返回 `block` 时，活动自动拒绝，命中图片、场景、标签和置信度作为拒绝原因发送给发布者。
5. 任一场景返回 `review` 且标签不是 `normal` 时，按命中规则自动拒绝；标签为 `normal` 时保留待审核，由后台人工复核。
6. 调用失败、图片下载超时、数据库临时失败时，任务按退避时间持续重试。

## 连续性保证

- 活动与审核任务使用同一事务，避免活动存在但审核任务缺失。
- 服务启动及运行期间定时补建待审核活动缺失的任务，历史数据和异常断层会自动续审。
- 多进程通过数据库租约选出唯一主 Worker，非主进程不扫描、不领取任务，主进程异常后自动接管。
- 任务领取使用条件更新，支持多实例并发执行。
- 处理锁超时后自动释放，服务重启不会丢失任务。
- 阿里云结果先持久化，再修改活动状态；后续节点失败不会重复调用计费接口。
- 活动状态只允许从 `pending` 原子更新，人工和自动审核不会互相覆盖。
- 自动审核通知使用幂等标识，重试时不会重复生成站内消息。
- 重试不设置永久终止次数，异常任务按最长一小时的间隔持续恢复。

## 审核场景

- `porn`：色情、低俗内容
- `terrorism`：暴恐、敏感内容及风险人物
- `ad`：广告、二维码及图片文字风险
- `live`：涉毒、赌博等不良场景

## 配置

```dotenv
ALIYUN_IMAGE_AUDIT_ENABLED=true
ALIYUN_IMAGE_AUDIT_ENDPOINT=imageaudit.cn-shanghai.aliyuncs.com
ALIYUN_IMAGE_AUDIT_REGION_ID=cn-shanghai
ALIYUN_IMAGE_AUDIT_SCENES=porn,terrorism,ad,live
ALIYUN_IMAGE_AUDIT_POLL_INTERVAL=2s
ALIYUN_IMAGE_AUDIT_LOCK_TIMEOUT=2m
ALIYUN_IMAGE_AUDIT_RECOVERY_INTERVAL=1m
ALIYUN_IMAGE_AUDIT_BATCH_SIZE=10
ALIYUN_IMAGE_AUDIT_WORKERS=2
```

AccessKey 继续复用服务端已有的 `ALIYUN_ACCESS_KEY_ID` 和 `ALIYUN_ACCESS_KEY_SECRET`。七牛图片地址必须能被阿里云在三秒内下载，私有空间需要提供有效期足够的签名地址。

普通任务按轮询间隔领取；卡死任务的锁恢复按一分钟单独执行，避免无任务时持续写数据库。

## 上线顺序

1. 开通阿里云视觉智能开放平台的图片内容安全能力并完成 RAM 授权。
2. 执行 `docs/sql/20260728_add_activity_image_audit.sql`。
3. 执行 `docs/sql/20260728_optimize_async_task_indexes.sql`。
4. 执行 `docs/sql/20260728_add_worker_leases.sql`。
5. 配置环境变量并部署服务端。
6. 通过测试图片验证 `pass`、`review`、`block` 和超时重试四条链路。
