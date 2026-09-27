-- +goose Up

-- +goose StatementBegin
-- Foto profil user. Path file di disk (di bawah AttachmentStorageRoot()/avatars),
-- disajikan lewat GET /users/:id/avatar yang ber-auth — bukan file statis.
ALTER TABLE users ADD COLUMN avatar_path VARCHAR(500) NULL AFTER mpn_number;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
ALTER TABLE users DROP COLUMN avatar_path;
-- +goose StatementEnd
