-- +goose Up

-- +goose StatementBegin
CREATE TABLE stock_opname_revision_assets (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id BIGINT UNSIGNED NOT NULL COMMENT 'Stock opname yang lagi direvisi',
    asset_id       BIGINT UNSIGNED NOT NULL COMMENT 'Asset yang dichecklist reviser — cuma ini yang boleh diubah selama DRAFT revisi',
    created_by     VARCHAR(100) NOT NULL    COMMENT 'UUID user yang mentrigger revisi (approver / eksekutor)',
    created_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_so_revision_asset (transaction_id, asset_id),
    INDEX idx_so_revision_asset_asset_id (asset_id),

    CONSTRAINT fk_so_revision_asset_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_revision_assets;
-- +goose StatementEnd
