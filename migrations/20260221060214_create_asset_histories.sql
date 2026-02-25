-- +goose Up
-- +goose StatementBegin
CREATE TABLE asset_histories (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    asset_id BIGINT UNSIGNED NOT NULL,
    transaction_type VARCHAR(50) NOT NULL COMMENT 'ACQUISITION, MUTATION, DISPOSAL, STOCK_OPNAME, DEPRECIATION, VALUE_UPDATE',
    transaction_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT 'Polymorphic - references id from transaction tables',
    document_number VARCHAR(100),
    transaction_date DATE NULL DEFAULT NULL,
    before_data JSON,
    after_data JSON,
    changed_by VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_asset_hist_asset_id (asset_id),
    KEY idx_asset_hist_type (transaction_type),
    KEY idx_asset_hist_date (transaction_date),
    KEY idx_asset_hist_created_at (created_at),
    CONSTRAINT fk_asset_hist_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Audit trail for all asset changes';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS asset_histories;
-- +goose StatementEnd