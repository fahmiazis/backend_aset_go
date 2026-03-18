-- +goose Up

-- +goose StatementBegin
ALTER TABLE document_number_sequences
    MODIFY COLUMN sequence_type ENUM('IO', 'ASSET', 'DN') NOT NULL
        COMMENT 'IO = Investment Order, ASSET = Asset Number, DN = Document Number';
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
ALTER TABLE document_number_sequences
    MODIFY COLUMN sequence_type ENUM('IO', 'ASSET') NOT NULL
        COMMENT 'IO = Investment Order, ASSET = Asset Number';
-- +goose StatementEnd