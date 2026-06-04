-- 直播竞拍系统 — 库表 DDL（阶段 1）
-- 金额单位：分（BIGINT），避免浮点误差

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ---------------------------------------------------------------------------
-- 用户
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    open_id       VARCHAR(64)     NOT NULL COMMENT 'Mock 登录标识，唯一',
    nickname      VARCHAR(64)     NOT NULL,
    avatar        VARCHAR(512)    NOT NULL DEFAULT '',
    role          ENUM('buyer', 'anchor', 'admin') NOT NULL DEFAULT 'buyer',
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_open_id (open_id),
    KEY idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户';

-- ---------------------------------------------------------------------------
-- 商品（主播上架）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS products (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    anchor_id     BIGINT UNSIGNED NOT NULL COMMENT '所属主播',
    name          VARCHAR(128)    NOT NULL,
    description   TEXT,
    cover_url     VARCHAR(512)    NOT NULL DEFAULT '',
    images        JSON            NULL COMMENT '多图 URL 数组',
    status        ENUM('draft', 'listed', 'auctioning', 'sold', 'off_shelf')
                  NOT NULL DEFAULT 'draft',
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_anchor_status (anchor_id, status),
    CONSTRAINT fk_products_anchor FOREIGN KEY (anchor_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品';

-- ---------------------------------------------------------------------------
-- 竞拍场次（含规则字段）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS auction_sessions (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    product_id            BIGINT UNSIGNED NOT NULL,
    anchor_id             BIGINT UNSIGNED NOT NULL,
    room_id               VARCHAR(64)     NOT NULL COMMENT 'WebSocket 房间 ID',
    status                ENUM('pending', 'running', 'settled', 'cancelled', 'failed')
                          NOT NULL DEFAULT 'pending',

    -- 规则（未开始可改）
    starting_price        BIGINT          NOT NULL DEFAULT 0 COMMENT '起拍价，分；支持 0 元起拍',
    bid_increment         BIGINT          NOT NULL COMMENT '加价幅度，分',
    cap_price             BIGINT          NULL COMMENT '封顶价，分；NULL 表示无封顶',
    duration_sec          INT UNSIGNED    NOT NULL COMMENT '竞拍基础时长（秒）',
    extend_threshold_sec  INT UNSIGNED    NOT NULL DEFAULT 10 COMMENT '结束前 N 秒内有出价则触发延时',
    extend_sec            INT UNSIGNED    NOT NULL DEFAULT 30 COMMENT '单次延时秒数，建议 10–30',

    -- 运行时快照（权威数据以 Redis + 落库为准）
    current_price         BIGINT          NOT NULL DEFAULT 0 COMMENT '当前最高价，分',
    bid_count             INT UNSIGNED    NOT NULL DEFAULT 0,
    participant_count     INT UNSIGNED    NOT NULL DEFAULT 0,
    winner_id             BIGINT UNSIGNED NULL,
    version               INT UNSIGNED    NOT NULL DEFAULT 0 COMMENT '乐观锁版本号',

    scheduled_start_at    DATETIME(3)     NULL COMMENT '计划开拍时间',
    started_at            DATETIME(3)     NULL COMMENT '实际开始时间',
    end_at                DATETIME(3)     NULL COMMENT '权威结束时间（含延时）',
    settled_at            DATETIME(3)     NULL,
    cancel_reason         VARCHAR(255)    NULL,

    created_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uk_room_id (room_id),
    KEY idx_product (product_id),
    KEY idx_anchor_status (anchor_id, status),
    KEY idx_status_end (status, end_at),
    CONSTRAINT fk_sessions_product FOREIGN KEY (product_id) REFERENCES products (id),
    CONSTRAINT fk_sessions_anchor FOREIGN KEY (anchor_id) REFERENCES users (id),
    CONSTRAINT fk_sessions_winner FOREIGN KEY (winner_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='竞拍场次';

-- ---------------------------------------------------------------------------
-- 出价记录
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS bids (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    session_id    BIGINT UNSIGNED NOT NULL,
    user_id       BIGINT UNSIGNED NOT NULL,
    amount        BIGINT          NOT NULL COMMENT '出价金额，分',
    request_id    VARCHAR(64)     NOT NULL COMMENT '客户端幂等键',
    seq           INT UNSIGNED    NOT NULL COMMENT '场次内递增序号',
    is_winning    TINYINT(1)      NOT NULL DEFAULT 0 COMMENT '是否为当前最高价',
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_session_request (session_id, request_id),
    UNIQUE KEY uk_session_seq (session_id, seq),
    KEY idx_session_amount (session_id, amount DESC),
    KEY idx_user_session (user_id, session_id),
    CONSTRAINT fk_bids_session FOREIGN KEY (session_id) REFERENCES auction_sessions (id),
    CONSTRAINT fk_bids_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='出价';

-- ---------------------------------------------------------------------------
-- 订单（成交后生成）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS orders (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    order_no      VARCHAR(32)     NOT NULL COMMENT '业务订单号',
    session_id    BIGINT UNSIGNED NOT NULL,
    product_id    BIGINT UNSIGNED NOT NULL,
    buyer_id      BIGINT UNSIGNED NOT NULL,
    seller_id     BIGINT UNSIGNED NOT NULL COMMENT '主播/商家',
    amount        BIGINT          NOT NULL COMMENT '成交金额，分',
    status        ENUM('pending_pay', 'paid', 'cancelled', 'closed')
                  NOT NULL DEFAULT 'pending_pay',
    paid_at       DATETIME(3)     NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_order_no (order_no),
    UNIQUE KEY uk_session (session_id),
    KEY idx_buyer (buyer_id, created_at DESC),
    KEY idx_seller (seller_id, created_at DESC),
    CONSTRAINT fk_orders_session FOREIGN KEY (session_id) REFERENCES auction_sessions (id),
    CONSTRAINT fk_orders_product FOREIGN KEY (product_id) REFERENCES products (id),
    CONSTRAINT fk_orders_buyer FOREIGN KEY (buyer_id) REFERENCES users (id),
    CONSTRAINT fk_orders_seller FOREIGN KEY (seller_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单';

SET FOREIGN_KEY_CHECKS = 1;
