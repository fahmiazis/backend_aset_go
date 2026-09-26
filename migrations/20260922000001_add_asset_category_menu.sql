-- +goose Up
-- +goose StatementBegin
-- Menu master kategori aset.
--
-- route_path harus sama dengan hasil normalisasi path request di
-- middleware.RequirePermission: /api/v1/asset-categories → /asset-categories.
--
-- Catatan: endpoint CRUD-nya dijaga middleware.RequireRole("admin"), bukan
-- RequirePermission. Jadi permission di bawah menentukan tampil/tidaknya menu
-- di sidebar, sementara aksi tulisnya tetap khusus admin.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id FROM menus WHERE name = 'Master Data' AND menu_type = 'group' AND deleted_at IS NULL LIMIT 1) g),
       'Asset Category', 'page',
       '/dashboard/asset-category', '/asset-categories',
       'solar:widget-broken', 1, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT * FROM menus) m
    WHERE m.route_path = '/asset-categories' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan permission yang muncul di matriks hak akses untuk menu ini
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), m.id, p.id, NOW(), NOW()
FROM menus m
JOIN permissions p ON p.value IN ('read', 'write', 'delete')
WHERE m.route_path = '/asset-categories'
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
WHERE m.route_path = '/asset-categories';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path = '/asset-categories';
-- +goose StatementEnd
