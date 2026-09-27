-- +goose Up

-- Relasi status fisik -> kondisi yang boleh dipilih. Menggantikan aturan
-- hardcode requires_not_applicable_condition / is_not_applicable_value:
-- sekarang tiap status fisik punya daftar kondisi sendiri yang bisa diatur
-- lewat master data (misal "Ada" -> Baik/Cukup/Kurang/Rusak Berat,
-- "Tidak Ada" -> Tidak Ada). Kolom flag lama dibiarkan di tabel (gak dipakai
-- lagi) biar migration ini bisa di-rollback tanpa kehilangan data.

-- +goose StatementBegin
CREATE TABLE stock_opname_physical_condition_rules (
    id                  BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    physical_status_id  BIGINT UNSIGNED NOT NULL,
    condition_id        BIGINT UNSIGNED NOT NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    UNIQUE KEY uq_so_physical_condition_rule (physical_status_id, condition_id),
    INDEX idx_so_physical_condition_rule_condition (condition_id),

    CONSTRAINT fk_so_physical_condition_rule_physical
        FOREIGN KEY (physical_status_id) REFERENCES stock_opname_physical_status_masters(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_so_physical_condition_rule_condition
        FOREIGN KEY (condition_id) REFERENCES stock_opname_condition_masters(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- Backfill dari aturan lama biar perilakunya sama persis: status yang
-- requires_not_applicable_condition=TRUE cuma boleh kondisi N.A, sisanya
-- cuma boleh kondisi non-N.A.
-- +goose StatementBegin
INSERT INTO stock_opname_physical_condition_rules (physical_status_id, condition_id)
SELECT p.id, c.id
FROM stock_opname_physical_status_masters p
JOIN stock_opname_condition_masters c
    ON c.is_not_applicable_value = p.requires_not_applicable_condition
WHERE p.deleted_at IS NULL AND c.deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS stock_opname_physical_condition_rules;
-- +goose StatementEnd
