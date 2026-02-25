-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_stock_opnames (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    physical_status VARCHAR(50) COMMENT 'EXISTS, MISSING, DAMAGED, OBSOLETE',
    `condition` VARCHAR(50) COMMENT 'GOOD, FAIR, POOR, BROKEN',
    asset_status VARCHAR(50) COMMENT 'ACTIVE, INACTIVE, MAINTENANCE, RETIRED, DISPOSED',
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_trans_so_transaction_id (transaction_id),
    KEY idx_trans_so_transaction_number (transaction_number),
    KEY idx_trans_so_asset_id (asset_id),
    CONSTRAINT fk_trans_so_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_trans_so_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Stock opname transaction detail per asset';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_stock_opnames;
-- +goose StatementEnd