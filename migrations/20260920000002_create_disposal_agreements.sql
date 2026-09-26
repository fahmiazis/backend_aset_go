-- +goose Up
-- +goose StatementBegin
-- Kesepakatan disposal (agreement) — pengelompokan beberapa transaksi disposal
-- yang sudah lolos APPROVAL_REQUEST untuk disetujui sekaligus oleh manajemen
-- puncak.
--
-- Dibuat sebagai entitas sendiri, bukan stage di masing-masing transaksi,
-- karena satu persetujuan mencakup banyak transaksi dari banyak cabang.
-- Akibatnya satu aset disposal punya dua nomor: nomor transaksi request
-- (0001/BC000005/BANDUNG BARAT/IX/2026-DPSL) dan nomor agreement
-- (0001/IX/2026-DPSL-AGMNT).
CREATE TABLE disposal_agreements (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    agreement_number  VARCHAR(100)    NOT NULL,
    current_stage     VARCHAR(50)     NOT NULL DEFAULT 'APPROVAL_AGREEMENT',
    status            VARCHAR(50)     NOT NULL DEFAULT 'PENDING',
    notes             TEXT            NULL,
    rejection_reason  TEXT            NULL,
    created_by        CHAR(36)        NOT NULL,
    created_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at        DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    deleted_at        DATETIME(3)     NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_disposal_agreement_number (agreement_number),
    KEY idx_disposal_agreements_stage (current_stage),
    KEY idx_disposal_agreements_created_by (created_by),
    KEY idx_disposal_agreements_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose StatementBegin
-- Anggota agreement. transaction_number disimpan supaya riwayat tetap terbaca
-- walau transaksinya berpindah stage.
CREATE TABLE disposal_agreement_items (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    agreement_id       BIGINT UNSIGNED NOT NULL,
    transaction_id     BIGINT UNSIGNED NOT NULL,
    transaction_number VARCHAR(100)    NOT NULL,
    created_at         DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_agreement_transaction (agreement_id, transaction_id),
    KEY idx_agreement_items_transaction (transaction_number),
    CONSTRAINT fk_agreement_items_agreement FOREIGN KEY (agreement_id)
        REFERENCES disposal_agreements (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu bertipe permission untuk middleware.RequirePermission.
-- route_path harus sama dengan hasil normalisasi path request
-- (/api/v1/transactions/disposal-agreements → /transactions/disposal-agreements).
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(), NULL, 'Disposal Agreement', 'permission',
       '/dashboard/disposal-agreement', '/transactions/disposal-agreements',
       0, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT * FROM menus) m
    WHERE m.route_path = '/transactions/disposal-agreements' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO permissions (id, value, label, description, module, created_at, updated_at)
SELECT UUID(), 'manage_disposal_agreement', 'Kelola Kesepakatan Disposal',
       'Membuat dan mengajukan kesepakatan disposal (agreement)',
       'disposal', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT * FROM permissions) p WHERE p.value = 'manage_disposal_agreement'
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan permission yang muncul di matriks hak akses untuk menu ini
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), m.id, p.id, NOW(), NOW()
FROM menus m
JOIN permissions p ON p.value IN ('manage_disposal_agreement', 'read')
WHERE m.route_path = '/transactions/disposal-agreements'
  AND m.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT * FROM menu_permissions) mp
      WHERE mp.menu_id = m.id AND mp.permission_id = p.id
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path = '/transactions/disposal-agreements';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path = '/transactions/disposal-agreements';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM permissions WHERE value = 'manage_disposal_agreement';
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS disposal_agreement_items;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS disposal_agreements;
-- +goose StatementEnd
