-- 订单支付超时：pending_pay 到期自动关单
ALTER TABLE orders
    ADD COLUMN pay_expire_at DATETIME(3) NULL COMMENT '待支付截止时间' AFTER status,
    ADD KEY idx_pending_expire (status, pay_expire_at);

-- 已有待支付订单补 30 分钟窗口（从创建时间起算）
UPDATE orders
SET pay_expire_at = DATE_ADD(created_at, INTERVAL 30 MINUTE)
WHERE status = 'pending_pay' AND pay_expire_at IS NULL;
