-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- Add assignment system fields
-- +goose StatementBegin
ALTER TABLE approval_flow_steps
ADD COLUMN type ENUM('it','non-it','all') DEFAULT 'all' AFTER is_visible;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flow_steps
ADD COLUMN category ENUM('budget','non-budget','return','all') DEFAULT 'all' AFTER type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flow_steps
ADD COLUMN approval_way ENUM('web','upload') DEFAULT 'web' AFTER category;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flow_steps 
DROP COLUMN type;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flow_steps 
DROP COLUMN category;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE approval_flow_steps
DROP COLUMN approval_way;
-- +goose StatementEnd