-- +goose Up

-- +goose StatementBegin
-- Hak akses upload Excel aset (POST /assets/import). Pola sama dengan
-- "Asset Run Depreciation": menu bertipe permission di bawah menu Asset.
-- Upload branch & user tidak butuh migrasi — endpoint-nya RequireRole("admin").
INSERT INTO permissions (id, value, label, description, module, created_at, updated_at)
SELECT UUID(), 'import_asset', 'Upload Asset',
       'Upload Excel aset baru / mass update aset dari halaman Asset',
       'asset', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM (SELECT value FROM permissions) p WHERE p.value = 'import_asset');
-- +goose StatementEnd

-- +goose StatementBegin
-- route_path = hasil normalisasi RequirePermission untuk POST /assets/import
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, route_path, deleted_at FROM menus) p
        WHERE p.route_path = '/assets' AND p.deleted_at IS NULL LIMIT 1),
       'Asset Upload', 'permission', '/dashboard/asset/upload', '/assets/import',
       0, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = '/assets/import' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM menus mn
JOIN permissions p ON p.value = 'import_asset'
WHERE mn.route_path = '/assets/import' AND mn.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT menu_id, permission_id FROM menu_permissions) mp
      WHERE mp.menu_id = mn.id AND mp.permission_id = p.id
  );
-- +goose StatementEnd

-- +goose StatementBegin
-- Role admin langsung diberi akses; role lain (mis. pic asset) di-assign
-- lewat /dashboard/role/:id.
INSERT INTO role_menus (id, role_id, menu_id, permissions, created_at, updated_at)
SELECT UUID(), r.id, m.id, '["import_asset"]', NOW(), NOW()
FROM roles r
JOIN menus m ON m.route_path = '/assets/import' AND m.deleted_at IS NULL
WHERE r.name = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT role_id, menu_id FROM role_menus) x
      WHERE x.role_id = r.id AND x.menu_id = m.id
  );
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DELETE rm FROM role_menus rm
JOIN menus m ON m.id = rm.menu_id
WHERE m.route_path = '/assets/import';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN permissions p ON p.id = mp.permission_id
WHERE p.value = 'import_asset';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path = '/assets/import' AND menu_type = 'permission';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM permissions WHERE value = 'import_asset';
-- +goose StatementEnd
