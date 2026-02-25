-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_mutations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    from_branch_code VARCHAR(50),
    to_branch_code VARCHAR(50),
    from_location VARCHAR(255),
    to_location VARCHAR(255),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_trans_mut_transaction_id (transaction_id),
    KEY idx_trans_mut_transaction_number (transaction_number),
    KEY idx_trans_mut_asset_id (asset_id),
    KEY idx_trans_mut_from_branch (from_branch_code),
    KEY idx_trans_mut_to_branch (to_branch_code),
    CONSTRAINT fk_trans_mut_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_trans_mut_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Mutation/transfer transaction detail per asset';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_mutations;
-- +goose StatementEnd