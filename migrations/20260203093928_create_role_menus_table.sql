-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS role_menus (
    id CHAR(36) PRIMARY KEY,
    role_id CHAR(36) NOT NULL,
    menu_id CHAR(36) NOT NULL,
    permissions JSON NOT NULL DEFAULT ('[]'),              -- ["read", "write", "delete"]
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE KEY idx_role_menu (role_id, menu_id),
    INDEX idx_role_id (role_id),
    INDEX idx_menu_id (menu_id),

    CONSTRAINT fk_role_menus_role
        FOREIGN KEY (role_id)
        REFERENCES roles(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_role_menus_menu
        FOREIGN KEY (menu_id)
        REFERENCES menus(id)
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS role_menus;
-- +goose StatementEnd