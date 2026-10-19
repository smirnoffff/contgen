-- name: GetCart :one
SELECT * FROM orders WHERE user_id = $1 AND status = 'CART';

-- name: SetOrderPaid :execrows
UPDATE orders SET status = 'PAID', payment_method = $2, total_amount = $3
WHERE id = $1 AND status = 'CART';

-- name: GetOrderByID :one
SELECT * FROM orders WHERE id = $1;
