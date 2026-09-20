-- +goose Up

-- +goose StatementBegin
CREATE TABLE stock_opname_borrow_documents (
    id                          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_stock_opname_id BIGINT UNSIGNED NOT NULL COMMENT 'FK ke transaction_stock_opnames — 1 dokumen per baris asset berstatus BORROWED',
    transaction_id              BIGINT UNSIGNED NOT NULL COMMENT 'Denormalized dari transaction_stock_opnames buat query per SO',
    asset_id                    BIGINT UNSIGNED NOT NULL,
    file_name                   VARCHAR(255) NOT NULL   COMMENT 'Nama file original',
    file_path                   VARCHAR(500) NOT NULL   COMMENT 'Path file di server',
    file_size                   BIGINT NOT NULL         COMMENT 'Ukuran file dalam bytes',
    uploaded_by                 VARCHAR(100) NOT NULL   COMMENT 'UUID user yang upload',
    created_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_so_borrow_doc_item (transaction_stock_opname_id),
    INDEX idx_so_borrow_doc_transaction_id (transaction_id),
    INDEX idx_so_borrow_doc_asset_id (asset_id),

    CONSTRAINT fk_so_borrow_doc_item
        FOREIGN KEY (transaction_stock_opname_id) REFERENCES transaction_stock_opnames(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_so_borrow_doc_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_borrow_documents;
-- +goose StatementEnd
