-- 登录注册：手机号 + 密码（bcrypt）
-- 演示账号默认密码：123456

SET NAMES utf8mb4;

ALTER TABLE users
    ADD COLUMN phone VARCHAR(20) NULL COMMENT '手机号，唯一' AFTER open_id,
    ADD COLUMN password_hash VARCHAR(255) NULL COMMENT 'bcrypt' AFTER phone;

ALTER TABLE users ADD UNIQUE KEY uk_phone (phone);

-- 种子账号绑定手机号（密码均为 123456）
UPDATE users SET phone = '13800000001', password_hash = '$2a$10$walqwk85vHj3N5FIkIhqkObNJgh7T5jpUrH5tdS02bXAb/uZvspty' WHERE open_id = 'anchor_001';
UPDATE users SET phone = '13800000002', password_hash = '$2a$10$walqwk85vHj3N5FIkIhqkObNJgh7T5jpUrH5tdS02bXAb/uZvspty' WHERE open_id = 'buyer_001';
UPDATE users SET phone = '13800000003', password_hash = '$2a$10$walqwk85vHj3N5FIkIhqkObNJgh7T5jpUrH5tdS02bXAb/uZvspty' WHERE open_id = 'buyer_002';
UPDATE users SET phone = '13800000004', password_hash = '$2a$10$walqwk85vHj3N5FIkIhqkObNJgh7T5jpUrH5tdS02bXAb/uZvspty' WHERE open_id = 'buyer_003';
UPDATE users SET phone = '13800000005', password_hash = '$2a$10$walqwk85vHj3N5FIkIhqkObNJgh7T5jpUrH5tdS02bXAb/uZvspty' WHERE open_id = 'admin_001';
