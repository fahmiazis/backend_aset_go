-- +goose Up

-- +goose StatementBegin
CREATE TABLE attachment_configs (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_type VARCHAR(50) NOT NULL    COMMENT 'procurement, mutation, disposal, stock_opname, ALL',
    stage           VARCHAR(50) NOT NULL     COMMENT 'Stage nama atau ALL untuk semua stage',
    branch_code     VARCHAR(50) NOT NULL     COMMENT 'Branch code spesifik atau ALL untuk semua branch',
    attachment_type VARCHAR(100) NOT NULL    COMMENT 'Nama/tipe attachment, misal: SURAT_PENGAJUAN, KTP, dll',
    description     TEXT NULL,
    is_required     TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1 = wajib, 0 = opsional',
    is_active       TINYINT(1) NOT NULL DEFAULT 1,
    created_by      VARCHAR(100) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_att_cfg_transaction_type (transaction_type),
    INDEX idx_att_cfg_stage (stage),
    INDEX idx_att_cfg_branch_code (branch_code),
    UNIQUE KEY uq_attachment_config (transaction_type, stage, branch_code, attachment_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_attachments (
    id                      BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id          BIGINT UNSIGNED NOT NULL,
    transaction_number      VARCHAR(100) NOT NULL,
    transaction_type        VARCHAR(50) NOT NULL,
    stage                   VARCHAR(50) NOT NULL     COMMENT 'Stage saat attachment diupload',
    attachment_config_id    BIGINT UNSIGNED NOT NULL,
    file_name               VARCHAR(255) NOT NULL    COMMENT 'Nama file original',
    file_path               VARCHAR(500) NOT NULL    COMMENT 'Path file di server',
    file_size               BIGINT NULL              COMMENT 'Ukuran file dalam bytes',
    mime_type               VARCHAR(100) NULL,
    status                  ENUM('PENDING', 'APPROVED', 'REJECTED') NOT NULL DEFAULT 'PENDING',
    uploaded_by             VARCHAR(100) NOT NULL    COMMENT 'UUID user yang upload',
    uploaded_at             DATETIME(3) NOT NULL,
    reviewed_by             VARCHAR(100) NULL        COMMENT 'UUID user yang review',
    reviewed_at             DATETIME(3) NULL,
    rejection_reason        TEXT NULL,
    created_at              DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at              DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_trx_att_transaction_id (transaction_id),
    INDEX idx_trx_att_transaction_number (transaction_number),
    INDEX idx_trx_att_stage (stage),
    INDEX idx_trx_att_status (status),
    INDEX idx_trx_att_config_id (attachment_config_id),
    INDEX idx_trx_att_uploaded_by (uploaded_by),

    CONSTRAINT fk_trx_att_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_trx_att_config
        FOREIGN KEY (attachment_config_id) REFERENCES attachment_configs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_attachments;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS attachment_configs;
-- +goose StatementEnd