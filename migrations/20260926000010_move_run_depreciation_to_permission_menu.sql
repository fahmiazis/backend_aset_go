-- +goose Up

-- +goose StatementBegin
-- run_depreciation dipindah dari menu Asset (20260926000009) ke menu bertipe
-- permission tersendiri, mengikuti pola Disposal Execute: tampil sebagai baris
-- sendiri di /dashboard/menu dan di matriks hak akses role.
-- route_path = hasil normalisasi RequirePermission untuk POST /depreciation/calculate.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, route_path, deleted_at FROM menus) p
        WHERE p.route_path = '/assets' AND p.deleted_at IS NULL LIMIT 1),
       'Asset Run Depreciation', 'permission', '/dashboard/asset/run-depreciation', '/depreciation/calculate',
       0, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT route_path, deleted_at FROM menus) m
    WHERE m.route_path = '/depreciation/calculate' AND m.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO menu_permissions (id, menu_id, permission_id, created_at, updated_at)
SELECT UUID(), mn.id, p.id, NOW(), NOW()
FROM menus mn
JOIN permissions p ON p.value = 'run_depreciation'
WHERE mn.route_path = '/depreciation/calculate' AND mn.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT menu_id, permission_id FROM menu_permissions) mp
      WHERE mp.menu_id = mn.id AND mp.permission_id = p.id
  );
-- +goose StatementEnd

-- +goose StatementBegin
-- Role yang sudah punya run_depreciation di menu Asset (admin, dari 0009)
-- dipindahkan ke menu baru.
INSERT INTO role_menus (id, role_id, menu_id, permissions, created_at, updated_at)
SELECT UUID(), rm.role_id, target.id, '["run_depreciation"]', NOW(), NOW()
FROM role_menus rm
JOIN menus asset ON asset.id = rm.menu_id AND asset.route_path = '/assets' AND asset.deleted_at IS NULL
JOIN menus target ON target.route_path = '/depreciation/calculate' AND target.deleted_at IS NULL
WHERE JSON_SEARCH(rm.permissions, 'one', 'run_depreciation') IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT role_id, menu_id FROM role_menus) x
      WHERE x.role_id = rm.role_id AND x.menu_id = target.id
  );
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE role_menus rm
JOIN menus m ON m.id = rm.menu_id
SET rm.permissions = JSON_REMOVE(rm.permissions, JSON_UNQUOTE(JSON_SEARCH(rm.permissions, 'one', 'run_depreciation'))),
    rm.updated_at = NOW()
WHERE m.route_path = '/assets'
  AND JSON_SEARCH(rm.permissions, 'one', 'run_depreciation') IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
JOIN permissions p ON p.id = mp.permission_id
WHERE m.route_path = '/assets' AND p.value = 'run_depreciation';
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
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
UPDATE role_menus rm
JOIN menus asset ON asset.id = rm.menu_id AND asset.route_path = '/assets'
JOIN role_menus src ON src.role_id = rm.role_id
JOIN menus target ON target.id = src.menu_id AND target.route_path = '/depreciation/calculate'
SET rm.permissions = JSON_ARRAY_APPEND(rm.permissions, '$', 'run_depreciation')
WHERE JSON_SEARCH(rm.permissions, 'one', 'run_depreciation') IS NULL
  AND JSON_SEARCH(src.permissions, 'one', 'run_depreciation') IS NOT NULL;
-- +goose StatementEnd

-- +goose StatementBegin
DELETE rm FROM role_menus rm
JOIN menus m ON m.id = rm.menu_id
WHERE m.route_path = '/depreciation/calculate';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path = '/depreciation/calculate';
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path = '/depreciation/calculate' AND menu_type = 'permission';
-- +goose StatementEnd
