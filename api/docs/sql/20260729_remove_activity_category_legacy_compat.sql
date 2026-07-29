-- 危险操作：此脚本不能随数字 ID 迁移脚本一起执行。
-- 执行前必须同时满足以下条件：
-- 1. 已确认老版本 App 停止使用或已完成强制升级；
-- 2. 服务端请求日志在观察期内不再出现英文分类 ID；
-- 3. 已删除 ResolveCategoryID 的英文 ID 查询分支；
-- 4. 已删除未携带 X-App-Category-ID-Version 时的字符串响应兼容；
-- 5. 已备份 activity_category_legacy_ids 表。
DROP TABLE activity_category_legacy_ids;
