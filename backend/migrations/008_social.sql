-- 直播间社交互动：评论、点赞、收藏、关注
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS room_comments (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    room_id     VARCHAR(64)     NOT NULL,
    user_id     BIGINT UNSIGNED NOT NULL,
    content     VARCHAR(200)    NOT NULL,
    is_hidden   TINYINT(1)      NOT NULL DEFAULT 0,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_room_created (room_id, created_at),
    KEY idx_user (user_id),
    CONSTRAINT fk_room_comments_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播间评论';

CREATE TABLE IF NOT EXISTS room_likes (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    room_id     VARCHAR(64)     NOT NULL,
    user_id     BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_room (room_id),
    KEY idx_user_room (user_id, room_id),
    CONSTRAINT fk_room_likes_user FOREIGN KEY (user_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='直播间点赞';

CREATE TABLE IF NOT EXISTS product_favorites (
    user_id     BIGINT UNSIGNED NOT NULL,
    product_id  BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (user_id, product_id),
    KEY idx_product (product_id),
    CONSTRAINT fk_fav_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_fav_product FOREIGN KEY (product_id) REFERENCES products (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品收藏';

CREATE TABLE IF NOT EXISTS anchor_follows (
    user_id     BIGINT UNSIGNED NOT NULL,
    anchor_id   BIGINT UNSIGNED NOT NULL,
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (user_id, anchor_id),
    KEY idx_anchor (anchor_id),
    CONSTRAINT fk_follow_user FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT fk_follow_anchor FOREIGN KEY (anchor_id) REFERENCES users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关注主播';
