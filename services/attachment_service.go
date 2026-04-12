package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

const AttachmentStoragePath = "/app/documents"

// ============================================================
// ATTACHMENT CONFIG CRUD
// ============================================================

func GetAttachmentConfigs(transactionType, stage, branchCode string) ([]dto.AttachmentConfigResponse, error) {
	query := config.DB.Model(&models.AttachmentConfig{}).Where("is_active = ?", true)

	if transactionType != "" {
		query = query.Where("transaction_type IN ?", []string{transactionType, models.AttachmentTransactionTypeAll})
	}
	if stage != "" {
		query = query.Where("stage IN ?", []string{stage, models.AttachmentStageAll})
	}
	if branchCode != "" {
		query = query.Where("branch_code IN ?", []string{branchCode, models.AttachmentBranchAll})
	}

	var configs []models.AttachmentConfig
	if err := query.Find(&configs).Error; err != nil {
		return nil, err
	}

	return mapAttachmentConfigsToResponse(configs), nil
}

func GetAttachmentConfigByID(id uint) (*dto.AttachmentConfigResponse, error) {
	var config_ models.AttachmentConfig
	if err := config.DB.First(&config_, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attachment config not found")
		}
		return nil, err
	}
	response := mapAttachmentConfigToResponse(config_)
	return &response, nil
}

func CreateAttachmentConfig(userID string, req dto.CreateAttachmentConfigRequest) (*dto.AttachmentConfigResponse, error) {
	cfg := models.AttachmentConfig{
		TransactionType: req.TransactionType,
		Stage:           req.Stage,
		BranchCode:      req.BranchCode,
		AttachmentType:  req.AttachmentType,
		Description:     req.Description,
		IsRequired:      req.IsRequired,
		IsActive:        req.IsActive,
		CreatedBy:       &userID,
	}

	if err := config.DB.Create(&cfg).Error; err != nil {
		return nil, err
	}

	response := mapAttachmentConfigToResponse(cfg)
	return &response, nil
}

func UpdateAttachmentConfig(id uint, req dto.UpdateAttachmentConfigRequest) (*dto.AttachmentConfigResponse, error) {
	var cfg models.AttachmentConfig
	if err := config.DB.First(&cfg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attachment config not found")
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Description != nil {
		updates["description"] = req.Description
	}
	if req.IsRequired != nil {
		updates["is_required"] = *req.IsRequired
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if err := config.DB.Model(&cfg).Updates(updates).Error; err != nil {
		return nil, err
	}

	response := mapAttachmentConfigToResponse(cfg)
	return &response, nil
}

func DeleteAttachmentConfig(id uint) error {
	var cfg models.AttachmentConfig
	if err := config.DB.First(&cfg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("attachment config not found")
		}
		return err
	}
	return config.DB.Delete(&cfg).Error
}

// ============================================================
// UPLOAD ATTACHMENT
// ============================================================

func UploadAttachment(
	userID string,
	transactionNumber string,
	transactionType string,
	stage string,
	configID uint,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (*dto.TransactionAttachmentResponse, error) {

	// Validasi transaction exist
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}

	// Validasi attachment config exist dan aktif
	var attachmentConfig models.AttachmentConfig
	if err := config.DB.First(&attachmentConfig, configID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attachment config not found")
		}
		return nil, err
	}

	if !attachmentConfig.IsActive {
		return nil, errors.New("attachment config is inactive")
	}

	// Cek apakah sudah ada attachment untuk config ini di transaksi ini
	// Kalau ada yang PENDING atau APPROVED, tidak bisa upload lagi
	var existingCount int64
	config.DB.Model(&models.TransactionAttachment{}).
		Where("transaction_id = ? AND attachment_config_id = ? AND status IN ?",
			transaction.ID, configID, []string{models.AttachmentStatusPending, models.AttachmentStatusApproved}).
		Count(&existingCount)

	if existingCount > 0 {
		return nil, errors.New("attachment already uploaded for this config, please wait for review or re-upload after rejection")
	}

	// Buat direktori kalau belum ada
	// Struktur: /home/dev/.../documents/{transaction_type}/{transaction_number}/{stage}/
	dirPath := filepath.Join(
		AttachmentStoragePath,
		transactionType,
		sanitizePathSegment(transactionNumber),
		stage,
	)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Nama file: {timestamp}_{original_filename}
	timestamp := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
	filePath := filepath.Join(dirPath, fileName)

	// Simpan file
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	fileSize, err := io.Copy(dst, file)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Detect mime type dari extension
	mimeType := detectMimeType(fileHeader.Filename)

	now := time.Now()
	attachment := models.TransactionAttachment{
		TransactionID:      transaction.ID,
		TransactionNumber:  transactionNumber,
		TransactionType:    transactionType,
		Stage:              stage,
		AttachmentConfigID: configID,
		FileName:           fileHeader.Filename,
		FilePath:           filePath,
		FileSize:           &fileSize,
		MimeType:           &mimeType,
		Status:             models.AttachmentStatusPending,
		UploadedBy:         userID,
		UploadedAt:         now,
	}

	if err := config.DB.Create(&attachment).Error; err != nil {
		// Hapus file kalau DB error
		os.Remove(filePath)
		return nil, err
	}

	attachment.AttachmentConfig = &attachmentConfig
	response := mapTransactionAttachmentToResponse(attachment)
	return &response, nil
}

