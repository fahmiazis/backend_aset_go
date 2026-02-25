-- +goose Up
-- +goose StatementBegin
CREATE TABLE depreciation_settings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    setting_type VARCHAR(50) NOT NULL COMMENT 'CATEGORY or ASSET',
    reference_id BIGINT UNSIGNED NULL DEFAULT NULL COMMENT 'category_id or asset_id depending on setting_type',
    reference_value VARCHAR(255) COMMENT 'category_code or asset_number for easy reference',
    calculation_method VARCHAR(50) NOT NULL COMMENT 'STRAIGHT_LINE, DECLINING_BALANCE, etc',
    depreciation_period VARCHAR(50) NOT NULL COMMENT 'MONTHLY or DAILY',
    useful_life_months INT NOT NULL,
    depreciation_rate DECIMAL(10,4),
    start_date DATE NOT NULL,
    end_date DATE NULL DEFAULT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_dep_settings_type (setting_type),
    KEY idx_dep_settings_reference (setting_type, reference_id),
    KEY idx_dep_settings_active (is_active),
    KEY idx_dep_settings_dates (start_date, end_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Depreciation calculation settings per category or asset';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS depreciation_settings;
-- +goose StatementEnd