-- name: GetValidPromocode :one
SELECT * FROM promocodes
WHERE code = $1 AND is_active = TRUE
  AND (valid_until IS NULL OR valid_until > NOW())
  AND (max_uses IS NULL OR used_count < max_uses);

-- name: SetOrderPromocode :exec
UPDATE orders SET promocode_id = $2, discount_percent_applied = $3
WHERE id = $1;

-- name: IncrementPromocodeUsage :exec
UPDATE promocodes SET used_count = used_count + 1 WHERE id = $1;

-- name: CreatePromocodeUsage :exec
INSERT INTO promocode_usages (promocode_id, user_id, order_id)
VALUES ($1, $2, $3);
