-- +goose Up
-- +goose StatementBegin
-- Mapping hak akses ↔ menu.
-- Menentukan permission MANA SAJA yang relevan untuk sebuah menu, sehingga
-- picker hak akses role hanya menawarkan permission yang benar-benar dipakai
-- menu tersebut (bukan semua permission untuk semua menu).
--
-- Catatan: tabel ini TIDAK menentukan siapa punya akses — itu tetap di
-- role_menus.permissions. Ini hanya daftar pilihan yang valid per menu.
CREATE TABLE IF NOT EXISTS menu_permissions (
    id CHAR(36) PRIMARY KEY,
    menu_id CHAR(36) NOT NULL,
    permission_id CHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY idx_menu_permission (menu_id, permission_id),
    INDEX idx_menu_permissions_menu (menu_id),
    INDEX idx_menu_permissions_permission (permission_id),

    CONSTRAINT fk_menu_permissions_menu
        FOREIGN KEY (menu_id)
        REFERENCES menus(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_menu_permissions_permission
        FOREIGN KEY (permission_id)
        REFERENCES permissions(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose StatementBegin
-- Seed mapping berdasarkan route_path.
--
-- Daftar di bawah adalah hasil normalisasi path oleh middleware.RequirePermission
-- (prefix /api/v1 dibuang, segment ID dilewati, maksimal 3 segment) dipasangkan
-- dengan permission yang dicek di folder routes/.
--
-- Menu yang route_path-nya belum cocok tidak akan ter-seed. Kalau nanti menunya
-- baru dibuat, jalankan ulang INSERT ini atau tambahkan barisnya manual.
INSERT IGNORE INTO menu_permissions (id, menu_id, permission_id)
SELECT UUID(), m.id, p.id
FROM menus m
JOIN (
    -- Approval
    SELECT '/transaction-approvals/initiate' AS route_path, 'create_transaction' AS permission_value
    -- Dokumen umum
    UNION ALL SELECT '/attachments/upload', 'upload_attachment'
    UNION ALL SELECT '/attachments/upload', 'create_transaction'
    UNION ALL SELECT '/attachments/review', 'review_attachment'
    -- Procurement
    UNION ALL SELECT '/transactions/procurement',                'create_transaction'
    UNION ALL SELECT '/transactions/procurement/submit',         'create_transaction'
    UNION ALL SELECT '/transactions/procurement/verify',         'verify_asset'
    UNION ALL SELECT '/transactions/procurement/approval',       'manage_approval'
    UNION ALL SELECT '/transactions/procurement/process-budget', 'process_budget'
    UNION ALL SELECT '/transactions/procurement/execute',        'execute_asset'
    UNION ALL SELECT '/transactions/procurement/gr',             'create_transaction'
    UNION ALL SELECT '/transactions/procurement/gr',             'gr'
    UNION ALL SELECT '/transactions/procurement/reject',         'reject_transaction'
    UNION ALL SELECT '/transactions/procurement/revise',         'update_transaction'
    -- Mutation
    UNION ALL SELECT '/transactions/mutation',                   'create_transaction'
    UNION ALL SELECT '/transactions/mutation/draft',             'create_transaction'
    UNION ALL SELECT '/transactions/mutation/approval',          'manage_approval'
    UNION ALL SELECT '/transactions/mutation/confirm-receiving', 'confirm_receiving'
    UNION ALL SELECT '/transactions/mutation/confirm-receiving', 'create_transaction'
    UNION ALL SELECT '/transactions/mutation/execute',           'execute_mutation'
    UNION ALL SELECT '/transactions/mutation/reject',            'reject_transaction'
    UNION ALL SELECT '/transactions/mutation/attachments',       'upload_attachment'
    UNION ALL SELECT '/transactions/mutation/attachments',       'create_transaction'
    UNION ALL SELECT '/transactions/mutation/attachments',       'review_attachment'
    UNION ALL SELECT '/mutation',                                'create_transaction'
    -- Disposal
    UNION ALL SELECT '/transactions/disposal',                    'create_transaction'
    UNION ALL SELECT '/transactions/disposal/draft',              'create_transaction'
    UNION ALL SELECT '/transactions/disposal/purchasing',         'manage_purchasing'
    UNION ALL SELECT '/transactions/disposal/approval-request',   'manage_approval'
    UNION ALL SELECT '/transactions/disposal/approval-agreement', 'manage_approval'
    UNION ALL SELECT '/transactions/disposal/execute',            'execute_disposal'
    UNION ALL SELECT '/transactions/disposal/finance',            'manage_finance'
    UNION ALL SELECT '/transactions/disposal/tax',                'manage_tax'
    UNION ALL SELECT '/transactions/disposal/asset-deletion',     'execute_asset_deletion'
    UNION ALL SELECT '/transactions/disposal/reject',             'reject_transaction'
    UNION ALL SELECT '/transactions/disposal/attachments',        'upload_attachment'
    UNION ALL SELECT '/transactions/disposal/attachments',        'review_attachment'
    UNION ALL SELECT '/transactions/disposal/old',                'create_transaction'
    -- Stock Opname
    UNION ALL SELECT '/transactions/stock-opname', 'create_transaction'
) rp
    ON rp.route_path = TRIM(TRAILING '/' FROM m.route_path)
JOIN permissions p
    ON p.value = rp.permission_value
WHERE m.deleted_at IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
-- Akses dasar (read/write/delete) berlaku untuk SEMUA menu.
INSERT IGNORE INTO menu_permissions (id, menu_id, permission_id)
SELECT UUID(), m.id, p.id
FROM menus m
CROSS JOIN permissions p
WHERE p.module = 'basic'
  AND m.deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS menu_permissions;
-- +goose StatementEnd
