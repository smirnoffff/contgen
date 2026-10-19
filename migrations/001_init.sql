CREATE TYPE job_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED');
CREATE TYPE user_role AS ENUM ('ADMIN', 'USER');
CREATE TYPE app_language AS ENUM ('EN', 'RU');
CREATE TYPE payment_method AS ENUM ('STARS', 'ADS', 'PROMO');
CREATE TYPE order_status AS ENUM ('CART', 'PAID', 'CANCELLED');
CREATE TYPE render_theme AS ENUM ('DARK_GLASS', 'CORPORATE_LIGHT', 'CYBER_HUD', 'VINTAGE_PRINT');
CREATE TYPE payment_status AS ENUM ('PENDING', 'SUCCEEDED', 'FAILED', 'REFUNDED');


CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    tg_id BIGINT NOT NULL UNIQUE,
    role user_role NOT NULL DEFAULT 'USER',
    username VARCHAR(255),
    first_name VARCHAR(255) NOT NULL,
    language app_language NOT NULL DEFAULT 'RU',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()

);

CREATE INDEX idx_users_username ON users(username) WHERE username IS NOT NULL;


CREATE TABLE music_tracks (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(250) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    duration_sec INT CHECK (duration_sec IS NULL OR duration_sec > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_music_tracks_user ON music_tracks(user_id);


CREATE TABLE datasets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(250) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_datasets_user ON datasets(user_id);

CREATE TABLE data_points (
    id BIGSERIAL PRIMARY KEY,
    dataset_id BIGINT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    entity_name VARCHAR(250) NOT NULL,
    value NUMERIC(15,2) NOT NULL,
    recorded_at INT NOT NULL
);

CREATE INDEX idx_dataset_time ON data_points(dataset_id, recorded_at);


CREATE TABLE promocodes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL UNIQUE,
    discount_percent INT NOT NULL CHECK (discount_percent BETWEEN 1 AND 100),
    valid_until TIMESTAMPTZ,
    max_uses INT CHECK (max_uses IS NULL OR max_uses > 0),
    used_count INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    promocode_id BIGINT REFERENCES promocodes(id) ON DELETE SET NULL,
    discount_percent_applied INT CHECK (discount_percent_applied IS NULL
        OR discount_percent_applied BETWEEN 1 AND 100),
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    payment_method payment_method,
    total_amount NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    status order_status NOT NULL DEFAULT 'CART',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_promocode ON orders(promocode_id);

CREATE UNIQUE INDEX idx_orders_single_cart ON orders(user_id) WHERE status = 'CART';

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_orders_updated_at
BEFORE UPDATE ON orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE promocode_usages (
    id BIGSERIAL PRIMARY KEY,
    promocode_id BIGINT NOT NULL REFERENCES promocodes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (promocode_id, user_id)
);


CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    method payment_method NOT NULL,
    external_id VARCHAR(255),
    amount NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (amount >= 0),
    status payment_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_order ON payments(order_id);


CREATE TABLE render_jobs (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    dataset_id BIGINT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    music_track_id BIGINT REFERENCES music_tracks(id) ON DELETE SET NULL,
    theme render_theme NOT NULL DEFAULT 'DARK_GLASS',
    status job_status NOT NULL DEFAULT 'PENDING',
    telegram_file_id VARCHAR(255),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CONSTRAINT chk_completed_consistency CHECK (
        status <> 'COMPLETED' OR completed_at IS NOT NULL
    ),
    CONSTRAINT chk_failed_consistency CHECK (
        status <> 'FAILED' OR error_message IS NOT NULL
    )
);

CREATE INDEX idx_render_jobs_order ON render_jobs(order_id);
CREATE INDEX idx_render_jobs_dataset ON render_jobs(dataset_id);
CREATE INDEX idx_render_jobs_music_track ON render_jobs(music_track_id);
