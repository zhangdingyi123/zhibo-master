-- 订单履约：收货地址 + 发货 + 确认收货
ALTER TABLE orders
    MODIFY status ENUM('pending_pay', 'paid', 'shipped', 'completed', 'cancelled', 'closed')
        NOT NULL DEFAULT 'pending_pay',
    ADD COLUMN receiver_name VARCHAR(64) NULL COMMENT '收货人' AFTER paid_at,
    ADD COLUMN receiver_phone VARCHAR(20) NULL COMMENT '收货手机' AFTER receiver_name,
    ADD COLUMN receiver_address VARCHAR(512) NULL COMMENT '收货地址' AFTER receiver_phone,
    ADD COLUMN tracking_no VARCHAR(64) NULL COMMENT '物流单号' AFTER receiver_address,
    ADD COLUMN shipped_at DATETIME(3) NULL AFTER tracking_no,
    ADD COLUMN completed_at DATETIME(3) NULL AFTER shipped_at;
