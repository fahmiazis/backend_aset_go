-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS menus (
    id CHAR(36) PRIMARY KEY,
    parent_id CHAR(36) NULL,
    name VARCHAR(100) NOT NULL,
    path VARCHAR(255) NULL,                                    -- frontend route path (untuk routing di React)
    route_path VARCHAR(255) NULL,                              -- backend API route path (untuk permission check di middleware)
    icon_name VARCHAR(100) NULL,                               -- nama icon dari lucide-react, misal: "LayoutDashboard"
    order_index INT NOT NULL DEFAULT 0,                        -- urutan di sidebar
    status ENUM('active', 'inactive') DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,

    INDEX idx_parent_id (parent_id),
    INDEX idx_route_path (route_path),
    INDEX idx_order_index (order_index),
    INDEX idx_status (status),
    INDEX idx_deleted_at (deleted_at),

    CONSTRAINT fk_menus_parent
        FOREIGN KEY (parent_id)
        REFERENCES menus(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS menus;
-- +goose StatementEnd