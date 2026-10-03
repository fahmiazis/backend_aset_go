package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
)

// Isi QR code label aset: nomor aset yang dienkripsi AES-256-GCM, bukan nomor
// aset polos. Kuncinya hanya ada di backend (QR_SECRET_KEY), jadi scanner biasa
// cuma melihat teks acak — aplikasi membukanya lewat POST /assets/qr/resolve.
//
// Format: "AMSQR1." + base64url(nonce || ciphertext+tag). Prefix dipakai
// aplikasi mobile untuk mengenali QR miliknya sebelum memanggil API.
//
// Nonce diturunkan dari HMAC nomor aset (deterministik), supaya QR satu aset
// selalu sama setiap kali dicetak ulang. Aman untuk GCM: nonce yang sama hanya
// pernah dipakai untuk plaintext yang sama persis.
//
// JANGAN mengganti QR_SECRET_KEY setelah label dicetak — semua label lama
// tidak akan bisa dibaca lagi.

const (
	assetQRPrefix  = "AMSQR1."
	assetQRVersion = "AMSQR1" // additional data GCM, mengikat versi format
)

var (
	ErrQRKeyMissing    = errors.New("QR_SECRET_KEY belum diisi di .env backend (minimal 16 karakter)")
	ErrQRPayloadFormat = errors.New("QR code tidak dikenali")
)

func assetQRKeys() (encKey, nonceKey []byte, err error) {
	secret := strings.TrimSpace(os.Getenv("QR_SECRET_KEY"))
	if len(secret) < 16 {
		return nil, nil, ErrQRKeyMissing
	}
	// dua kunci terpisah dari satu secret: enkripsi & penurun nonce
	enc := sha256.Sum256([]byte("ams-qr-enc|" + secret))
	nonce := sha256.Sum256([]byte("ams-qr-nonce|" + secret))
	return enc[:], nonce[:], nil
}

func assetQRCipher(encKey []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptAssetQR — nomor aset → isi QR code.
func EncryptAssetQR(assetNumber string) (string, error) {
	encKey, nonceKey, err := assetQRKeys()
	if err != nil {
		return "", err
	}
	aead, err := assetQRCipher(encKey)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, nonceKey)
	mac.Write([]byte(assetNumber))
	nonce := mac.Sum(nil)[:aead.NonceSize()]

	sealed := aead.Seal(nil, nonce, []byte(assetNumber), []byte(assetQRVersion))
	return assetQRPrefix + base64.RawURLEncoding.EncodeToString(append(nonce, sealed...)), nil
}

// DecryptAssetQR — isi QR code → nomor aset. Gagal kalau bukan QR buatan
// sistem ini atau isinya diubah (tag GCM tidak cocok).
func DecryptAssetQR(payload string) (string, error) {
	encKey, _, err := assetQRKeys()
	if err != nil {
		return "", err
	}
	payload = strings.TrimSpace(payload)
	if !strings.HasPrefix(payload, assetQRPrefix) {
		return "", ErrQRPayloadFormat
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(payload, assetQRPrefix))
	if err != nil {
		return "", ErrQRPayloadFormat
	}
	aead, err := assetQRCipher(encKey)
	if err != nil {
		return "", err
	}
	if len(raw) < aead.NonceSize()+aead.Overhead() {
		return "", ErrQRPayloadFormat
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte(assetQRVersion))
	if err != nil {
		return "", ErrQRPayloadFormat
	}
	return string(plain), nil
}

// GetAssetQRCodes — isi QR untuk banyak aset sekaligus (satu halaman tab QR).
// Aset di luar cabang user / tidak ada dilewati diam-diam, sama dengan
// GET /assets yang tidak pernah menampilkannya.
func GetAssetQRCodes(assetNumbers []string, viewer AssetViewer) ([]dto.AssetQRCodeResponse, error) {
	if _, _, err := assetQRKeys(); err != nil {
		return nil, err
	}

	query := config.DB.Model(&models.Asset{}).
		Where("deleted_at IS NULL AND asset_number IN ?", assetNumbers)
	if codes, all := AssetBranchScope(viewer); !all {
		if len(codes) == 0 {
			return []dto.AssetQRCodeResponse{}, nil
		}
		query = query.Where("branch_code IN ?", codes)
	}

	var numbers []string
	if err := query.Pluck("asset_number", &numbers).Error; err != nil {
		return nil, err
	}

	result := make([]dto.AssetQRCodeResponse, 0, len(numbers))
	for _, number := range numbers {
		payload, err := EncryptAssetQR(number)
		if err != nil {
			return nil, err
		}
		result = append(result, dto.AssetQRCodeResponse{AssetNumber: number, QRPayload: payload})
	}
	return result, nil
}