// ============================================================
// REVIEW ATTACHMENT (APPROVE / REJECT)
// ============================================================

func ReviewAttachment(reviewerID string, attachmentID uint, req dto.ReviewAttachmentRequest) (*dto.TransactionAttachmentResponse, error) {
	if req.Status == models.AttachmentStatusRejected && (req.RejectionReason == nil || *req.RejectionReason == "") {
		return nil, errors.New("rejection_reason is required when rejecting")
	}

	var attachment models.TransactionAttachment
	if err := config.DB.
		Preload("AttachmentConfig").
		First(&attachment, attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("attachment not found")
		}
		return nil, err
	}

	if attachment.Status != models.AttachmentStatusPending {
		return nil, errors.New("only PENDING attachments can be reviewed")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":      req.Status,
		"reviewed_by": reviewerID,
		"reviewed_at": now,
	}
	if req.RejectionReason != nil {
		updates["rejection_reason"] = req.RejectionReason
	}

	if err := config.DB.Model(&attachment).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Reload
	config.DB.Preload("AttachmentConfig").First(&attachment, attachmentID)
	response := mapTransactionAttachmentToResponse(attachment)
	return &response, nil
}

// ============================================================
// GET ATTACHMENTS
// ============================================================

func GetTransactionAttachments(transactionNumber, transactionType, stage string) ([]dto.TransactionAttachmentResponse, error) {
	query := config.DB.
		Preload("AttachmentConfig").
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType)

	if stage != "" {
		query = query.Where("stage = ?", stage)
	}

	var attachments []models.TransactionAttachment
	if err := query.Order("created_at ASC").Find(&attachments).Error; err != nil {
		return nil, err
	}

	return mapTransactionAttachmentsToResponse(attachments), nil
}

// GetAttachmentStatusSummary — cek apakah semua required attachment sudah APPROVED
// untuk stage tertentu. Dipakai sebelum transisi ke stage berikutnya.
func GetAttachmentStatusSummary(transactionNumber, transactionType, stage, branchCode string) (*dto.AttachmentStatusSummary, error) {
	// Get required configs untuk stage ini berdasarkan prioritas branch
	requiredConfigs, err := getRequiredConfigs(transactionType, stage, branchCode)
	if err != nil {
		return nil, err
	}

	// Get semua attachment yang sudah diupload untuk stage ini
	var attachments []models.TransactionAttachment
	config.DB.
		Preload("AttachmentConfig").
		Where("transaction_number = ? AND transaction_type = ? AND stage = ?",
			transactionNumber, transactionType, stage).
		Find(&attachments)

	// Map attachment by config_id
	attachmentByConfig := make(map[uint]models.TransactionAttachment)
	for _, att := range attachments {
		attachmentByConfig[att.AttachmentConfigID] = att
	}

	totalRequired := len(requiredConfigs)
	totalApproved := 0
	totalPending := 0
	totalRejected := 0
	missingRequired := make([]dto.AttachmentConfigResponse, 0)

	for _, cfg := range requiredConfigs {
		att, exists := attachmentByConfig[cfg.ID]
		if !exists {
			// Belum diupload
			missingRequired = append(missingRequired, mapAttachmentConfigToResponse(cfg))
			continue
		}

		switch att.Status {
		case models.AttachmentStatusApproved:
			totalApproved++
		case models.AttachmentStatusPending:
			totalPending++
		case models.AttachmentStatusRejected:
			totalRejected++
		}
	}

	// Bisa lanjut kalau semua required sudah APPROVED dan tidak ada yang REJECTED
	canProceed := totalApproved == totalRequired &&
		totalRejected == 0 &&
		len(missingRequired) == 0

	return &dto.AttachmentStatusSummary{
		TransactionNumber: transactionNumber,
		Stage:             stage,
		CanProceed:        canProceed,
		TotalRequired:     totalRequired,
		TotalApproved:     totalApproved,
		TotalPending:      totalPending,
		TotalRejected:     totalRejected,
		MissingRequired:   missingRequired,
		Attachments:       mapTransactionAttachmentsToResponse(attachments),
	}, nil
}

// ============================================================
// HELPERS
// ============================================================

