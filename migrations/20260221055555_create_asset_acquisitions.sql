-- +goose Up
-- +goose StatementBegin
CREATE TABLE asset_acquisitions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    document_number VARCHAR(100) NOT NULL UNIQUE,
    transaction_date DATE NOT NULL,
    asset_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT 'NULL for new asset, filled after asset created',
    asset_number VARCHAR(100),
    asset_name VARCHAR(255) NOT NULL,
    acquisition_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    category_id BIGINT UNSIGNED NULL DEFAULT NULL,
    branch_code VARCHAR(50),
    location VARCHAR(255),
    io_number VARCHAR(100),
    notes TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT' COMMENT 'DRAFT, APPROVED, REJECTED',
    created_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMP NULL DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_asset_acq_asset_id (asset_id),
    KEY idx_asset_acq_status (status),
    KEY idx_asset_acq_date (transaction_date),
    KEY idx_asset_acq_branch_code (branch_code),
    CONSTRAINT fk_asset_acq_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE SET NULL,
    CONSTRAINT fk_asset_acq_category FOREIGN KEY (category_id) REFERENCES asset_categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Asset procurement/acquisition transactions';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS asset_acquisitions;
-- +goose StatementEnd
