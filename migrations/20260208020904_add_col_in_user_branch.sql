-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_branchs
ADD COLUMN branch_type ENUM('homebase','temporary','assignment') DEFAULT 'homebase' AFTER branch_id;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE user_branchs
ADD COLUMN is_active BOOLEAN DEFAULT TRUE AFTER branch_type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE user_branchs
ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_branch_type ON user_branchs(branch_type);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_is_active ON user_branchs(is_active);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_is_active ON user_branchs;
-- +goose StatementEnd

-- +goose StatementBegin
DROP INDEX idx_branch_type ON user_branchs;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE user_branchs DROP COLUMN updated_at;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE user_branchs DROP COLUMN is_active;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE user_branchs DROP COLUMN branch_type;
-- +goose StatementEnd