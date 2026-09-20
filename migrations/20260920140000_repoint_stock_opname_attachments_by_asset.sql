-- +goose Up
-- +goose StatementBegin
ALTER TABLE stock_opname_asset_photos
    DROP FOREIGN KEY fk_so_photo_item,
    DROP KEY uq_so_photo_item,
    DROP COLUMN transaction_stock_opname_id,
    ADD UNIQUE KEY uq_so_photo_asset (transaction_id, asset_id);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE stock_opname_borrow_documents
    DROP FOREIGN KEY fk_so_borrow_doc_item,
    DROP KEY uq_so_borrow_doc_item,
    DROP COLUMN transaction_stock_opname_id,
    ADD UNIQUE KEY uq_so_borrow_doc_asset (transaction_id, asset_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE stock_opname_borrow_documents
    DROP KEY uq_so_borrow_doc_asset,
    ADD COLUMN transaction_stock_opname_id BIGINT UNSIGNED NOT NULL AFTER id,
    ADD UNIQUE KEY uq_so_borrow_doc_item (transaction_stock_opname_id),
    ADD CONSTRAINT fk_so_borrow_doc_item FOREIGN KEY (transaction_stock_opname_id) REFERENCES transaction_stock_opnames(id) ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE stock_opname_asset_photos
    DROP KEY uq_so_photo_asset,
    ADD COLUMN transaction_stock_opname_id BIGINT UNSIGNED NOT NULL AFTER id,
    ADD UNIQUE KEY uq_so_photo_item (transaction_stock_opname_id),
    ADD CONSTRAINT fk_so_photo_item FOREIGN KEY (transaction_stock_opname_id) REFERENCES transaction_stock_opnames(id) ON DELETE CASCADE;
-- +goose StatementEnd
