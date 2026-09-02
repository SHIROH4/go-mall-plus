-- Add lease-based claiming to existing Outbox tables.
-- New environments already receive these columns from deploy/sql/init.sql.
ALTER TABLE `outbox`
    ADD COLUMN IF NOT EXISTS `lock_token` varchar(64) NOT NULL DEFAULT '' COMMENT 'Worker 抢占令牌' AFTER `error_message`,
    ADD COLUMN IF NOT EXISTS `locked_at` datetime DEFAULT NULL COMMENT 'Worker 抢占时间' AFTER `lock_token`;
