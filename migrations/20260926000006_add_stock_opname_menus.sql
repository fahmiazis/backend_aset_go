-- +goose Up

-- +goose StatementBegin
-- Permission khusus stock opname — dipakai POST /transactions/stock-opname/execute
INSERT INTO permissions (id, value, label, description, module, created_at, updated_at)
SELECT UUID(), 'execute_stock_opname', 'Eksekusi Stock Opname',
       'Menjalankan hasil stock opname setelah semua approval disetujui (EXECUTE_STOCK_OPNAME → FINISHED)',
       'stock_opname', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM (SELECT value FROM permissions) p WHERE p.value = 'execute_stock_opname');
-- +goose StatementEnd

-- +goose StatementBegin
-- Grup Stock Opname di sidebar, tepat setelah Disposal. Grup level atas
-- sesudahnya (Master Data, Settings) digeser satu supaya urutannya tidak
-- bertumpuk.
UPDATE menus
SET order_index = order_index + 1
WHERE parent_id IS NULL
  AND menu_type <> 'permission'
  AND deleted_at IS NULL
  AND order_index >= 4
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT name, menu_type, deleted_at FROM menus) g
      WHERE g.name = 'Stock Opname' AND g.menu_type = 'group' AND g.deleted_at IS NULL
  );
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(), NULL, 'Stock Opname', 'group', NULL, NULL, 'lucide:clipboard-check', 4, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT name, menu_type, deleted_at FROM menus) g
    WHERE g.name = 'Stock Opname' AND g.menu_type = 'group' AND g.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Halaman di dalam grup Stock Opname.
--
-- route_path = hasil normalisasi middleware.RequirePermission (buang /api/v1,
-- lewati segment id, maksimal 3 segment):
--   POST /transactions/stock-opname          → /transactions/stock-opname
--   GET  /transactions/stock-opname/report/* → /transactions/stock-opname/report
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, name, menu_type, deleted_at FROM menus) g
        WHERE g.name = 'Stock Opname' AND g.menu_type = 'group' AND g.deleted_at IS NULL LIMIT 1),
       src.name, 'page', src.path, src.route_path, src.icon, src.ord, 'active', NOW(), NOW()
FROM (
              SELECT 'Stock Opname List'   AS name, '/dashboard/stock-opname'        AS path, '/transactions/stock-opname'        AS route_path, 'lucide:list-checks'   AS icon, 0 AS ord
    UNION ALL SELECT 'Stock Opname Report',         '/dashboard/stock-opname/report',         '/transactions/stock-opname/report',         'lucide:chart-column',        1
) src
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = src.route_path AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pengaturan jendela tanggal submit stock opname — ditaruh di grup Settings,
-- sejajar Attachment Setting dan Email Setting.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, name, menu_type, deleted_at FROM menus) g
        WHERE g.name = 'Settings' AND g.menu_type = 'group' AND g.deleted_at IS NULL LIMIT 1),
       'Stock Opname Setting', 'page',
       '/dashboard/stock-opname/config', '/transactions/stock-opname/config',
       'lucide:calendar-cog', 2, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = '/transactions/stock-opname/config' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Menu bertipe permission untuk endpoint aksi. Tidak tampil di sidebar, hanya
-- memetakan route → hak akses. Tanpa baris ini RequirePermission menolak
-- request dengan "route not registered in menu" untuk SEMUA user.
--   /draft/*       (isi temuan, foto, dokumen pinjam, template, submit)
--   /approval/*    (initiate approval manual)
--   /execute       (EXECUTE_STOCK_OPNAME → FINISHED)
--   /reject
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, route_path, deleted_at FROM menus) p
        WHERE p.route_path = '/transactions/stock-opname' AND p.deleted_at IS NULL LIMIT 1),
       src.name, 'permission', src.path, src.route_path, src.ord, 'active', NOW(), NOW()
FROM (
              SELECT 'Stock Opname Draft'    AS name, '/dashboard/stock-opname/draft'    AS path, '/transactions/stock-opname/draft'    AS route_path, 0 AS ord
    UNION ALL SELECT 'Stock Opname Approval',         '/dashboard/stock-opname/approval',         '/transactions/stock-opname/approval',         1
    UNION ALL SELECT 'Stock Opname Execute',          '/dashboard/stock-opname/execute',          '/transactions/stock-opname/execute',          2
    UNION ALL SELECT 'Stock Opname Reject',           '/dashboard/stock-opname/reject',           '/transactions/stock-opname/reject',           3
) src
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = src.route_path AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan permission yang muncul di matriks hak akses per menu. Assign ke
-- role dilakukan manual lewat /dashboard/role/:id.
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM (
              SELECT '/transactions/stock-opname'          AS route_path, 'read'                 AS perm
    UNION ALL SELECT '/transactions/stock-opname',                        'create_transaction'
    UNION ALL SELECT '/transactions/stock-opname/report',                 'read'
    UNION ALL SELECT '/transactions/stock-opname/config',                 'read'
    UNION ALL SELECT '/transactions/stock-opname/config',                 'write'
    UNION ALL SELECT '/transactions/stock-opname/draft',                  'create_transaction'
    UNION ALL SELECT '/transactions/stock-opname/approval',               'manage_approval'
    UNION ALL SELECT '/transactions/stock-opname/execute',                'execute_stock_opname'
    UNION ALL SELECT '/transactions/stock-opname/reject',                 'reject_transaction'
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
WHERE m.route_path LIKE '/transactions/stock-opname%'
   OR (m.name = 'Stock Opname' AND m.menu_type = 'group');
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path LIKE '/transactions/stock-opname%';
-- +goose StatementEnd

-- +goose StatementBegin
-- anak dulu (permission → halaman), baru grupnya
DELETE FROM menus WHERE menu_type = 'permission' AND route_path LIKE '/transactions/stock-opname/%';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path IN (
    '/transactions/stock-opname',
    '/transactions/stock-opname/report',
    '/transactions/stock-opname/config'
);
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE name = 'Stock Opname' AND menu_type = 'group';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE menus
SET order_index = order_index - 1
WHERE parent_id IS NULL
  AND menu_type <> 'permission'
  AND deleted_at IS NULL
  AND order_index >= 5;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM permissions WHERE value = 'execute_stock_opname';
-- +goose StatementEnd
