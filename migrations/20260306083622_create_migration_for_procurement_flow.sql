-- +goose Up

-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN current_stage VARCHAR(50) NOT NULL DEFAULT 'DRAFT'
        COMMENT 'DRAFT|VERIFIKASI_ASET|APPROVAL|PROSES_BUDGET|EKSEKUSI_ASET|GR|SELESAI|REJECTED'
        AFTER status;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN io_number VARCHAR(50) NULL
        COMMENT 'Nomor Investment Order, tergenerate saat PROSES_BUDGET'
        AFTER current_stage;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    ADD INDEX idx_transactions_current_stage (current_stage);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_stages (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id     BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100) NOT NULL,
    from_stage         VARCHAR(50) NULL     COMMENT 'NULL jika stage pertama',
    to_stage           VARCHAR(50) NOT NULL,
    action             VARCHAR(50) NOT NULL COMMENT 'SUBMIT|VERIFY|APPROVE|REJECT|PROCESS_BUDGET|EXECUTE|GR|REVISE',
    actor_id           VARCHAR(100) NOT NULL COMMENT 'UUID user yang melakukan aksi',
    actor_name         VARCHAR(100) NULL,
    notes              TEXT NULL,
    metadata           JSON NULL            COMMENT 'Data tambahan per stage jika diperlukan',
    created_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_transaction_stages_transaction_id (transaction_id),
    INDEX idx_transaction_stages_transaction_number (transaction_number),
    INDEX idx_transaction_stages_to_stage (to_stage),
    INDEX idx_transaction_stages_actor_id (actor_id),

    CONSTRAINT fk_transaction_stages_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transaction_item_verifications (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id              BIGINT UNSIGNED NOT NULL,
    transaction_procurement_id  BIGINT UNSIGNED NOT NULL,
    item_type                   ENUM('ASSET', 'NON_ASSET') NOT NULL
        COMMENT 'Hasil verifikasi PIC Asset',
    is_active                   TINYINT(1) NOT NULL DEFAULT 1
        COMMENT '0 jika item dikeluarkan dari list karena NON_ASSET',
    verified_by                 VARCHAR(100) NOT NULL COMMENT 'UUID PIC Asset',
    verified_at                 DATETIME(3) NOT NULL,
    notes                       TEXT NULL,
    created_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_item_verification (transaction_procurement_id),
    INDEX idx_item_verif_transaction_id (transaction_id),
    INDEX idx_item_verif_procurement_id (transaction_procurement_id),
    INDEX idx_item_verif_item_type (item_type),

    CONSTRAINT fk_item_verif_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_item_verif_procurement
        FOREIGN KEY (transaction_procurement_id) REFERENCES transaction_procurements(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE document_number_sequences (
    id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sequence_type   ENUM('IO', 'ASSET') NOT NULL
        COMMENT 'IO = Investment Order, ASSET = Asset Number',
    reference_code  VARCHAR(50) NOT NULL
        COMMENT 'branch_code untuk IO, category_code untuk ASSET',
    last_sequence   INT UNSIGNED NOT NULL DEFAULT 0
        COMMENT 'Nomor urut terakhir yang sudah dipakai, global tidak reset',
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_sequence (sequence_type, reference_code),
    INDEX idx_seq_type (sequence_type),
    INDEX idx_seq_reference_code (reference_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE asset_gr (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id      BIGINT UNSIGNED NOT NULL,
    transaction_number  VARCHAR(100) NOT NULL,
    asset_id            BIGINT UNSIGNED NOT NULL
        COMMENT 'Asset yang di-GR, sudah tergenerate saat EKSEKUSI_ASET',
    asset_number        VARCHAR(100) NOT NULL,
    branch_code         VARCHAR(50) NOT NULL COMMENT 'Branch tujuan yang melakukan GR',
    gr_date             DATE NOT NULL,
    gr_by               VARCHAR(100) NOT NULL COMMENT 'UUID user branch tujuan',
    gr_at               DATETIME(3) NOT NULL,
    notes               TEXT NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_asset_gr (asset_id) COMMENT 'Satu asset hanya bisa di-GR sekali',
    INDEX idx_asset_gr_transaction_id (transaction_id),
    INDEX idx_asset_gr_transaction_number (transaction_number),
    INDEX idx_asset_gr_asset_id (asset_id),
    INDEX idx_asset_gr_branch_code (branch_code),
    INDEX idx_asset_gr_gr_date (gr_date),

    CONSTRAINT fk_asset_gr_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_asset_gr_asset
        FOREIGN KEY (asset_id) REFERENCES assets(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS asset_gr;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS document_number_sequences;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_item_verifications;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_stages;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    DROP INDEX idx_transactions_current_stage,
    DROP COLUMN io_number,
    DROP COLUMN current_stage;
-- +goose StatementEnd