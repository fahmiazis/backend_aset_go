-- +goose Up
-- +goose StatementBegin
CREATE TABLE asset_stock_opnames (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    stock_opname_id BIGINT UNSIGNED NOT NULL,
    asset_id BIGINT UNSIGNED NOT NULL,
    asset_number VARCHAR(100) NOT NULL,
    asset_name VARCHAR(255) NOT NULL,
    description TEXT,
    brand VARCHAR(100),
    unit_of_measure VARCHAR(50),
    unit_quantity DECIMAL(15,2),
    `condition` VARCHAR(50) COMMENT 'GOOD, FAIR, POOR, BROKEN',
    physical_status VARCHAR(50) COMMENT 'EXISTS, MISSING, DAMAGED, OBSOLETE',
    asset_status VARCHAR(50) COMMENT 'ACTIVE, INACTIVE, MAINTENANCE, RETIRED, DISPOSED',
    location VARCHAR(255),
    grouping VARCHAR(100),
    notes TEXT,
    book_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    acquisition_value DECIMAL(18,2) NOT NULL DEFAULT 0,
    accumulated_depreciation DECIMAL(18,2) NOT NULL DEFAULT 0,
    category_id BIGINT UNSIGNED NULL DEFAULT NULL,
    branch_code VARCHAR(50),
    io_number VARCHAR(100),
    record_type VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_asset_so_unique_asset (stock_opname_id, asset_id),
    KEY idx_asset_so_stock_opname_id (stock_opname_id),
    KEY idx_asset_so_asset_id (asset_id),
    KEY idx_asset_so_asset_number (asset_number),
    KEY idx_asset_so_branch_code (branch_code),
    CONSTRAINT fk_asset_so_stock_opname FOREIGN KEY (stock_opname_id) REFERENCES stock_opnames(id) ON DELETE CASCADE,
    CONSTRAINT fk_asset_so_asset FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT,
    CONSTRAINT fk_asset_so_category FOREIGN KEY (category_id) REFERENCES asset_categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Stock opname snapshot - locked asset data at SO date';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS asset_stock_opnames;
-- +goose StatementEnd