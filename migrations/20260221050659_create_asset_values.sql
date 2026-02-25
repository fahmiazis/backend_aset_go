-- +goose Up
-- +goose StatementBegin
CREATE TABLE asset_values (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    asset_id BIGINT UNSIGNED NOT NULL,
    effective_date DATE NOT NULL,
    book_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    acquisition_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    accumulated_depreciation DECIMAL(18,2) NOT NULL DEFAULT 0,
    `condition` VARCHAR(50) COMMENT 'GOOD, FAIR, POOR, BROKEN',
    physical_status VARCHAR(50) COMMENT 'EXISTS, MISSING, DAMAGED, OBSOLETE',
    asset_status VARCHAR(50) COMMENT 'ACTIVE, INACTIVE, MAINTENANCE, RETIRED, DISPOSED',
    is_active TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Only one active record per asset allowed',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_asset_values_asset_id (asset_id),
    KEY idx_asset_values_active (asset_id, is_active),
    KEY idx_asset_values_effective_date (effective_date),
    CONSTRAINT fk_asset_values_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Dynamic asset values - tracks changes over time';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS asset_values;
-- +goose StatementEnd
