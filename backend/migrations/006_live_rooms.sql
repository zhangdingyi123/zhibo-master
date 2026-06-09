-- 多商品连拍：直播房间 + 场次队列
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS live_rooms (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    anchor_id           BIGINT UNSIGNED NOT NULL,
    title               VARCHAR(128)    NOT NULL DEFAULT '',
    room_id             VARCHAR(64)     NOT NULL COMMENT '稳定 WebSocket 房间 ID',
    status              ENUM('idle', 'live', 'ended') NOT NULL DEFAULT 'idle',
    current_session_id  BIGINT UNSIGNED NULL COMMENT '当前进行中场次',
    created_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_live_room_id (room_id),
    KEY idx_anchor_status (anchor_id, status),
    CONSTRAINT fk_live_rooms_anchor FOREIGN KEY (anchor_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播房间（一场多 SKU）';

ALTER TABLE auction_sessions
    ADD COLUMN live_room_id BIGINT UNSIGNED NULL COMMENT '所属直播房间' AFTER anchor_id,
    ADD COLUMN seq_in_room  INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '连拍序号' AFTER live_room_id;

ALTER TABLE auction_sessions
    DROP INDEX uk_room_id,
    ADD KEY idx_room_status (room_id, status),
    ADD KEY idx_live_room_seq (live_room_id, seq_in_room);

ALTER TABLE auction_sessions
    ADD CONSTRAINT fk_sessions_live_room FOREIGN KEY (live_room_id) REFERENCES live_rooms (id);

ALTER TABLE live_rooms
    ADD CONSTRAINT fk_live_rooms_current_session FOREIGN KEY (current_session_id) REFERENCES auction_sessions (id);
