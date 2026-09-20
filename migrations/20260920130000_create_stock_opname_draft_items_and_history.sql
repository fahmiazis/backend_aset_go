-- +goose Up
-- +goose StatementBegin
CREATE TABLE stock_opname_draft_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    physical_status VARCHAR(50) COMMENT 'EXISTS, MISSING, BORROWED',
    `condition` VARCHAR(50) COMMENT 'GOOD, FAIR, POOR, BROKEN, NOT_APPLICABLE',
    asset_status VARCHAR(50) COMMENT 'ACTIVE, INACTIVE, MAINTENANCE, RETIRED, DISPOSED',
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_so_draft_transaction_id (transaction_id),
    KEY idx_so_draft_transaction_number (transaction_number),
    KEY idx_so_draft_asset_id (asset_id),
    CONSTRAINT fk_so_draft_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_so_draft_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Stock opname item rows while transaction is in DRAFT stage';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE stock_opname_item_history (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    physical_status VARCHAR(50) COMMENT 'EXISTS, MISSING, BORROWED',
    `condition` VARCHAR(50) COMMENT 'GOOD, FAIR, POOR, BROKEN, NOT_APPLICABLE',
    asset_status VARCHAR(50) COMMENT 'ACTIVE, INACTIVE, MAINTENANCE, RETIRED, DISPOSED',
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_so_hist_transaction_id (transaction_id),
    KEY idx_so_hist_transaction_number (transaction_number),
    KEY idx_so_hist_asset_id (asset_id),
    CONSTRAINT fk_so_hist_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_so_hist_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Stock opname item rows after transaction reaches FINISHED';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_item_history;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_draft_items;
-- +goose StatementEnd
