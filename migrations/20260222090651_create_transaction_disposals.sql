-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_disposals (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    transaction_id BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    disposal_method VARCHAR(50) COMMENT 'SALE, SCRAP, DONATE, WRITE_OFF',
    disposal_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    disposal_reason TEXT,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_trans_disp_transaction_id (transaction_id),
    KEY idx_trans_disp_transaction_number (transaction_number),
    KEY idx_trans_disp_asset_id (asset_id),
    KEY idx_trans_disp_method (disposal_method),
    CONSTRAINT fk_trans_disp_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_trans_disp_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Disposal transaction detail per asset';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_disposals;
-- +goose StatementEnd