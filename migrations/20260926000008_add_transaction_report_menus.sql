-- +goose Up

-- +goose StatementBegin
-- Grup Report di sidebar, tepat setelah Stock Opname. Grup level atas
-- sesudahnya (Master Data, Settings) digeser satu.
UPDATE menus
SET order_index = order_index + 1
WHERE parent_id IS NULL
  AND menu_type <> 'permission'
  AND deleted_at IS NULL
  AND order_index >= 6
  AND NOT EXISTS (
      SELECT 1 FROM (SELECT name, menu_type, deleted_at FROM menus) g
      WHERE g.name = 'Report' AND g.menu_type = 'group' AND g.deleted_at IS NULL
  );
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(), NULL, 'Report', 'group', NULL, NULL, 'lucide:file-spreadsheet', 6, 'active', NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM (SELECT name, menu_type, deleted_at FROM menus) g
    WHERE g.name = 'Report' AND g.menu_type = 'group' AND g.deleted_at IS NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
-- route_path = hasil normalisasi RequirePermission. Export excel memakai
-- endpoint yang sama (?format=xlsx), jadi tidak perlu menu permission terpisah.
INSERT INTO menus (id, parent_id, name, menu_type, path, route_path, icon_name, order_index, status, created_at, updated_at)
SELECT UUID(),
       (SELECT id FROM (SELECT id, name, menu_type, deleted_at FROM menus) g
        WHERE g.name = 'Report' AND g.menu_type = 'group' AND g.deleted_at IS NULL LIMIT 1),
       src.name, 'page', src.path, src.route_path, src.icon, src.ord, 'active', NOW(), NOW()
FROM (
              SELECT 'Procurement Report' AS name, '/dashboard/procurement-report' AS path, '/reports/procurement' AS route_path, 'lucide:shopping-cart'   AS icon, 0 AS ord
    UNION ALL SELECT 'Mutation Report',            '/dashboard/mutation-report',            '/reports/mutation',            'lucide:arrow-left-right',        1
    UNION ALL SELECT 'Disposal Report',            '/dashboard/disposal-report',            '/reports/disposal',            'lucide:trash-2',                 2
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
              SELECT '/reports/procurement' AS route_path, 'read' AS perm
    UNION ALL SELECT '/reports/mutation',                  'read'
    UNION ALL SELECT '/reports/disposal',                  'read'
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
WHERE m.route_path IN ('/reports/procurement', '/reports/mutation', '/reports/disposal')
   OR (m.name = 'Report' AND m.menu_type = 'group');
-- +goose StatementEnd

-- +goose StatementBegin
DELETE mp FROM menu_permissions mp
JOIN menus m ON m.id = mp.menu_id
WHERE m.route_path IN ('/reports/procurement', '/reports/mutation', '/reports/disposal');
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE route_path IN ('/reports/procurement', '/reports/mutation', '/reports/disposal');
-- +goose StatementEnd

-- +goose StatementBegin
DELETE FROM menus WHERE name = 'Report' AND menu_type = 'group';
-- +goose StatementEnd

-- +goose StatementBegin
UPDATE menus
SET order_index = order_index - 1
WHERE parent_id IS NULL
  AND menu_type <> 'permission'
  AND deleted_at IS NULL
  AND order_index >= 7;
-- +goose StatementEnd
