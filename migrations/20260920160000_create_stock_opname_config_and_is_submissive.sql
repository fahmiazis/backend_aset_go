-- +goose Up
-- +goose StatementBegin
CREATE TABLE stock_opname_configs (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    submission_start_day  TINYINT UNSIGNED NOT NULL COMMENT 'Tanggal mulai jendela submit tiap bulan (1-31)',
    submission_end_day    TINYINT UNSIGNED NOT NULL COMMENT 'Tanggal akhir jendela submit tiap bulan (1-31). Kalau < start_day berarti wrap ke bulan berikutnya (mis. 25 s/d 8)',
    updated_by            VARCHAR(100)     NULL COMMENT 'UUID user admin yang terakhir update',
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Config stock opname — single-row settings, ditambah kolom baru seiring butuh config lain (mis. hak akses, nyusul)';
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO stock_opname_configs (submission_start_day, submission_end_day) VALUES (25, 8);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN is_submissive BOOLEAN NULL COMMENT 'Stock opname doang: TRUE kalau di-submit dalam jendela stock_opname_configs, FALSE kalau di luar jendela, NULL kalau belum pernah di-submit' AFTER current_stage;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP COLUMN is_submissive;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_configs;
-- +goose StatementEnd
