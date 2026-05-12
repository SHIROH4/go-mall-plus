-- MySQL 初始化脚本 - 电商Demo
-- 强迫读取该文件的 MySQL 客户端使用 utf8mb4 交互！
SET NAMES utf8mb4;

CREATE DATABASE IF NOT EXISTS `ecommerce_demo` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `ecommerce_demo`;

-- ----------------------------
-- 1. 用户表 (User)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `user` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `username` varchar(64) NOT NULL DEFAULT '' COMMENT '用户名',
    `password` varchar(255) NOT NULL DEFAULT '' COMMENT '加密后的密码',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- ----------------------------
-- 2. 商品分类表 (Category)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `category` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(64) NOT NULL DEFAULT '' COMMENT '分类名称',
    `icon` varchar(255) NOT NULL DEFAULT '' COMMENT '分类图标',
    `sort` int(11) NOT NULL DEFAULT '0' COMMENT '排序',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商品分类表';

-- ----------------------------
-- 3. 商品表 (Product)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `product` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `name` varchar(128) NOT NULL DEFAULT '' COMMENT '商品名称',
    `desc` varchar(255) NOT NULL DEFAULT '' COMMENT '商品描述',
    `price` int(11) NOT NULL DEFAULT '0' COMMENT '商品价格(单位:分)',
    `image_url` varchar(255) NOT NULL DEFAULT '' COMMENT '商品图片URL',
    `category_id` bigint(20) NOT NULL DEFAULT '0' COMMENT '分类ID',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商品表';

-- ----------------------------
-- 4. 库存表 (Stock)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `stock` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `product_id` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '商品ID',
    `stock_num` int(11) NOT NULL DEFAULT '0' COMMENT '真实库存量',
    `version` int(11) NOT NULL DEFAULT '0' COMMENT '乐观锁版本号(防超卖兜底)',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_product_id` (`product_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='库存表';

-- ----------------------------
-- 5. 订单表 (Order)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `order` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `order_no` varchar(64) NOT NULL DEFAULT '' COMMENT '业务订单号(雪花算法)',
    `user_id` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '用户ID',
    `product_id` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '商品ID',
    `count` int(11) NOT NULL DEFAULT '0' COMMENT '购买数量',
    `total_amount` int(11) NOT NULL DEFAULT '0' COMMENT '总金额(单位:分)',
    `status` tinyint(3) NOT NULL DEFAULT '0' COMMENT '订单状态: 0待支付 1已支付 2已取消 3已超时',
    `expire_time` datetime DEFAULT NULL COMMENT '订单超时时间',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_order_no` (`order_no`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_status_expire` (`status`, `expire_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单表';

-- ----------------------------
-- 6. 支付表 (Payment)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `payment` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `payment_no` varchar(32) NOT NULL DEFAULT '' COMMENT '支付单号',
    `order_no` varchar(64) NOT NULL DEFAULT '' COMMENT '关联订单号',
    `user_id` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '用户ID',
    `amount` bigint(20) NOT NULL DEFAULT '0' COMMENT '支付金额(单位:分)',
    `status` tinyint(3) NOT NULL DEFAULT '0' COMMENT '状态: 0待支付 1已支付 2已取消 3已超时',
    `pay_channel` varchar(20) DEFAULT '' COMMENT '支付渠道: alipay/wechat',
    `pay_time` datetime DEFAULT NULL COMMENT '支付时间',
    `expire_time` datetime NOT NULL COMMENT '过期时间',
    `callback_data` text COMMENT '回调原始数据',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_payment_no` (`payment_no`),
    KEY `idx_order_no` (`order_no`),
    KEY `idx_user_id` (`user_id`),
    KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='支付表';

-- ----------------------------
-- 7. 收货地址表 (Address)
-- ----------------------------
CREATE TABLE IF NOT EXISTS `address` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `receiver_name` varchar(50) NOT NULL COMMENT '收货人姓名',
  `phone` varchar(20) NOT NULL COMMENT '联系电话',
  `province` varchar(50) NOT NULL COMMENT '省份',
  `city` varchar(50) NOT NULL COMMENT '城市',
  `district` varchar(50) NOT NULL COMMENT '区县',
  `detail_address` varchar(200) NOT NULL COMMENT '详细地址',
  `postal_code` varchar(10) DEFAULT '' COMMENT '邮政编码',
  `is_default` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否默认地址',
  `create_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='收货地址表';

-- ----------------------------
-- 初始化测试数据
-- ----------------------------
-- 插入分类数据
INSERT INTO `category` (`id`, `name`, `icon`, `sort`) VALUES
(1, '手机数码', '/icons/phone.png', 1),
(2, '电脑办公', '/icons/laptop.png', 2),
(3, '影音娱乐', '/icons/music.png', 3),
(4, '智能设备', '/icons/smart.png', 4),
(5, '生活电器', '/icons/home.png', 5),
(6, '游戏主机', '/icons/game.png', 6),
(7, '美妆护肤', '/icons/beauty.png', 7),
(8, '酒水饮料', '/icons/wine.png', 8);

-- 插入商品数据
INSERT INTO `product` (`id`, `name`, `desc`, `price`, `image_url`, `category_id`) VALUES
(1, 'Apple iPhone 15 Pro', '256GB 钛金属', 899900, '/images/iphone15.jpg', 1),
(2, 'MacBook Pro 14寸', 'M3 Pro芯片 18+512G', 1699900, '/images/macbook.jpg', 2),
(3, 'AirPods Pro 2', '主动降噪无线耳机', 189900, '/images/airpods.jpg', 3),
(4, '小米14 Ultra', '徕卡影像 骁龙8Gen3', 649900, '/images/xiaomi14.jpg', 1),
(5, '戴森吹风机', '智能温控 快速干发', 299900, '/images/dyson.jpg', 5),
(6, 'Switch OLED', '游戏机 日版', 209900, '/images/switch.jpg', 6),
(7, 'SK-II 神仙水', '护肤精华露 230ml', 89900, '/images/sk2.jpg', 7),
(8, '茅台飞天', '53度 500ml', 149900, '/images/maotai.jpg', 8);

-- 插入库存数据
INSERT INTO `stock` (`product_id`, `stock_num`, `version`) VALUES
(1, 50, 0), (2, 30, 0), (3, 100, 0), (4, 80, 0),
(5, 40, 0), (6, 60, 0), (7, 120, 0), (8, 20, 0);
