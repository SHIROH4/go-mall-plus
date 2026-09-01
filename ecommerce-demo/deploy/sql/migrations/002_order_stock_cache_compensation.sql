ALTER TABLE `order`
    ADD COLUMN `stock_cache_restored` tinyint(1) NOT NULL DEFAULT '0'
        COMMENT '超时订单的Redis库存是否已补偿'
        AFTER `expire_time`,
    ADD KEY `idx_timeout_compensation` (`status`, `stock_cache_restored`);
