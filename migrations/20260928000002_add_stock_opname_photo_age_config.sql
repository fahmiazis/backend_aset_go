-- +goose Up
-- +goose StatementBegin
ALTER TABLE stock_opname_configs
    ADD COLUMN photo_upload_max_age_days INT NOT NULL DEFAULT 10 COMMENT 'Foto bukti fisik: maks umur tanggal modified file dihitung dari saat upload (hari)' AFTER borrow_doc_is_required,
    ADD COLUMN photo_submit_max_age_days INT NOT NULL DEFAULT 10 COMMENT 'Foto bukti fisik: maks jarak dari tanggal upload foto sampai submit (hari)' AFTER photo_upload_max_age_days;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE stock_opname_configs
    DROP COLUMN photo_upload_max_age_days,
    DROP COLUMN photo_submit_max_age_days;
-- +goose StatementEnd
