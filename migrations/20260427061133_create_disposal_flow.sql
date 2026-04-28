-- +goose Up

-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN disposal_type VARCHAR(20) NULL
        COMMENT 'DISPOSE atau SELL'
        AFTER mutation_to_branch_code,
    ADD COLUMN sale_value DECIMAL(18,2) NULL
        COMMENT 'Nilai jual asset (khusus SELL)'
        AFTER disposal_type,
    ADD COLUMN approval_request_number VARCHAR(100) NULL
        COMMENT 'Nomor transaksi approval pengajuan disposal'
        AFTER sale_value,
    ADD COLUMN approval_agreement_number VARCHAR(100) NULL
        COMMENT 'Nomor transaksi approval persetujuan disposal'
        AFTER approval_request_number,
    ADD INDEX idx_transactions_disposal_type (disposal_type);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_disposal_assets (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id      BIGINT UNSIGNED NOT NULL,
    transaction_number  VARCHAR(100) NOT NULL,
    asset_id            BIGINT UNSIGNED NOT NULL,
    asset_number        VARCHAR(100) NOT NULL,
    disposal_type       VARCHAR(20) NOT NULL    COMMENT 'DISPOSE atau SELL',
    disposal_reason     TEXT NULL,
    sale_value          DECIMAL(18,2) NULL      COMMENT 'Nilai jual per asset (khusus SELL, diisi oleh purchasing)',
    document_number     VARCHAR(50) NULL        COMMENT 'Generated saat asset deletion',
    notes               TEXT NULL,
    status              ENUM('PENDING','DELETED','CANCELLED') NOT NULL DEFAULT 'PENDING',
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_disposal_asset (transaction_id, asset_id),
    INDEX idx_disp_asset_transaction_id (transaction_id),
    INDEX idx_disp_asset_asset_id (asset_id),
    INDEX idx_disp_asset_status (status),

    CONSTRAINT fk_disp_asset_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_disp_asset_asset
        FOREIGN KEY (asset_id) REFERENCES assets(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_disposal_attachments (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id              BIGINT UNSIGNED NOT NULL,
    transaction_number          VARCHAR(100) NOT NULL,
    transaction_disposal_asset_id BIGINT UNSIGNED NOT NULL,
    asset_id                    BIGINT UNSIGNED NOT NULL,
    asset_number                VARCHAR(100) NOT NULL,
    attachment_config_id        BIGINT UNSIGNED NOT NULL,
    stage                       VARCHAR(50) NOT NULL    COMMENT 'Stage saat attachment diupload',
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
    INDEX idx_disp_att_transaction_id (transaction_id),
    INDEX idx_disp_att_transaction_number (transaction_number),
    INDEX idx_disp_att_disposal_asset_id (transaction_disposal_asset_id),
    INDEX idx_disp_att_asset_id (asset_id),
    INDEX idx_disp_att_stage (stage),
    INDEX idx_disp_att_status (status),

    CONSTRAINT fk_disp_att_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_disp_att_disposal_asset
        FOREIGN KEY (transaction_disposal_asset_id) REFERENCES transaction_disposal_assets(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_disp_att_config
        FOREIGN KEY (attachment_config_id) REFERENCES attachment_configs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_disposal_attachments;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_disposal_assets;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    DROP INDEX idx_transactions_disposal_type,
    DROP COLUMN approval_agreement_number,
    DROP COLUMN approval_request_number,
    DROP COLUMN sale_value,
    DROP COLUMN disposal_type;
-- +goose StatementEnd