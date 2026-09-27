package services

import (
	"backend-go/config"
	"backend-go/models"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ============================================================================
// PROFIL USER — ganti password sendiri & foto profil
// ============================================================================

// MaxAvatarSize — batas ukuran foto profil
const MaxAvatarSize = 2 << 20 // 2 MB

// avatarExtensions — tipe file yang diterima, dideteksi dari isi file
// (bukan dari nama/Content-Type kiriman client)
var avatarExtensions = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ChangeOwnPassword — user mengganti password-nya sendiri; password lama wajib
// benar. Admin mengganti password user lain lewat PUT /users/:id.
func ChangeOwnPassword(userID, currentPassword, newPassword string) error {
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}
	if currentPassword == newPassword {
		return errors.New("new password must be different from the current password")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	return config.DB.Model(&user).Update("password", string(hashed)).Error
}

// UserAvatarPath — path file foto profil; error kalau user tidak punya foto
// atau filenya hilang dari disk.
func UserAvatarPath(userID string) (string, error) {
	var user models.User
	if err := config.DB.Select("id", "avatar_path").First(&user, "id = ?", userID).Error; err != nil {
		return "", errors.New("user not found")
	}
	if user.AvatarPath == nil || *user.AvatarPath == "" {
		return "", errors.New("avatar not found")
	}
	if _, err := os.Stat(*user.AvatarPath); err != nil {
		return "", errors.New("avatar file not found on server")
	}
	return *user.AvatarPath, nil
}

// SaveUserAvatar menyimpan foto profil baru lalu menghapus file lama.
func SaveUserAvatar(userID string, file *multipart.FileHeader) error {
	if file.Size > MaxAvatarSize {
		return fmt.Errorf("file too large (max %d MB)", MaxAvatarSize>>20)
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	ext, ok := avatarExtensions[http.DetectContentType(head[:n])]
	if !ok {
		return errors.New("only JPG, PNG, or WEBP images are allowed")
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return err
	}

	dir := filepath.Join(AttachmentStorageRoot(), "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create avatar directory: %w", err)
	}

	path := filepath.Join(dir, fmt.Sprintf("%s_%d%s", userID, time.Now().UnixNano(), ext))
	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(path)
		return err
	}
	dst.Close()

	if err := config.DB.Model(&user).Update("avatar_path", path).Error; err != nil {
		os.Remove(path)
		return err
	}

	if user.AvatarPath != nil && *user.AvatarPath != "" && *user.AvatarPath != path {
		os.Remove(*user.AvatarPath)
	}
	return nil
}

// RemoveUserAvatar menghapus foto profil (kolom dikosongkan, file dihapus).
func RemoveUserAvatar(userID string) error {
	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}
	if user.AvatarPath == nil || *user.AvatarPath == "" {
		return nil
	}

	if err := config.DB.Model(&user).Update("avatar_path", nil).Error; err != nil {
		return err
	}
	os.Remove(*user.AvatarPath)
	return nil
}
