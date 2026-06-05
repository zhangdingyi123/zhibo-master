-- 用户消息收件箱（写扩散模型：事件发生时写入各用户信箱）
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS user_messages (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL COMMENT '收件人',
    event_type    VARCHAR(32)     NOT NULL COMMENT 'outbid/extended/settled/cancelled 等',
    category      ENUM('auction', 'order', 'system') NOT NULL DEFAULT 'auction',
    title         VARCHAR(128)    NOT NULL,
    body          VARCHAR(512)    NOT NULL DEFAULT '',
    payload       JSON            NULL COMMENT 'sessionId/roomId/orderId 等上下文',
    is_read       TINYINT(1)      NOT NULL DEFAULT 0,
    dedupe_key    VARCHAR(128)    NULL COMMENT '幂等键，防止重复写扩散',
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_dedupe (user_id, dedupe_key),
    KEY idx_user_inbox (user_id, is_read, created_at DESC),
    CONSTRAINT fk_messages_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户消息收件箱（写扩散）';
