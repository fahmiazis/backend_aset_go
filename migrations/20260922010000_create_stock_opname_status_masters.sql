-- +goose Up
-- +goose StatementBegin
CREATE TABLE stock_opname_physical_status_masters (
    id                                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code                               VARCHAR(50) NOT NULL,
    label                              VARCHAR(100) NOT NULL,
    requires_borrow_document           BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Wajib upload dokumen peminjaman sebelum status ini bisa disimpan/submit',
    requires_not_applicable_condition  BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'TRUE: condition wajib salah satu yang is_not_applicable_value=true. FALSE: condition dilarang pakai nilai itu',
    counts_as_missing                  BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Dihitung sebagai "Hilang" / "SAP ADA FISIK TIDAK" di laporan',
    is_system                          BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Status bawaan sistem, tidak bisa dihapus lewat API',
    created_at                         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at                         TIMESTAMP NULL DEFAULT NULL,
    UNIQUE KEY uq_stock_opname_physical_status_masters_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Master data status fisik stock opname — dulu hardcode EXISTS/MISSING/BORROWED';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE stock_opname_condition_masters (
    id                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    code                     VARCHAR(50) NOT NULL,
    label                    VARCHAR(100) NOT NULL,
    is_not_applicable_value  BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Representasi "Tidak Ada"/N.A, dulu hardcode NOT_APPLICABLE',
    report_bucket            VARCHAR(20) NOT NULL DEFAULT '' COMMENT 'BAIK | RUSAK | kosong (tidak dihitung bucket manapun)',
    is_system                BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Kondisi bawaan sistem, tidak bisa dihapus lewat API',
    created_at               TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at               TIMESTAMP NULL DEFAULT NULL,
    UNIQUE KEY uq_stock_opname_condition_masters_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Master data kondisi aset stock opname — dulu hardcode GOOD/FAIR/POOR/BROKEN/NOT_APPLICABLE';
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO stock_opname_physical_status_masters
    (code, label, requires_borrow_document, requires_not_applicable_condition, counts_as_missing, is_system)
VALUES
    ('EXISTS',   'Ada',       FALSE, FALSE, FALSE, TRUE),
    ('MISSING',  'Tidak Ada', FALSE, TRUE,  TRUE,  TRUE),
    ('BORROWED', 'Dipinjam',  TRUE,  TRUE,  FALSE, TRUE);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO stock_opname_condition_masters
    (code, label, is_not_applicable_value, report_bucket, is_system)
VALUES
    ('GOOD',           'Baik',        FALSE, 'BAIK',  TRUE),
    ('FAIR',           'Cukup',       FALSE, 'BAIK',  TRUE),
    ('POOR',           'Kurang',      FALSE, 'RUSAK', TRUE),
    ('BROKEN',         'Rusak Berat', FALSE, 'RUSAK', TRUE),
    ('NOT_APPLICABLE', 'Tidak Ada',   TRUE,  '',      TRUE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_condition_masters;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_physical_status_masters;
-- +goose StatementEnd
