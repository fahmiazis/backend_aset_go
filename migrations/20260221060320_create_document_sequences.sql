-- +goose Up
-- +goose StatementBegin
CREATE TABLE document_sequences (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    prefix VARCHAR(10) NOT NULL COMMENT 'ACQ, MUT, DIS, SO',
    year INT NOT NULL,
    month INT NOT NULL,
    last_number INT NOT NULL DEFAULT 0 COMMENT 'Last generated sequential number',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_doc_seq_prefix_year_month (prefix, year, month),
    KEY idx_doc_seq_prefix (prefix),
    KEY idx_doc_seq_year_month (year, month)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Document number generator/counter';
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO document_sequences (prefix, year, month, last_number) VALUES
    ('ACQ', YEAR(CURDATE()), MONTH(CURDATE()), 0),
    ('MUT', YEAR(CURDATE()), MONTH(CURDATE()), 0),
    ('DIS', YEAR(CURDATE()), MONTH(CURDATE()), 0),
    ('SO',  YEAR(CURDATE()), MONTH(CURDATE()), 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS document_sequences;
-- +goose StatementEnd