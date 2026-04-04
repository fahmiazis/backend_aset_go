-- +goose Up

-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN mutation_category_id BIGINT UNSIGNED NULL
        COMMENT 'Category ID yang dipakai di mutasi — semua asset harus sama category'
        AFTER io_number,
    ADD COLUMN mutation_to_branch_code VARCHAR(50) NULL
        COMMENT 'Branch tujuan mutasi — semua asset harus sama tujuan'
        AFTER mutation_category_id,
    ADD INDEX idx_transactions_mutation_category (mutation_category_id),
    ADD INDEX idx_transactions_mutation_branch (mutation_to_branch_code);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_mutation_assets (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id      BIGINT UNSIGNED NOT NULL,
    transaction_number  VARCHAR(100) NOT NULL,
    asset_id            BIGINT UNSIGNED NOT NULL,
    asset_number        VARCHAR(100) NOT NULL,
    from_branch_code    VARCHAR(50) NOT NULL    COMMENT 'Branch asal asset saat ini',
    to_branch_code      VARCHAR(50) NOT NULL    COMMENT 'Branch tujuan mutasi',
    from_location       VARCHAR(255) NULL,
    to_location         VARCHAR(255) NULL,
    document_number     VARCHAR(50) NULL        COMMENT 'Generated saat eksekusi',
    notes               TEXT NULL,
    status              ENUM('PENDING','EXECUTED','CANCELLED') NOT NULL DEFAULT 'PENDING',
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_mutation_asset (transaction_id, asset_id)
        COMMENT 'Satu asset hanya bisa ada 1x per transaksi mutasi',
    INDEX idx_mut_asset_transaction_id (transaction_id),
    INDEX idx_mut_asset_transaction_number (transaction_number),
    INDEX idx_mut_asset_asset_id (asset_id),
    INDEX idx_mut_asset_status (status),

    CONSTRAINT fk_mut_asset_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_mut_asset_asset
        FOREIGN KEY (asset_id) REFERENCES assets(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_mutation_attachments (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id              BIGINT UNSIGNED NOT NULL,
    transaction_number          VARCHAR(100) NOT NULL,
    transaction_mutation_asset_id BIGINT UNSIGNED NOT NULL COMMENT 'Link ke asset spesifik di mutasi',
    asset_id                    BIGINT UNSIGNED NOT NULL,
    asset_number                VARCHAR(100) NOT NULL,
    attachment_config_id        BIGINT UNSIGNED NOT NULL,
    file_name                   VARCHAR(255) NOT NULL,
    file_path                   VARCHAR(500) NOT NULL,
    file_size                   BIGINT NULL,
    mime_type                   VARCHAR(100) NULL,
    status                      ENUM('PENDING','APPROVED','REJECTED') NOT NULL DEFAULT 'PENDING',
    uploaded_by                 VARCHAR(100) NOT NULL,
    uploaded_at                 DATETIME(3) NOT NULL,
    reviewed_by                 VARCHAR(100) NULL,
    reviewed_at                 DATETIME(3) NULL,
    rejection_reason            TEXT NULL,
    created_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_mut_att_transaction_id (transaction_id),
    INDEX idx_mut_att_transaction_number (transaction_number),
    INDEX idx_mut_att_mutation_asset_id (transaction_mutation_asset_id),
    INDEX idx_mut_att_asset_id (asset_id),
    INDEX idx_mut_att_status (status),

    CONSTRAINT fk_mut_att_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_mut_att_mutation_asset
        FOREIGN KEY (transaction_mutation_asset_id) REFERENCES transaction_mutation_assets(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_mut_att_config
        FOREIGN KEY (attachment_config_id) REFERENCES attachment_configs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_mutation_attachments;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_mutation_assets;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    DROP INDEX idx_transactions_mutation_branch,
    DROP INDEX idx_transactions_mutation_category,
    DROP COLUMN mutation_to_branch_code,
    DROP COLUMN mutation_category_id;
-- +goose StatementEnd