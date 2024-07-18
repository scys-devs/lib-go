CREATE TABLE `purchase`
(
    `id`         bigint(20)                                                   NOT NULL AUTO_INCREMENT,
    `user_id`    bigint(20)                                                   NOT NULL,
    `pkg_id`     varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    `txn_id`     varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    `gmt_create` bigint(20)                                                   NOT NULL,
    `gmt_expire` bigint(20)                                                   NOT NULL,
    `gmt_refund` bigint(20)                                                   NOT NULL DEFAULT '0',
    `env`        varchar(12) COLLATE utf8mb4_general_ci                       NOT NULL DEFAULT '',
    `platform`   tinyint(1)                                                   NOT NULL DEFAULT 1 COMMENT '1=ios 2=android',
    PRIMARY KEY (`id`) USING BTREE,
    UNIQUE KEY `user_id` (`txn_id`, `user_id`) USING BTREE
) DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci
;


CREATE TABLE `purchase_subs`
(
    `id`          bigint(20)                                                   NOT NULL AUTO_INCREMENT,
    `original_id` varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    `pkg_id`      varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    `periods`     int(11)                                                      NOT NULL,
    `gmt_create`  bigint(20)                                                   NOT NULL,
    `gmt_latest`  bigint(20)                                                   NOT NULL,
    `gmt_cancel`  bigint(20)                                                   NOT NULL DEFAULT '0',
    `platform`    tinyint(1)                                                   NOT NULL DEFAULT 1 COMMENT '1=ios 2=android',
    PRIMARY KEY (`id`),
    UNIQUE KEY `original_id` (`original_id`)
) DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci
;


CREATE TABLE `user`
(
    `id`              bigint(20)                                                    NOT NULL AUTO_INCREMENT,
    `uuid`            varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci  NOT NULL,
    `device`          varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci  NOT NULL,
    `device_system`   varchar(30) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci  NOT NULL,
    `gmt_create`      bigint(20)                                                    NOT NULL,
    `subs_expires_at` bigint(20)                                                    NOT NULL DEFAULT '0',
    `subs_pkg_id`     varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci  NOT NULL DEFAULT '',
    `ip_addr`         varchar(16) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci  NOT NULL DEFAULT '',
    `fcm_token`       varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '',
    `lang`            varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci  NOT NULL DEFAULT 'en',
    `timezone_offset` int(11)                                                       NOT NULL DEFAULT '0',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uuid` (`uuid`) USING BTREE
) DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci
;
