# 活动图片自动审核

## 目标

活动发布接口只完成活动与审核任务落库，不同步等待阿里云图片审核。后台任务完成审核后复用活动原有审核通过或拒绝流程，减少后台人工审核量。

## 处理流程

1. 发布活动时，在同一数据库事务中写入 `activities` 和 `activity_image_audit_tasks`。
2. 服务启动及运行期间对账所有 `pending` 活动，不限制发布时间，缺失任务会自动补建。
3. 后台单 Worker 每次只领取一个到期任务，处理完成后再按顺序领取下一条。
4. 所有图片场景均返回 `pass` 时，活动自动审核通过。
5. 任一场景返回 `block` 时，活动自动拒绝，并以自然文案告知发布者对应图片、场景和标签。
6. 任一场景返回 `review` 且标签不是 `normal` 时，按内容规则自动拒绝；标签为 `normal` 时保留待审核，由后台人工复核。
7. `ad` 场景中的 `spam`、`npx`、`qrcode`、`programCode`、`ad` 标签按通过处理，其他场景结果不受影响。
8. 调用失败、图片下载超时、数据库临时失败时，任务按退避时间持续重试。

## 连续性保证

- 活动与审核任务使用同一事务，避免活动存在但审核任务缺失。
- 服务启动及运行期间持续对账所有待审核活动，历史数据和异常断层会自动续审。
- 单个审核 Worker 每次只领取一条任务，避免提前锁定后排任务和并发审核压力。
- 多实例部署时只允许一个实例启用 `ALIYUN_IMAGE_AUDIT_ENABLED`，其余实例关闭图片审核 Worker。
- 处理锁超时后自动释放，服务重启不会丢失任务。
- 阿里云结果先持久化，再修改活动状态；后续节点失败不会重复调用计费接口。
- 活动状态只允许从 `pending` 原子更新，人工和自动审核不会互相覆盖。
- 自动审核通知使用幂等标识，重试时不会重复生成站内消息。
- 重试不设置永久终止次数，异常任务按最长一小时的间隔持续恢复。
- 服务重启保留原有重试时间，不会提前唤醒供应商故障期间的退避任务。
- 新发布活动最多上传五张图片，单个活动只调用一次阿里云批量审核接口。

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
ALIYUN_IMAGE_AUDIT_POLL_INTERVAL=5s
ALIYUN_IMAGE_AUDIT_LOCK_TIMEOUT=2m
ALIYUN_IMAGE_AUDIT_RECOVERY_INTERVAL=1m
ALIYUN_IMAGE_AUDIT_RECONCILE_INTERVAL=10m
ALIYUN_IMAGE_AUDIT_BATCH_SIZE=1
```

AccessKey 继续复用服务端已有的 `ALIYUN_ACCESS_KEY_ID` 和 `ALIYUN_ACCESS_KEY_SECRET`。七牛图片地址必须能被阿里云在三秒内下载，私有空间需要提供有效期足够的签名地址。

普通任务按五秒轮询且每次只领取一条；卡死任务每分钟恢复一次；全量待审核活动每十分钟补偿对账一次。三个周期彼此独立，减少空队列时的数据库压力。

## 上线顺序

1. 开通阿里云视觉智能开放平台的图片内容安全能力并完成 RAM 授权。
2. 执行 `docs/sql/20260728_add_activity_image_audit.sql`。
3. 执行 `docs/sql/20260728_optimize_async_task_indexes.sql`。
4. 配置环境变量并部署服务端。
5. 通过测试图片验证 `pass`、`review`、`block` 和超时重试四条链路。