// getRequiredConfigs ambil attachment config dengan prioritas:
// 1. branch spesifik + stage spesifik
// 2. branch spesifik + stage ALL
// 3. branch ALL + stage spesifik
// 4. branch ALL + stage ALL
func getRequiredConfigs(transactionType, stage, branchCode string) ([]models.AttachmentConfig, error) {
	var configs []models.AttachmentConfig

	if err := config.DB.
		Where("transaction_type IN ? AND stage IN ? AND branch_code IN ? AND is_required = ? AND is_active = ?",
			[]string{transactionType, models.AttachmentTransactionTypeAll},
			[]string{stage, models.AttachmentStageAll},
			[]string{branchCode, models.AttachmentBranchAll},
			true, true,
		).
		Find(&configs).Error; err != nil {
		return nil, err
	}

	// Deduplikasi — kalau ada config spesifik dan ALL untuk attachment_type yang sama,
	// pakai yang lebih spesifik (branch spesifik > ALL, stage spesifik > ALL)
	seen := make(map[string]models.AttachmentConfig)
	for _, cfg := range configs {
		key := cfg.AttachmentType
		existing, exists := seen[key]
		if !exists {
			seen[key] = cfg
			continue
		}

		// Hitung spesifisitas (makin tinggi makin spesifik)
		score := func(c models.AttachmentConfig) int {
			s := 0
			if c.BranchCode != models.AttachmentBranchAll {
				s += 2
			}
			if c.Stage != models.AttachmentStageAll {
				s += 1
			}
			return s
		}

		if score(cfg) > score(existing) {
			seen[key] = cfg
		}
	}

	result := make([]models.AttachmentConfig, 0, len(seen))
	for _, cfg := range seen {
		result = append(result, cfg)
	}

	return result, nil
}

// sanitizePathSegment ganti karakter tidak aman jadi underscore
// Handles: / \ space dan karakter spesial lainnya
func sanitizePathSegment(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '/' || c == '\\' || c == ' ':
			result = append(result, '_')
		case c >= 32 && c < 127 && c != ':' && c != '*' && c != '?' && c != '"' && c != '<' && c != '>' && c != '|':
			result = append(result, c)
		default:
			result = append(result, '_')
		}
	}
	return string(result)
}

// detectMimeType deteksi mime type berdasarkan extension file
func detectMimeType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

// ============================================================
// MAPPERS
// ============================================================

func mapAttachmentConfigToResponse(cfg models.AttachmentConfig) dto.AttachmentConfigResponse {
	return dto.AttachmentConfigResponse{
		ID:              cfg.ID,
		TransactionType: cfg.TransactionType,
		Stage:           cfg.Stage,
		BranchCode:      cfg.BranchCode,
		AttachmentType:  cfg.AttachmentType,
		Description:     cfg.Description,
		IsRequired:      cfg.IsRequired,
		IsActive:        cfg.IsActive,
		CreatedBy:       cfg.CreatedBy,
		CreatedAt:       cfg.CreatedAt,
		UpdatedAt:       cfg.UpdatedAt,
	}
}

func mapAttachmentConfigsToResponse(configs []models.AttachmentConfig) []dto.AttachmentConfigResponse {
	result := make([]dto.AttachmentConfigResponse, len(configs))
	for i, cfg := range configs {
		result[i] = mapAttachmentConfigToResponse(cfg)
	}
	return result
}

func mapTransactionAttachmentToResponse(att models.TransactionAttachment) dto.TransactionAttachmentResponse {
	response := dto.TransactionAttachmentResponse{
		ID:                 att.ID,
		TransactionID:      att.TransactionID,
		TransactionNumber:  att.TransactionNumber,
		TransactionType:    att.TransactionType,
		Stage:              att.Stage,
		AttachmentConfigID: att.AttachmentConfigID,
		FileName:           att.FileName,
		FilePath:           att.FilePath,
		FileSize:           att.FileSize,
		MimeType:           att.MimeType,
		Status:             att.Status,
		UploadedBy:         att.UploadedBy,
		UploadedAt:         att.UploadedAt,
		ReviewedBy:         att.ReviewedBy,
		ReviewedAt:         att.ReviewedAt,
		RejectionReason:    att.RejectionReason,
		CreatedAt:          att.CreatedAt,
	}

	if att.AttachmentConfig != nil {
		response.AttachmentType = &att.AttachmentConfig.AttachmentType
		response.IsRequired = &att.AttachmentConfig.IsRequired
	}

	return response
}

func mapTransactionAttachmentsToResponse(atts []models.TransactionAttachment) []dto.TransactionAttachmentResponse {
	result := make([]dto.TransactionAttachmentResponse, len(atts))
	for i, att := range atts {
		result[i] = mapTransactionAttachmentToResponse(att)
	}
	return result
}
