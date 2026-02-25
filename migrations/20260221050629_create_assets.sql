-- +goose Up
-- +goose StatementBegin
CREATE TABLE assets (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    asset_number VARCHAR(100) NOT NULL UNIQUE,
    asset_name VARCHAR(255) NOT NULL,
    description TEXT,
    brand VARCHAR(100),
    unit_of_measure VARCHAR(50),
    unit_quantity DECIMAL(15,2),
    location VARCHAR(255),
    grouping VARCHAR(100),
    category_id BIGINT UNSIGNED NULL DEFAULT NULL,
    branch_code VARCHAR(50),
    io_number VARCHAR(100),
    record_type VARCHAR(50),
    asset_status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE' COMMENT 'ACTIVE, INACTIVE, MAINTENANCE, RETIRED, DISPOSED',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL DEFAULT NULL,
    KEY idx_assets_branch_code (branch_code),
    KEY idx_assets_category_id (category_id),
    KEY idx_assets_status (asset_status),
    KEY idx_assets_deleted_at (deleted_at),
    KEY idx_assets_location (location),
    CONSTRAINT fk_assets_category FOREIGN KEY (category_id) REFERENCES asset_categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Main asset master table';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS assets;
-- +goose StatementEnd