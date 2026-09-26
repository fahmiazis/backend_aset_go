-- +goose Up
-- +goose StatementBegin
ALTER TABLE stock_opname_configs
    ADD COLUMN borrow_doc_allow_pdf   BOOLEAN NOT NULL DEFAULT TRUE  COMMENT 'Dokumen peminjaman boleh format PDF' AFTER submission_end_day,
    ADD COLUMN borrow_doc_allow_word  BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Dokumen peminjaman boleh format Word (.doc/.docx)' AFTER borrow_doc_allow_pdf,
    ADD COLUMN borrow_doc_allow_photo BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Dokumen peminjaman boleh format foto (JPEG/PNG/WEBP)' AFTER borrow_doc_allow_word,
    ADD COLUMN borrow_doc_is_required BOOLEAN NOT NULL DEFAULT TRUE  COMMENT 'Dokumen peminjaman wajib diisi buat status Dipinjam' AFTER borrow_doc_allow_photo;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE stock_opname_configs
    DROP COLUMN borrow_doc_allow_pdf,
    DROP COLUMN borrow_doc_allow_word,
    DROP COLUMN borrow_doc_allow_photo,
    DROP COLUMN borrow_doc_is_required;
-- +goose StatementEnd
