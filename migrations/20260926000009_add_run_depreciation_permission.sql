-- +goose Up

-- +goose StatementBegin
-- Hak akses tombol "Run Depreciation" di halaman Asset.
-- Catatan: pemetaan ke menu Asset di migrasi ini dipindah oleh
-- 20260926000010 ke menu permission "Asset Run Depreciation"
-- (route_path /depreciation/calculate), mengikuti pola Disposal Execute.
INSERT INTO permissions (id, value, label, description, module, created_at, updated_at)
SELECT UUID(), 'run_depreciation', 'Run Depreciation',
       'Menjalankan perhitungan penyusutan bulanan dari halaman Asset',
       'depreciation', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM (SELECT value FROM permissions) p WHERE p.value = 'run_depreciation');
-- +goose StatementEnd

-- +goose StatementBegin
-- Pilihan checkbox di matriks hak akses menu Asset
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM menus mn
JOIN permissions p ON p.value = 'run_depreciation'
WHERE mn.route_path = '/assets' AND mn.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT menu_id, permission_id FROM menu_permissions) mp
      WHERE mp.menu_id = mn.id AND mp.permission_id = p.id
  );
-- +goose StatementEnd

-- +goose StatementBegin
-- Endpoint ini dulunya RequireRole("admin"). Role admin langsung diberi hak
-- aksesnya supaya tidak kehilangan akses; role lain di-assign manual lewat
-- /dashboard/role/:id.
UPDATE role_menus rm
JOIN roles r ON r.id = rm.role_id
JOIN menus m ON m.id = rm.menu_id
SET rm.permissions = JSON_ARRAY_APPEND(rm.permissions, '$', 'run_depreciation'),
    rm.updated_at = NOW()
WHERE r.name = 'admin'
  AND m.route_path = '/assets' AND m.deleted_at IS NULL
  AND JSON_SEARCH(rm.permissions, 'one', 'run_depreciation') IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO role_menus (id, role_id, menu_id, permissions, created_at, updated_at)
SELECT UUID(), r.id, m.id, '["read","run_depreciation"]', NOW(), NOW()
FROM roles r
JOIN menus m ON m.route_path = '/assets' AND m.deleted_at IS NULL
WHERE r.name = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT role_id, menu_id FROM role_menus) x
      WHERE x.role_id = r.id AND x.menu_id = m.id
  );
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
UPDATE role_menus rm
JOIN menus m ON m.id = rm.menu_id
SET rm.permissions = JSON_REMOVE(rm.permissions, JSON_UNQUOTE(JSON_SEARCH(rm.permissions, 'one', 'run_depreciation')))
WHERE m.route_path = '/assets'
  AND JSON_SEARCH(rm.permissions, 'one', 'run_depreciation') IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN permissions p ON p.id = mp.permission_id
WHERE p.value = 'run_depreciation';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM permissions WHERE value = 'run_depreciation';
-- +goose StatementEnd
