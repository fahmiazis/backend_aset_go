-- +goose Up

-- +goose StatementBegin
CREATE TABLE stock_opname_asset_photos (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_stock_opname_id BIGINT UNSIGNED NOT NULL COMMENT 'FK ke transaction_stock_opnames — 1 foto per baris asset di draft',
    transaction_id              BIGINT UNSIGNED NOT NULL COMMENT 'Denormalized dari transaction_stock_opnames buat query cek duplikat per SO',
    asset_id                    BIGINT UNSIGNED NOT NULL,
    file_name                   VARCHAR(255) NOT NULL   COMMENT 'Nama file original',
    file_path                   VARCHAR(500) NOT NULL   COMMENT 'Path file di server',
    file_size                   BIGINT NOT NULL         COMMENT 'Ukuran file dalam bytes (max 2MB, divalidasi di aplikasi)',
    file_hash                   CHAR(64) NOT NULL       COMMENT 'SHA-256 hex isi file, buat cek foto duplikat antar-asset dalam 1 stock opname',
    captured_at                 DATETIME(3) NOT NULL    COMMENT 'Tanggal foto diambil, dibaca dari EXIF DateTimeOriginal',
    uploaded_by                 VARCHAR(100) NOT NULL   COMMENT 'UUID user yang upload',
    created_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_so_photo_item (transaction_stock_opname_id),
    INDEX idx_so_photo_transaction_id (transaction_id),
    INDEX idx_so_photo_asset_id (asset_id),
    INDEX idx_so_photo_hash (file_hash),

    CONSTRAINT fk_so_photo_item
        FOREIGN KEY (transaction_stock_opname_id) REFERENCES transaction_stock_opnames(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_so_photo_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_asset_photos;
-- +goose StatementEnd
