-- +goose Up
-- +goose StatementBegin
CREATE TABLE monthly_depreciation_calculations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    asset_id BIGINT UNSIGNED NOT NULL,
    period VARCHAR(7) NOT NULL COMMENT 'Format: YYYY-MM',
    calculation_date DATE NOT NULL,
    beginning_book_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    depreciation_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    beginning_accumulated_depreciation DECIMAL(18,2) NOT NULL DEFAULT 0,
    ending_accumulated_depreciation DECIMAL(18,2) NOT NULL DEFAULT 0,
    ending_book_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    calculation_method VARCHAR(50),
    depreciation_setting_id BIGINT UNSIGNED NULL DEFAULT NULL,
    is_locked TINYINT(1) NOT NULL DEFAULT 0 COMMENT '1 = finalized, cannot be recalculated',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_monthly_dep_asset_period (asset_id, period),
    KEY idx_monthly_dep_asset_id (asset_id),
    KEY idx_monthly_dep_period (period),
    KEY idx_monthly_dep_locked (is_locked),
    CONSTRAINT fk_monthly_dep_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE,
    CONSTRAINT fk_monthly_dep_setting FOREIGN KEY (depreciation_setting_id) REFERENCES depreciation_settings(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Monthly depreciation calculation results';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS monthly_depreciation_calculations;
-- +goose StatementEnd
