package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// ============================================================
// EMAIL TEMPLATE CRUD
// Konsepnya sama dengan attachment_configs: satu baris per kombinasi
// (jenis transaksi, stage, aksi). Bedanya tidak ada nilai ALL — isi email
// selalu spesifik untuk satu langkah.
// ============================================================

// emailTemplateStages — stage yang boleh diberi template, per jenis transaksi.
// Stage akhir (FINISHED, REJECTED, CANCELLED) tidak punya aksi lanjutan.
// APPROVAL_AGREEMENT disposal juga tidak: transaksinya maju lewat agreement
// kolektif, bukan tombol aksi di detail transaksi.
var emailTemplateStages = map[string][]string{
	TxProcurement: {
		models.StageDraft,
		models.StageAssetVerification,
		models.StageApproval,
		models.StageProcessBudget,
		models.StageExecuteAsset,
		models.StageGR,
	},
	TxMutationFlow: {
		models.StageMutationDraft,
		models.StageMutationApproval,
		models.StageMutationReceiving,
		models.StageMutationExecute,
	},
	TxDisposalFlow: {
		models.StageDisposalDraft,
		models.StageDisposalPurchasing,
		models.StageDisposalApprovalRequest,
		models.StageDisposalExecute,
		models.StageDisposalFinance,
		models.StageDisposalTax,
		models.StageDisposalAssetDeletion,
	},
	TxHandover: {
		models.StageHandoverDraft,
		models.StageHandoverApproval,
		models.StageHandoverReceiving,
	},
	// Agreement: CREATE = saat agreement dibuat (belum punya stage),
	// APPROVAL_AGREEMENT = approve/tolak step agreement
	TxDisposalAgreement: {
		EmailStageAgreementCreate,
		models.StageAgreementApproval,
	},
}

func validateEmailTemplateStage(transactionType, stage string) error {
	for _, s := range emailTemplateStages[transactionType] {
		if s == stage {
			return nil
		}
	}
	return fmt.Errorf("stage %s tidak dikenal untuk transaksi %s", stage, transactionType)
}

// normalizeRoleIDs membuang duplikat/kosong dan memastikan semua role ada.
func normalizeRoleIDs(roleIDs []string) ([]string, error) {
	seen := map[string]bool{}
	unique := make([]string, 0, len(roleIDs))
	for _, id := range roleIDs {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return unique, nil
	}

	var count int64
	config.DB.Model(&models.Role{}).Where("id IN ?", unique).Count(&count)
	if int(count) != len(unique) {
		return nil, errors.New("sebagian role CC tidak ditemukan")
	}
	return unique, nil
}

func GetEmailTemplates(transactionType string) ([]dto.EmailTemplateResponse, error) {
	query := config.DB.Preload("CCRoles").Order("transaction_type, stage, action")
	if transactionType != "" {
		query = query.Where("transaction_type = ?", transactionType)
	}

	var templates []models.EmailTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, err
	}
	return mapEmailTemplatesToResponse(templates), nil
}

func GetEmailTemplateByID(id uint) (*dto.EmailTemplateResponse, error) {
	var tpl models.EmailTemplate
	if err := config.DB.Preload("CCRoles").First(&tpl, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("email template not found")
		}
		return nil, err
	}
	responses := mapEmailTemplatesToResponse([]models.EmailTemplate{tpl})
	return &responses[0], nil
}

func CreateEmailTemplate(userID string, req dto.CreateEmailTemplateRequest) (*dto.EmailTemplateResponse, error) {
	if err := validateEmailTemplateStage(req.TransactionType, req.Stage); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Subject) == "" || strings.TrimSpace(req.Body) == "" {
		return nil, errors.New("subject dan body wajib diisi")
	}

	roleIDs, err := normalizeRoleIDs(req.CCRoleIDs)
	if err != nil {
		return nil, err
	}
	// dialog email mewajibkan CC minimal 1, jadi template tanpa role CC
	// hanya akan memaksa user mengetik CC manual setiap kali
	if len(roleIDs) == 0 {
		return nil, errors.New("role CC minimal 1")
	}

	var existing int64
	config.DB.Model(&models.EmailTemplate{}).
		Where("transaction_type = ? AND stage = ? AND action = ?", req.TransactionType, req.Stage, req.Action).
		Count(&existing)
	if existing > 0 {
		return nil, errors.New("template untuk transaksi, stage, dan aksi ini sudah ada — ubah yang sudah ada")
	}

	tpl := models.EmailTemplate{
		TransactionType: req.TransactionType,
		Stage:           req.Stage,
		Action:          req.Action,
		Subject:         strings.TrimSpace(req.Subject),
		Body:            req.Body,
		IsActive:        req.IsActive,
		CreatedBy:       &userID,
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// is_active default:true di GORM membuat nilai false ter-skip saat
		// INSERT, jadi ditulis ulang secara eksplisit.
		if err := tx.Create(&tpl).Error; err != nil {
			return err
		}
		if !req.IsActive {
			if err := tx.Model(&tpl).Update("is_active", false).Error; err != nil {
				return err
			}
		}
		return replaceCCRoles(tx, tpl.ID, roleIDs)
	})
	if err != nil {
		return nil, err
	}

	return GetEmailTemplateByID(tpl.ID)
}

