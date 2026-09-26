-- +goose Up

-- +goose StatementBegin
-- Pemegang aset hasil serah terima. NULL = dipegang cabang.
ALTER TABLE assets
    ADD COLUMN IF NOT EXISTS assigned_user_id CHAR(36) NULL AFTER asset_status,
    ADD COLUMN IF NOT EXISTS assigned_at DATETIME(3) NULL AFTER assigned_user_id,
    ADD INDEX IF NOT EXISTS idx_assets_assigned_user (assigned_user_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Header serah terima memakai tabel transactions (seperti mutasi memakai
-- mutation_to_branch_code): jenis serah terima dan user penerima.
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS handover_type VARCHAR(20) NULL COMMENT 'HANDOVER | RETURN' AFTER mutation_to_branch_code,
    ADD COLUMN IF NOT EXISTS handover_to_user_id CHAR(36) NULL COMMENT 'penerima (HANDOVER)' AFTER handover_type;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS transaction_handover_assets (
    id                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    transaction_id        BIGINT UNSIGNED NOT NULL,
    transaction_number    VARCHAR(100) NOT NULL,
    asset_id              BIGINT UNSIGNED NOT NULL,
    asset_number          VARCHAR(100) NOT NULL,
    from_user_id          CHAR(36)     NULL     COMMENT 'pemegang sebelum serah terima, NULL = cabang',
    previous_asset_status VARCHAR(50)  NOT NULL COMMENT 'dipulihkan saat selesai/batal',
    notes                 TEXT         NULL,
    status                ENUM('PENDING', 'COMPLETED', 'CANCELLED') NOT NULL DEFAULT 'PENDING',
    needs_revision        TINYINT(1)   NOT NULL DEFAULT 0,
    revision_notes        TEXT         NULL,
    created_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),

    PRIMARY KEY (id),
    INDEX idx_handover_asset_transaction (transaction_id),
    INDEX idx_handover_asset_number (transaction_number),
    INDEX idx_handover_asset_asset (asset_id),
    INDEX idx_handover_asset_status (status),
    CONSTRAINT fk_handover_asset_transaction
        FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_handover_asset_asset
        FOREIGN KEY (asset_id) REFERENCES assets(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu Asset Handover di sidebar, tepat setelah Mutation. Menu level atas
-- sesudahnya digeser satu.
UPDATE menus
SET order_index = order_index + 1
WHERE parent_id IS NULL
  AND menu_type <> 'permission'
  AND deleted_at IS NULL
  AND order_index >= 3
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
      WHERE m.route_path = '/transactions/handover' AND m.deleted_at IS NULL
  );
-- +goose StatementEnd

-- +goose StatementBegin
-- route_path = hasil normalisasi RequirePermission: POST /transactions/handover
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(), NULL, 'Asset Handover', 'page', '/dashboard/handover', '/transactions/handover',
       'lucide:hand-helping', 3, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = '/transactions/handover' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu hak akses (tidak tampil di sidebar):
--   /draft/*            ubah & submit draft          → create_transaction
--   /confirm-receiving  konfirmasi PENGEMBALIAN       → confirm_receiving
-- Konfirmasi serah terima ke user dilakukan penerimanya sendiri dan tidak
-- butuh permission; menu confirm-receiving dibaca service untuk pengembalian.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, route_path, deleted_at FROM menus) p
        WHERE p.route_path = '/transactions/handover' AND p.deleted_at IS NULL LIMIT 1),
       src.name, 'permission', src.path, src.route_path, src.ord, 'active', NOW(), NOW()
FROM (
              SELECT 'Asset Handover Draft'   AS name, '/dashboard/handover/draft'   AS path, '/transactions/handover/draft'             AS route_path, 0 AS ord
    UNION ALL SELECT 'Asset Handover Return Confirmation', '/dashboard/handover/confirm-receiving', '/transactions/handover/confirm-receiving', 1
) src
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = src.route_path AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan permission di matriks hak akses. Assign ke role manual lewat
-- /dashboard/role/:id.
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM (
              SELECT '/transactions/handover'                   AS route_path, 'read'               AS perm
    UNION ALL SELECT '/transactions/handover',                                 'create_transaction'
    UNION ALL SELECT '/transactions/handover/draft',                           'create_transaction'
    UNION ALL SELECT '/transactions/handover/confirm-receiving',               'confirm_receiving'
) map
JOIN menus mn ON mn.route_path = map.route_path AND mn.deleted_at IS NULL
JOIN permissions p ON p.value = map.perm
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT menu_id, permission_id FROM menu_permissions) mp
    WHERE mp.menu_id = mn.id AND mp.permission_id = p.id
);
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DELETE rm FROM role_menus rm
JOIN menus m ON m.id = rm.menu_id
WHERE m.route_path LIKE '/transactions/handover%';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path LIKE '/transactions/handover%';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE menu_type = 'permission' AND route_path LIKE '/transactions/handover/%';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path = '/transactions/handover';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE menus
SET order_index = order_index - 1
WHERE parent_id IS NULL
  AND menu_type <> 'permission'
  AND deleted_at IS NULL
  AND order_index >= 4;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_handover_assets;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE transactions
    DROP COLUMN IF EXISTS handover_to_user_id,
    DROP COLUMN IF EXISTS handover_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE assets
    DROP INDEX IF EXISTS idx_assets_assigned_user,
    DROP COLUMN IF EXISTS assigned_at,
    DROP COLUMN IF EXISTS assigned_user_id;
-- +goose StatementEnd
