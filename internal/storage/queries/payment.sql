-- name: GetPendingPaymentForOrder :one
SELECT * FROM payments
WHERE order_id = $1 AND status = 'PENDING'
ORDER BY id DESC LIMIT 1;

-- name: GetPaymentByExternalID :one
SELECT * FROM payments
WHERE external_id = $1
ORDER BY id DESC LIMIT 1
FOR UPDATE;

-- name: CreatePayment :one
INSERT INTO payments (order_id, method, external_id, amount, status)
VALUES ($1, $2, $3, $4, 'PENDING')
RETURNING *;

-- name: MarkPaymentSucceeded :execrows
UPDATE payments SET status = 'SUCCEEDED', external_id = $2
WHERE id = $1 AND status = 'PENDING';

-- name: MarkPaymentFailed :execrows
UPDATE payments SET status = 'FAILED'
WHERE id = $1 AND status = 'PENDING';

-- name: LockOrderByID :one
SELECT * FROM orders WHERE id = $1 FOR UPDATE;

-- name: SetOrderCheckoutTotals :execrows
UPDATE orders SET promocode_id = $2, discount_percent_applied = $3,
 discount_amount = $4, total_amount = $5
WHERE id = $1 AND user_id = $6 AND status = 'CART';

-- name: SetOrderPaidByPayment :execrows
UPDATE orders SET status = 'PAID', payment_method = $2
WHERE id = $1 AND status = 'CART';

-- name: LockValidPromocode :one
SELECT * FROM promocodes
WHERE code = $1 AND is_active = TRUE
  AND (valid_until IS NULL OR valid_until > NOW())
  AND (max_uses IS NULL OR used_count < max_uses)
FOR UPDATE;

-- name: HasUsedPromocode :one
SELECT EXISTS (SELECT 1 FROM promocode_usages WHERE promocode_id = $1 AND user_id = $2);

-- name: ConsumePromocode :execrows
UPDATE promocodes SET used_count = used_count + 1
WHERE id = $1 AND is_active = TRUE
  AND (valid_until IS NULL OR valid_until > NOW())
  AND (max_uses IS NULL OR used_count < max_uses);

-- name: GetPaymentByChargeID :one
SELECT * FROM payments
WHERE external_id = $1 AND status = 'SUCCEEDED'
ORDER BY id DESC LIMIT 1;

-- name: GetPaymentByID :one
SELECT * FROM payments WHERE id = $1 FOR UPDATE;

-- name: DeletePromocodeUsage :execrows
DELETE FROM promocode_usages WHERE promocode_id = $1 AND order_id = $2;

-- name: ReleasePromocode :execrows
UPDATE promocodes SET used_count = GREATEST(used_count - 1, 0) WHERE id = $1;

-- name: ResetOrderCheckoutTotals :execrows
UPDATE orders SET promocode_id = NULL, discount_percent_applied = NULL,
 discount_amount = 0, total_amount = 0
WHERE id = $1 AND status = 'CART';
