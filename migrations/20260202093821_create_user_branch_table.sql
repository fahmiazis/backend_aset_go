-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_branchs (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    branch_id CHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY idx_user_branch (user_id, branch_id),
    INDEX idx_user_id (user_id),
    INDEX idx_branch_id (branch_id),
    
    CONSTRAINT fk_user_branchs_user 
        FOREIGN KEY (user_id) 
        REFERENCES users(id) 
        ON DELETE CASCADE,
    
    CONSTRAINT fk_user_branchs_branch 
        FOREIGN KEY (branch_id) 
        REFERENCES branchs(id) 
        ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_branchs;
-- +goose StatementEnd