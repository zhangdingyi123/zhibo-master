-- Mock 用户、商品、场次（阶段 1 种子数据）
-- 执行前需先跑 001_schema.sql

SET NAMES utf8mb4;

-- 用户：1 主播 + 3 买家 + 1 管理员
INSERT INTO users (id, open_id, nickname, avatar, role) VALUES
(1, 'anchor_001', '主播小美', 'https://picsum.photos/seed/anchor/200', 'anchor'),
(2, 'buyer_001',  '买家阿强', 'https://picsum.photos/seed/buyer1/200', 'buyer'),
(3, 'buyer_002',  '买家莉莉', 'https://picsum.photos/seed/buyer2/200', 'buyer'),
(4, 'buyer_003',  '买家老王', 'https://picsum.photos/seed/buyer3/200', 'buyer'),
(5, 'admin_001',  '运营管理员', 'https://picsum.photos/seed/admin/200', 'admin')
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname);

-- 商品
INSERT INTO products (id, anchor_id, name, description, cover_url, images, status) VALUES
(1, 1, 'Vintage 机械腕表', '95 新，盒证齐全，直播专拍', 'https://picsum.photos/seed/watch/400',
 JSON_ARRAY('https://picsum.photos/seed/watch/400', 'https://picsum.photos/seed/watch2/400'), 'listed'),
(2, 1, '限定潮玩盲盒', '直播间 0 元起拍福利款', 'https://picsum.photos/seed/toy/400',
 JSON_ARRAY('https://picsum.photos/seed/toy/400'), 'draft'),
(3, 1, '手工皮具钱包', '意大利植鞣革', 'https://picsum.photos/seed/wallet/400',
 JSON_ARRAY('https://picsum.photos/seed/wallet/400'), 'listed')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 场次 1：未开始，0 元起拍，有封顶
INSERT INTO auction_sessions (
    id, product_id, anchor_id, room_id, status,
    starting_price, bid_increment, cap_price,
    duration_sec, extend_threshold_sec, extend_sec,
    current_price, scheduled_start_at
) VALUES (
    1, 1, 1, 'room_sess_1', 'pending',
    0, 1000, 500000,
    120, 10, 30,
    0, DATE_ADD(NOW(3), INTERVAL 1 HOUR)
) ON DUPLICATE KEY UPDATE status = VALUES(status);

-- 场次 2：未开始，常规起拍
INSERT INTO auction_sessions (
    id, product_id, anchor_id, room_id, status,
    starting_price, bid_increment, cap_price,
    duration_sec, extend_threshold_sec, extend_sec,
    current_price, scheduled_start_at
) VALUES (
    2, 3, 1, 'room_sess_2', 'pending',
    9900, 500, NULL,
    180, 10, 20,
    9900, DATE_ADD(NOW(3), INTERVAL 2 HOUR)
) ON DUPLICATE KEY UPDATE status = VALUES(status);

-- 场次 3：已成交样例（历史记录演示）
INSERT INTO auction_sessions (
    id, product_id, anchor_id, room_id, status,
    starting_price, bid_increment, cap_price,
    duration_sec, extend_threshold_sec, extend_sec,
    current_price, bid_count, participant_count, winner_id,
    started_at, end_at, settled_at
) VALUES (
    3, 1, 1, 'room_sess_3', 'settled',
    10000, 1000, NULL,
    60, 10, 30,
    35000, 5, 3, 2,
    DATE_SUB(NOW(3), INTERVAL 2 DAY),
    DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 75 SECOND,
    DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 75 SECOND
) ON DUPLICATE KEY UPDATE status = VALUES(status);

INSERT INTO bids (session_id, user_id, amount, request_id, seq, is_winning, created_at) VALUES
(3, 2, 10000, 'seed-bid-3-1', 1, 0, DATE_SUB(NOW(3), INTERVAL 2 DAY)),
(3, 3, 15000, 'seed-bid-3-2', 2, 0, DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 10 SECOND),
(3, 4, 20000, 'seed-bid-3-3', 3, 0, DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 20 SECOND),
(3, 2, 25000, 'seed-bid-3-4', 4, 0, DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 40 SECOND),
(3, 2, 35000, 'seed-bid-3-5', 5, 1, DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 70 SECOND)
ON DUPLICATE KEY UPDATE amount = VALUES(amount);

INSERT INTO orders (order_no, session_id, product_id, buyer_id, seller_id, amount, status, paid_at) VALUES
('ZB202605240001', 3, 1, 2, 1, 35000, 'paid', DATE_SUB(NOW(3), INTERVAL 2 DAY) + INTERVAL 80 SECOND)
ON DUPLICATE KEY UPDATE status = VALUES(status);