func UpdateEmailTemplate(id uint, req dto.UpdateEmailTemplateRequest) (*dto.EmailTemplateResponse, error) {
	var tpl models.EmailTemplate
	if err := config.DB.First(&tpl, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("email template not found")
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Subject != nil {
		subject := strings.TrimSpace(*req.Subject)
		if subject == "" {
			return nil, errors.New("subject tidak boleh kosong")
		}
		updates["subject"] = subject
	}
	if req.Body != nil {
		if strings.TrimSpace(*req.Body) == "" {
			return nil, errors.New("body tidak boleh kosong")
		}
		updates["body"] = *req.Body
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	var roleIDs []string
	if req.CCRoleIDs != nil {
		var err error
		if roleIDs, err = normalizeRoleIDs(*req.CCRoleIDs); err != nil {
			return nil, err
		}
		if len(roleIDs) == 0 {
			return nil, errors.New("role CC minimal 1")
		}
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&tpl).Updates(updates).Error; err != nil {
				return err
			}
		}
		// CC role hanya diganti kalau field-nya dikirim (partial update)
		if req.CCRoleIDs != nil {
			return replaceCCRoles(tx, tpl.ID, roleIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return GetEmailTemplateByID(tpl.ID)
}

func DeleteEmailTemplate(id uint) error {
	var tpl models.EmailTemplate
	if err := config.DB.First(&tpl, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("email template not found")
		}
		return err
	}
	// CC role ikut terhapus lewat ON DELETE CASCADE; log email tetap ada
	// (email_template_id jadi NULL).
	return config.DB.Delete(&tpl).Error
}

func replaceCCRoles(tx *gorm.DB, templateID uint, roleIDs []string) error {
	if err := tx.Where("email_template_id = ?", templateID).
		Delete(&models.EmailTemplateCCRole{}).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		row := models.EmailTemplateCCRole{EmailTemplateID: templateID, RoleID: roleID}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func mapEmailTemplatesToResponse(templates []models.EmailTemplate) []dto.EmailTemplateResponse {
	// nama role diambil sekali untuk seluruh daftar
	roleIDs := []string{}
	for _, tpl := range templates {
		for _, cc := range tpl.CCRoles {
			roleIDs = append(roleIDs, cc.RoleID)
		}
	}
	roleNames := map[string]string{}
	if len(roleIDs) > 0 {
		var roles []models.Role
		config.DB.Where("id IN ?", roleIDs).Find(&roles)
		for _, r := range roles {
			roleNames[r.ID] = r.Name
		}
	}

	result := make([]dto.EmailTemplateResponse, 0, len(templates))
	for _, tpl := range templates {
		ccRoles := make([]dto.EmailTemplateRoleResponse, 0, len(tpl.CCRoles))
		for _, cc := range tpl.CCRoles {
			ccRoles = append(ccRoles, dto.EmailTemplateRoleResponse{ID: cc.RoleID, Name: roleNames[cc.RoleID]})
		}
		result = append(result, dto.EmailTemplateResponse{
			ID:              tpl.ID,
			TransactionType: tpl.TransactionType,
			Stage:           tpl.Stage,
			Action:          tpl.Action,
			Subject:         tpl.Subject,
			Body:            tpl.Body,
			IsActive:        tpl.IsActive,
			CCRoles:         ccRoles,
			CreatedBy:       tpl.CreatedBy,
			CreatedAt:       tpl.CreatedAt,
			UpdatedAt:       tpl.UpdatedAt,
		})
	}
	return result
}
