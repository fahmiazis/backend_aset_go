-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_procurements (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    item_name VARCHAR(255) NOT NULL,
    category_id BIGINT UNSIGNED NULL DEFAULT NULL,
    quantity INT NOT NULL COMMENT 'Total quantity for this item (sum of all branches)',
    unit_price DECIMAL(18,2) NOT NULL DEFAULT 0,
    total_price DECIMAL(18,2) NOT NULL DEFAULT 0,
    branch_code VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_trans_proc_transaction_id (transaction_id),
    KEY idx_trans_proc_transaction_number (transaction_number),
    KEY idx_trans_proc_category_id (category_id),
    KEY idx_trans_proc_branch_code (branch_code),
    CONSTRAINT fk_trans_proc_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_trans_proc_category FOREIGN KEY (category_id) REFERENCES asset_categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Procurement transaction detail - main items';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_procurements;
-- +goose StatementEnd