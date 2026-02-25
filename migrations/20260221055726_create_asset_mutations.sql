-- +goose Up
-- +goose StatementBegin
CREATE TABLE asset_mutations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    document_number VARCHAR(100) NOT NULL UNIQUE,
    transaction_date DATE NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    from_branch_code VARCHAR(50),
    to_branch_code VARCHAR(50),
    from_location VARCHAR(255),
    to_location VARCHAR(255),
    notes TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT' COMMENT 'DRAFT, APPROVED, REJECTED',
    created_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMP NULL DEFAULT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_asset_mut_asset_id (asset_id),
    KEY idx_asset_mut_status (status),
    KEY idx_asset_mut_date (transaction_date),
    KEY idx_asset_mut_from_branch (from_branch_code),
    KEY idx_asset_mut_to_branch (to_branch_code),
    CONSTRAINT fk_asset_mut_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Asset transfer/mutation transactions';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS asset_mutations;
-- +goose StatementEnd