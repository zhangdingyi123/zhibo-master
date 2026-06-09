-- 售后异常：退款 / 取消原因
ALTER TABLE orders
    MODIFY status ENUM(
        'pending_pay', 'paid', 'shipped', 'completed',
        'cancelled', 'closed', 'refunded'
    ) NOT NULL DEFAULT 'pending_pay',
    ADD COLUMN cancel_reason VARCHAR(256) NULL COMMENT '取消/退款原因' AFTER completed_at,
    ADD COLUMN cancelled_by ENUM('buyer', 'seller', 'system') NULL COMMENT '关单操作方' AFTER cancel_reason,
    ADD COLUMN cancelled_at DATETIME(3) NULL COMMENT '取消时间' AFTER cancelled_by,
    ADD COLUMN refunded_at DATETIME(3) NULL COMMENT '退款时间' AFTER cancelled_at;
