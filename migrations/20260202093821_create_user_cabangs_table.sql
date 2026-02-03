-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_cabangs (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    cabang_id CHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY idx_user_cabang (user_id, cabang_id),
    INDEX idx_user_id (user_id),
    INDEX idx_cabang_id (cabang_id),
    
    CONSTRAINT fk_user_cabangs_user 
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE,
    
    CONSTRAINT fk_user_cabangs_cabang 
        FOREIGN KEY (cabang_id) 
        REFERENCES cabangs(id) 
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_cabangs;
-- +goose StatementEnd