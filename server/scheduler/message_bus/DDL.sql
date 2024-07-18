CREATE TABLE `message_bus`
(
    `id`         bigint                                                        NOT NULL AUTO_INCREMENT,
    `user_id`    bigint                                                        NOT NULL,
    `group_id`   varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL,
    `gmt_create` bigint                                                        NOT NULL,
    `sent`       int                                                           NOT NULL,
    PRIMARY KEY (`id`),
    KEY          `user_id` (`user_id`,`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;