package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ============================================================
// SERAH TERIMA ASET
//
// DRAFT → APPROVAL (flow ASSET_HANDOVER_APPROVAL) → HANDOVER_RECEIVING → FINISHED
//
// Aturan utama: hanya aset di cabang HOMEBASE pengaju, dan penerima (HANDOVER)
// harus ber-homebase di cabang yang sama. Aset lintas cabang wajib dimutasi
// dulu — itu disengaja supaya perpindahan antar cabang selalu tercatat.
//
// - HANDOVER: aset diserahkan ke user penerima; yang konfirmasi terima adalah
//   penerima itu sendiri
// - RETURN: aset yang sedang dipegang user dikembalikan ke cabang; yang
//   konfirmasi adalah user ber-permission confirm_receiving di cabang itu
//
// Selama ajuan aktif, aset dikunci dengan status IN_HANDOVER dan status
// sebelumnya disimpan di baris (previous_asset_status) untuk dipulihkan.
// ============================================================

const handoverConfirmRoute = "/transactions/handover/confirm-receiving"

func getHandoverTransaction(transactionNumber string) (*models.Transaction, error) {
	var transaction models.Transaction
	if err := config.DB.
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxHandover).
		First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("handover transaction not found")
		}
		return nil, err
	}
	return &transaction, nil
}

func handoverTypeOf(t *models.Transaction) string {
	if t.HandoverType == nil {
		return models.HandoverTypeHandover
	}
	return *t.HandoverType
}

// userHomebaseBranchCode — cabang homebase aktif; error kalau tidak punya.
func userHomebaseBranchCode(userID string) (string, error) {
	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return "", err
	}
	return homebase.Branch.BranchCode, nil
}

// activeHandoverAssetIDs — subquery asset_id yang sedang ikut ajuan serah
// terima lain yang masih berjalan.
func activeHandoverAssetIDs(excludeTransactionID uint) *gorm.DB {
	return config.DB.Model(&models.TransactionHandoverAsset{}).
		Select("asset_id").
		Where("status = ? AND transaction_id <> ?", models.HandoverAssetStatusPending, excludeTransactionID)
}

// eligibleHandoverAssetQuery — aset yang boleh dipilih untuk jenis serah
// terima tertentu di satu cabang.
func eligibleHandoverAssetQuery(handoverType, branchCode string, excludeTransactionID uint) *gorm.DB {
	query := config.DB.Model(&models.Asset{}).
		Where("deleted_at IS NULL AND branch_code = ?", branchCode).
		Where("id NOT IN (?)", activeHandoverAssetIDs(excludeTransactionID))

	if handoverType == models.HandoverTypeReturn {
		// yang dikembalikan: aset yang sedang dipegang user
		return query.Where("assigned_user_id IS NOT NULL").
			Where("asset_status IN ?", []string{models.AssetStatusAvailable, "ACTIVE", models.AssetStatusInHandover})
	}
	return query.Where("asset_status IN ?", []string{models.AssetStatusAvailable, "ACTIVE", models.AssetStatusInHandover})
}

// validateHandoverRecipient — penerima wajib user aktif ber-homebase di cabang
// yang sama dengan aset.
func validateHandoverRecipient(toUserID *string, branchCode, creatorID string) (string, error) {
	if toUserID == nil || strings.TrimSpace(*toUserID) == "" {
		return "", errors.New("recipient (to_user_id) is required for HANDOVER")
	}
	recipient := strings.TrimSpace(*toUserID)

	var user models.User
	if err := config.DB.Where("id = ? AND status = ?", recipient, "active").First(&user).Error; err != nil {
		return "", errors.New("recipient not found or inactive")
	}

	code, err := userHomebaseBranchCode(recipient)
	if err != nil || code != branchCode {
		return "", fmt.Errorf("recipient must have homebase in branch %s — for another branch, mutate the assets first", branchCode)
	}
	return recipient, nil
}

// replaceHandoverAssets — ganti isi baris aset draft. Aset yang dilepas
// dipulihkan statusnya; aset baru dikunci IN_HANDOVER.
func replaceHandoverAssets(tx *gorm.DB, transaction *models.Transaction, handoverType, branchCode, toUserID string, assetIDs []uint) error {
	unique := map[uint]bool{}
	ids := make([]uint, 0, len(assetIDs))
	for _, id := range assetIDs {
		if !unique[id] {
			unique[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return errors.New("at least one asset is required")
	}

	var assets []models.Asset
	if err := eligibleHandoverAssetQuery(handoverType, branchCode, transaction.ID).
		Where("id IN ?", ids).Find(&assets).Error; err != nil {
		return err
	}
	if len(assets) != len(ids) {
		return fmt.Errorf("some assets are not eligible: they must be in your homebase branch %s, not in another active transaction%s",
			branchCode,
			map[bool]string{true: ", and currently held by a user", false: ""}[handoverType == models.HandoverTypeReturn])
	}

	if handoverType == models.HandoverTypeHandover {
		for _, a := range assets {
			if a.AssignedUserID != nil && *a.AssignedUserID == toUserID {
				return fmt.Errorf("asset %s is already held by the recipient", a.AssetNumber)
			}
		}
	}

	// baris lama
	var existing []models.TransactionHandoverAsset
	if err := tx.Where("transaction_id = ? AND status = ?", transaction.ID, models.HandoverAssetStatusPending).
		Find(&existing).Error; err != nil {
		return err
	}
	keep := map[uint]models.TransactionHandoverAsset{}
	for _, row := range existing {
		if unique[row.AssetID] {
			keep[row.AssetID] = row
			continue
		}
		// dilepas dari draft → pulihkan status aset, hapus barisnya
		if err := tx.Model(&models.Asset{}).Where("id = ?", row.AssetID).
			Update("asset_status", row.PreviousAssetStatus).Error; err != nil {
			return err
		}
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
	}

	for _, a := range assets {
		if _, ok := keep[a.ID]; ok {
			continue
		}
		previous := a.AssetStatus
		if previous == models.AssetStatusInHandover {
			// sudah dikunci oleh draft ini sebelumnya (mis. revisi) — tidak
			// seharusnya terjadi karena barisnya tersimpan, tapi jangan sampai
			// status "IN_HANDOVER" tersimpan sebagai status asal
			previous = models.AssetStatusAvailable
		}
		row := models.TransactionHandoverAsset{
			TransactionID:       transaction.ID,
			TransactionNumber:   transaction.TransactionNumber,
			AssetID:             a.ID,
			AssetNumber:         a.AssetNumber,
			FromUserID:          a.AssignedUserID,
			PreviousAssetStatus: previous,
			Status:              models.HandoverAssetStatusPending,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Asset{}).Where("id = ?", a.ID).
			Update("asset_status", models.AssetStatusInHandover).Error; err != nil {
			return err
		}
	}
	return nil
}

// releaseHandoverAssets — ajuan batal/ditolak: status aset dipulihkan, baris
// CANCELLED.
func releaseHandoverAssets(tx *gorm.DB, transactionID uint) error {
	var rows []models.TransactionHandoverAsset
	if err := tx.Where("transaction_id = ? AND status = ?", transactionID, models.HandoverAssetStatusPending).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := tx.Model(&models.Asset{}).Where("id = ?", row.AssetID).
			Update("asset_status", row.PreviousAssetStatus).Error; err != nil {
			return err
		}
		if err := tx.Model(&row).Update("status", models.HandoverAssetStatusCancelled).Error; err != nil {
			return err
		}
	}
	return nil
}

// ============================================================
// CREATE / UPDATE DRAFT
// ============================================================

func CreateHandoverDraft(userID string, req dto.CreateHandoverRequest) (*dto.HandoverDetailResponse, error) {
	branchCode, err := userHomebaseBranchCode(userID)
	if err != nil {
		return nil, err
	}

	transactionDate := time.Now()
	if strings.TrimSpace(req.TransactionDate) != "" {
		if transactionDate, err = time.Parse("2006-01-02", req.TransactionDate); err != nil {
			return nil, errors.New("invalid transaction date format, use YYYY-MM-DD")
		}
	}

	var recipient *string
	if req.HandoverType == models.HandoverTypeHandover {
		id, err := validateHandoverRecipient(req.ToUserID, branchCode, userID)
		if err != nil {
			return nil, err
		}
		recipient = &id
	}

	transactionNumber, err := GenerateTransactionNumber(userID, TxHandover)
	if err != nil {
		return nil, err
	}

	handoverType := req.HandoverType
	transaction := models.Transaction{
		TransactionNumber: transactionNumber,
		TransactionType:   TxHandover,
		TransactionDate:   transactionDate,
		Status:            models.TransactionStatusDraft,
		CurrentStage:      models.StageHandoverDraft,
		Notes:             req.Notes,
		CreatedBy:         userID,
		HandoverType:      &handoverType,
		HandoverToUserID:  recipient,
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&transaction).Error; err != nil {
			return err
		}
		to := ""
		if recipient != nil {
			to = *recipient
		}
		return replaceHandoverAssets(tx, &transaction, handoverType, branchCode, to, req.AssetIDs)
	})
	if err != nil {
		return nil, err
	}

	return GetHandoverDetail(transactionNumber, userID)
}

func UpdateHandoverDraft(userID, transactionNumber string, req dto.UpdateHandoverDraftRequest) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only modify your own handover draft")
	}
	if transaction.CurrentStage != models.StageHandoverDraft {
		return nil, errors.New("only DRAFT handovers can be modified")
	}

	branchCode, err := userHomebaseBranchCode(userID)
	if err != nil {
		return nil, err
	}

	handoverType := handoverTypeOf(transaction)
	updates := map[string]interface{}{"notes": req.Notes}
	to := ""
	if handoverType == models.HandoverTypeHandover {
		if to, err = validateHandoverRecipient(req.ToUserID, branchCode, userID); err != nil {
			return nil, err
		}
		updates["handover_to_user_id"] = to
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(transaction).Updates(updates).Error; err != nil {
			return err
		}
		return replaceHandoverAssets(tx, transaction, handoverType, branchCode, to, req.AssetIDs)
	})
	if err != nil {
		return nil, err
	}

	return GetHandoverDetail(transactionNumber, userID)
}

// ============================================================
// SUBMIT — DRAFT → APPROVAL (approval langsung dibentuk)
// ============================================================

func resolveHandoverApprovalFlow(transaction *models.Transaction) (*dto.ApprovalFlowResponse, error) {
	flow, err := GetApprovalFlowByCodeAndBranch(models.FlowAssetHandoverApproval, GetCreatorBranchCode(transaction.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("approval flow %s not found, please configure it first", models.FlowAssetHandoverApproval)
	}
	if !flow.IsActive {
		return nil, fmt.Errorf("approval flow %s is inactive", models.FlowAssetHandoverApproval)
	}
	if len(flow.FlowSteps) == 0 {
		return nil, fmt.Errorf("approval flow %s has no steps configured", models.FlowAssetHandoverApproval)
	}
	return flow, nil
}

func SubmitHandover(userID, transactionNumber string, req dto.HandoverActionRequest) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CreatedBy != userID {
		return nil, errors.New("you can only submit your own handovers")
	}
	if transaction.CurrentStage != models.StageHandoverDraft {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageHandoverDraft)
	}

	var assetCount int64
	config.DB.Model(&models.TransactionHandoverAsset{}).
		Where("transaction_id = ? AND status = ?", transaction.ID, models.HandoverAssetStatusPending).
		Count(&assetCount)
	if assetCount == 0 {
		return nil, errors.New("cannot submit handover with no assets")
	}

	// dokumen DRAFT (kalau dikonfigurasi) cukup sudah diunggah
	if err := checkAttachmentCanProceed(transactionNumber, TxHandover, models.StageHandoverDraft,
		GetCreatorBranchCode(transaction.CreatedBy)); err != nil {
		return nil, err
	}

	// flow dicek sebelum stage pindah — jangan sampai transaksi terlanjur di
	// APPROVAL tanpa baris approval
	flow, err := resolveHandoverApprovalFlow(transaction)
	if err != nil {
		return nil, err
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, transaction, models.StageHandoverApproval); err != nil {
			return err
		}
		// revisi dianggap selesai begitu disubmit ulang
		if err := tx.Model(&models.TransactionHandoverAsset{}).
			Where("transaction_id = ?", transaction.ID).
			Updates(map[string]interface{}{"needs_revision": false, "revision_notes": nil}).Error; err != nil {
			return err
		}
		return recordStage(tx, transaction.ID, transactionNumber,
			fromStage, models.StageHandoverApproval, models.ActionSubmit, userID, nil, req.Notes)
	})
	if err != nil {
		return nil, err
	}

	if err := InitiateTransactionApproval(dto.CreateTransactionApprovalRequest{
		FlowID:            flow.ID,
		TransactionNumber: transactionNumber,
		TransactionType:   TxHandover,
	}); err != nil {
		return nil, fmt.Errorf("handover submitted but approval initiation failed: %w", err)
	}

	return GetHandoverDetail(transactionNumber, userID)
}

// autoCompleteHandoverApproval — semua step approved → HANDOVER_RECEIVING
func autoCompleteHandoverApproval(userID, transactionNumber, transactionType string) error {
	if transactionType != TxHandover {
		return nil
	}

	var total, approved int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ?", transactionNumber, transactionType).
		Count(&total)
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND status = ?", transactionNumber, transactionType, "approved").
		Count(&approved)
	if total == 0 || approved < total {
		return nil
	}

	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return err
	}
	if transaction.CurrentStage != models.StageHandoverApproval {
		return nil
	}

	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := updateTransactionStage(tx, transaction, models.StageHandoverReceiving); err != nil {
			return err
		}
		return recordStage(tx, transaction.ID, transactionNumber,
			models.StageHandoverApproval, models.StageHandoverReceiving, models.ActionApprove, userID, nil, nil)
	})
}

// autoRejectHandover — step approval ditolak → REJECTED, aset dilepas
func autoRejectHandover(userID, transactionNumber, transactionType, notes string) error {
	if transactionType != TxHandover {
		return nil
	}
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return err
	}
	if transaction.CurrentStage != models.StageHandoverApproval {
		return nil
	}
	reason := "Rejected by approver"
	if notes != "" {
		reason = notes
	}
	return terminateHandover(userID, transaction, models.StageHandoverRejected, models.ActionReject, reason)
}

func terminateHandover(userID string, transaction *models.Transaction, toStage, action, reason string) error {
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := releaseHandoverAssets(tx, transaction.ID); err != nil {
			return err
		}
		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, transaction, toStage); err != nil {
			return err
		}
		return recordStage(tx, transaction.ID, transaction.TransactionNumber,
			fromStage, toStage, action, userID, nil, &reason)
	})
	if err == nil {
		MarkTransactionAsExpired(transaction.TransactionNumber)
	}
	return err
}

// ============================================================
// KONFIRMASI TERIMA — HANDOVER_RECEIVING → FINISHED
// ============================================================

// canConfirmHandover — HANDOVER: penerimanya sendiri. RETURN: user dengan
// confirm_receiving di menu serah terima dan ber-homebase di cabang aset.
func canConfirmHandover(userID string, transaction *models.Transaction) bool {
	if userID == "" || transaction.CurrentStage != models.StageHandoverReceiving {
		return false
	}
	if handoverTypeOf(transaction) == models.HandoverTypeHandover {
		return transaction.HandoverToUserID != nil && *transaction.HandoverToUserID == userID
	}
	if !userHasMenuPermission(userRoleIDs(userID), handoverConfirmRoute, []string{"confirm_receiving"}) {
		return false
	}
	code, err := userHomebaseBranchCode(userID)
	return err == nil && code == GetCreatorBranchCode(transaction.CreatedBy)
}

func ConfirmHandoverReceiving(userID, transactionNumber string, req dto.HandoverActionRequest) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CurrentStage != models.StageHandoverReceiving {
		return nil, fmt.Errorf("transaction is not in %s stage", models.StageHandoverReceiving)
	}
	if !canConfirmHandover(userID, transaction) {
		if handoverTypeOf(transaction) == models.HandoverTypeHandover {
			return nil, errors.New("only the recipient can confirm this handover")
		}
		return nil, errors.New("only users with confirm_receiving permission in this branch can confirm the return")
	}

	// dokumen serah terima (BAST) cukup sudah diunggah — diunggah oleh pihak
	// yang konfirmasi sendiri, jadi tidak ada reviewer terpisah
	summary, err := GetAttachmentStatusSummary(transactionNumber, TxHandover, models.StageHandoverReceiving,
		GetCreatorBranchCode(transaction.CreatedBy))
	if err != nil {
		return nil, err
	}
	if len(summary.MissingRequired) > 0 {
		missing := make([]string, 0, len(summary.MissingRequired))
		for _, m := range summary.MissingRequired {
			missing = append(missing, m.AttachmentType)
		}
		return nil, fmt.Errorf("required documents not yet uploaded: %s", strings.Join(missing, ", "))
	}
	if summary.TotalRejected > 0 {
		return nil, errors.New("some documents were rejected, please re-upload them")
	}

	var rows []models.TransactionHandoverAsset
	if err := config.DB.Where("transaction_id = ? AND status = ?", transaction.ID, models.HandoverAssetStatusPending).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	handoverType := handoverTypeOf(transaction)
	now := time.Now()

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			updates := map[string]interface{}{"asset_status": row.PreviousAssetStatus}
			var newHolder *string
			if handoverType == models.HandoverTypeHandover {
				newHolder = transaction.HandoverToUserID
				updates["assigned_user_id"] = *transaction.HandoverToUserID
				updates["assigned_at"] = now
			} else {
				updates["assigned_user_id"] = nil
				updates["assigned_at"] = nil
			}
			if err := tx.Model(&models.Asset{}).Where("id = ?", row.AssetID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Model(&row).Update("status", models.HandoverAssetStatusCompleted).Error; err != nil {
				return err
			}

			before, _ := json.Marshal(map[string]interface{}{"assigned_user_id": row.FromUserID})
			after, _ := json.Marshal(map[string]interface{}{"assigned_user_id": newHolder, "handover_type": handoverType})
			beforeStr, afterStr := string(before), string(after)
			transactionID := transaction.ID
			if err := CreateAssetHistory(tx, models.AssetHistory{
				AssetID:         row.AssetID,
				TransactionType: TxHandover,
				TransactionID:   &transactionID,
				DocumentNumber:  &transaction.TransactionNumber,
				TransactionDate: &now,
				BeforeData:      &beforeStr,
				AfterData:       &afterStr,
				ChangedBy:       &userID,
			}); err != nil {
				return err
			}
		}

		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, transaction, models.StageHandoverFinished); err != nil {
			return err
		}
		return recordStage(tx, transaction.ID, transactionNumber,
			fromStage, models.StageHandoverFinished, models.ActionGR, userID, nil, req.Notes)
	})
	if err != nil {
		return nil, err
	}

	return GetHandoverDetail(transactionNumber, userID)
}

// RejectHandoverReceiving — pihak penerima menolak menerima (barang tidak
// sesuai, dsb). Ajuan berakhir REJECTED dan aset dilepas.
func RejectHandoverReceiving(userID, transactionNumber string, req dto.RejectHandoverReceivingRequest) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if !canConfirmHandover(userID, transaction) {
		return nil, errors.New("only the receiving party can reject at this stage")
	}
	if err := terminateHandover(userID, transaction, models.StageHandoverRejected, models.ActionReject, req.Reason); err != nil {
		return nil, err
	}
	return GetHandoverDetail(transactionNumber, userID)
}

// ============================================================
// REVISI & BATAL
// ============================================================

func ReturnHandoverForRevision(userID, transactionNumber string, req dto.ReturnForRevisionRequest) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CurrentStage != models.StageHandoverApproval {
		return nil, fmt.Errorf("cannot revise transaction in %s stage", transaction.CurrentStage)
	}
	if err := assertCurrentApprover(userID, transactionNumber, TxHandover); err != nil {
		return nil, err
	}

	var targets []models.TransactionHandoverAsset
	if err := config.DB.Where("transaction_id = ? AND id IN ? AND status = ?",
		transaction.ID, req.RowIDs, models.HandoverAssetStatusPending).Find(&targets).Error; err != nil {
		return nil, err
	}
	if len(targets) != len(req.RowIDs) {
		return nil, errors.New("some selected assets do not belong to this transaction or are no longer active")
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := markRowsForRevision(tx, &models.TransactionHandoverAsset{}, transaction.ID, req.RowIDs, req.RevisionNotes); err != nil {
			return err
		}
		if err := resetApprovalsForRevision(tx, transactionNumber, TxHandover); err != nil {
			return err
		}
		fromStage := transaction.CurrentStage
		if err := updateTransactionStage(tx, transaction, models.StageHandoverDraft); err != nil {
			return err
		}
		return recordStage(tx, transaction.ID, transactionNumber,
			fromStage, models.StageHandoverDraft, models.ActionRevise, userID, nil, &req.RevisionNotes)
	})
	if err != nil {
		return nil, err
	}
	return GetHandoverDetail(transactionNumber, userID)
}

func CancelHandover(userID, transactionNumber string, req dto.CancelTransactionRequest) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}
	if transaction.CreatedBy != userID {
		return nil, errors.New("only the creator can cancel this transaction")
	}
	if transaction.CurrentStage != models.StageHandoverDraft && transaction.CurrentStage != models.StageHandoverApproval {
		return nil, fmt.Errorf("cannot cancel transaction in %s stage", transaction.CurrentStage)
	}

	if err := config.DB.Where("transaction_number = ? AND transaction_type = ?", transactionNumber, TxHandover).
		Delete(&models.TransactionApproval{}).Error; err != nil {
		return nil, err
	}
	if err := terminateHandover(userID, transaction, models.StageHandoverCancelled, models.ActionCancel, req.Reason); err != nil {
		return nil, err
	}
	return GetHandoverDetail(transactionNumber, userID)
}

// ============================================================
// READ
// ============================================================

func GetHandoverDetail(transactionNumber, viewerID string) (*dto.HandoverDetailResponse, error) {
	transaction, err := getHandoverTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	var rows []models.TransactionHandoverAsset
	config.DB.Preload("Asset.Category").
		Where("transaction_id = ?", transaction.ID).
		Order("id").Find(&rows)

	userIDs := []string{}
	for _, row := range rows {
		if row.FromUserID != nil {
			userIDs = append(userIDs, *row.FromUserID)
		}
	}
	names := resolveUserFullnames(userIDs)

	assets := make([]dto.HandoverAssetResponse, 0, len(rows))
	active := 0
	for _, row := range rows {
		item := dto.HandoverAssetResponse{
			ID:                  row.ID,
			AssetID:             row.AssetID,
			AssetNumber:         row.AssetNumber,
			FromUserID:          row.FromUserID,
			PreviousAssetStatus: row.PreviousAssetStatus,
			Status:              row.Status,
			NeedsRevision:       row.NeedsRevision,
			RevisionNotes:       row.RevisionNotes,
		}
		if row.Asset != nil {
			item.AssetName = row.Asset.AssetName
			item.BranchCode = row.Asset.BranchCode
			if row.Asset.Category != nil {
				item.CategoryName = &row.Asset.Category.CategoryName
			}
		}
		if row.FromUserID != nil {
			if n, ok := names[*row.FromUserID]; ok {
				name := n
				item.FromUserName = &name
			}
		}
		if row.Status != models.HandoverAssetStatusCancelled {
			active++
		}
		assets = append(assets, item)
	}

	var stages []models.TransactionStage
	config.DB.Where("transaction_id = ?", transaction.ID).Order("created_at ASC").Find(&stages)

	response := &dto.HandoverDetailResponse{
		Transaction:         mapTransactionHeaderToResponse(*transaction),
		HandoverType:        handoverTypeOf(transaction),
		ToUserID:            transaction.HandoverToUserID,
		BranchCode:          GetCreatorBranchCode(transaction.CreatedBy),
		Assets:              assets,
		Stages:              mapTransactionStagesToResponse(stages),
		TotalAssets:         active,
		WaitingForMe:        isWaitingForUser(viewerID, transaction, handoverWaiting),
		CanConfirmReceiving: canConfirmHandover(viewerID, transaction),
	}
	response.NeedsRevision = response.Transaction.NeedsRevision
	if transaction.HandoverToUserID != nil {
		response.ToUserName = resolveUserFullname(*transaction.HandoverToUserID)
	}
	return response, nil
}

func GetAllHandovers(filter dto.HandoverListFilter) ([]dto.HandoverListItem, int64, error) {
	query := config.DB.Model(&models.Transaction{}).Where("transaction_type = ?", TxHandover)

	if filter.CurrentStage != nil && strings.TrimSpace(*filter.CurrentStage) != "" {
		stages := []string{}
		for _, s := range strings.Split(*filter.CurrentStage, ",") {
			if s = strings.TrimSpace(s); s != "" {
				stages = append(stages, s)
			}
		}
		query = query.Where("current_stage IN ?", stages)
	}
	if filter.HandoverType != nil && *filter.HandoverType != "" {
		query = query.Where("handover_type = ?", *filter.HandoverType)
	}
	if filter.StartDate != nil && *filter.StartDate != "" {
		query = query.Where("transaction_date >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil && *filter.EndDate != "" {
		query = query.Where("transaction_date <= ?", *filter.EndDate)
	}
	if filter.Search != nil && strings.TrimSpace(*filter.Search) != "" {
		keyword := "%" + strings.TrimSpace(*filter.Search) + "%"
		assetMatches := config.DB.Model(&models.TransactionHandoverAsset{}).
			Select("transaction_handover_assets.transaction_id").
			Joins("LEFT JOIN assets ON assets.id = transaction_handover_assets.asset_id").
			Where("transaction_handover_assets.asset_number LIKE ? OR assets.asset_name LIKE ?", keyword, keyword)
		query = query.Where(config.DB.Where("transaction_number LIKE ?", keyword).
			Or("notes LIKE ?", keyword).
			Or("id IN (?)", assetMatches))
	}
	if filter.WaitingForMe && filter.ViewerUserID != "" {
		query = applyWaitingFilter(query, filter.ViewerUserID, handoverWaiting)
	}

	page, limit := filter.Page, filter.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var transactions []models.Transaction
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	userIDs := []string{}
	txIDs := []uint{}
	for _, t := range transactions {
		userIDs = append(userIDs, t.CreatedBy)
		if t.HandoverToUserID != nil {
			userIDs = append(userIDs, *t.HandoverToUserID)
		}
		txIDs = append(txIDs, t.ID)
	}
	names := resolveUserFullnames(userIDs)

	type countRow struct {
		TransactionID uint
		N             int
	}
	var counts []countRow
	if len(txIDs) > 0 {
		config.DB.Model(&models.TransactionHandoverAsset{}).
			Select("transaction_id, COUNT(*) AS n").
			Where("transaction_id IN ? AND status <> ?", txIDs, models.HandoverAssetStatusCancelled).
			Group("transaction_id").Scan(&counts)
	}
	countOf := map[uint]int{}
	for _, c := range counts {
		countOf[c.TransactionID] = c.N
	}

	items := make([]dto.HandoverListItem, 0, len(transactions))
	for _, t := range transactions {
		item := dto.HandoverListItem{
			TransactionNumber: t.TransactionNumber,
			TransactionDate:   t.TransactionDate,
			CurrentStage:      t.CurrentStage,
			Status:            t.Status,
			HandoverType:      handoverTypeOf(&t),
			CreatedBy:         t.CreatedBy,
			BranchCode:        GetCreatorBranchCode(t.CreatedBy),
			TotalAssets:       countOf[t.ID],
			NeedsRevision:     transactionNeedsRevision(t.ID, TxHandover),
			CreatedAt:         t.CreatedAt,
		}
		if n, ok := names[t.CreatedBy]; ok {
			name := n
			item.CreatedByName = &name
		}
		if t.HandoverToUserID != nil {
			if n, ok := names[*t.HandoverToUserID]; ok {
				name := n
				item.ToUserName = &name
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

// GetHandoverEligibleAssets — aset di homebase user yang bisa dipilih.
// excludeNumber: draft yang sedang diubah (asetnya sendiri tetap boleh).
func GetHandoverEligibleAssets(userID, handoverType, search, excludeNumber string) ([]dto.HandoverEligibleAsset, error) {
	if handoverType != models.HandoverTypeHandover && handoverType != models.HandoverTypeReturn {
		return nil, errors.New("handover_type must be HANDOVER or RETURN")
	}
	branchCode, err := userHomebaseBranchCode(userID)
	if err != nil {
		return nil, err
	}

	var excludeID uint
	if excludeNumber != "" {
		if t, err := getHandoverTransaction(excludeNumber); err == nil {
			excludeID = t.ID
		}
	}

	query := eligibleHandoverAssetQuery(handoverType, branchCode, excludeID)
	if excludeID == 0 {
		// aset IN_HANDOVER hanya relevan untuk draft yang sedang diubah
		query = query.Where("asset_status <> ?", models.AssetStatusInHandover)
	} else {
		query = query.Where("asset_status <> ? OR id IN (?)", models.AssetStatusInHandover,
			config.DB.Model(&models.TransactionHandoverAsset{}).Select("asset_id").
				Where("transaction_id = ? AND status = ?", excludeID, models.HandoverAssetStatusPending))
	}
	if s := strings.TrimSpace(search); s != "" {
		keyword := "%" + s + "%"
		query = query.Where("(asset_number LIKE ? OR asset_name LIKE ?)", keyword, keyword)
	}

	var assets []models.Asset
	if err := query.Preload("Category").Order("asset_number").Limit(500).Find(&assets).Error; err != nil {
		return nil, err
	}

	holderIDs := []string{}
	for _, a := range assets {
		if a.AssignedUserID != nil {
			holderIDs = append(holderIDs, *a.AssignedUserID)
		}
	}
	names := resolveUserFullnames(holderIDs)

	result := make([]dto.HandoverEligibleAsset, 0, len(assets))
	for _, a := range assets {
		item := dto.HandoverEligibleAsset{
			AssetID:        a.ID,
			AssetNumber:    a.AssetNumber,
			AssetName:      a.AssetName,
			AssetStatus:    a.AssetStatus,
			Location:       a.Location,
			AssignedUserID: a.AssignedUserID,
		}
		if a.Category != nil {
			item.CategoryName = &a.Category.CategoryName
		}
		if a.AssignedUserID != nil {
			if n, ok := names[*a.AssignedUserID]; ok {
				name := n
				item.AssignedUserName = &name
			}
		}
		result = append(result, item)
	}
	return result, nil
}

// GetHandoverRecipients — user aktif ber-homebase di cabang homebase peminta.
func GetHandoverRecipients(userID string) ([]dto.HandoverRecipient, error) {
	homebase, err := GetUserActiveHomebase(userID)
	if err != nil {
		return nil, err
	}

	var memberships []models.UserBranch
	config.DB.Where("branch_id = ? AND branch_type = ? AND is_active = ?", homebase.BranchID, "homebase", true).
		Find(&memberships)

	ids := make([]string, 0, len(memberships))
	for _, m := range memberships {
		ids = append(ids, m.UserID)
	}
	if len(ids) == 0 {
		return []dto.HandoverRecipient{}, nil
	}

	var users []models.User
	config.DB.Where("id IN ? AND status = ?", ids, "active").Order("fullname").Find(&users)

	result := make([]dto.HandoverRecipient, 0, len(users))
	for _, u := range users {
		result = append(result, dto.HandoverRecipient{UserID: u.ID, Fullname: u.Fullname, Username: u.Username, Email: u.Email})
	}
	return result, nil
}

// assertAssetNotHeld — aset yang sedang dipegang user tidak boleh dimutasi
// atau didisposal sebelum dikembalikan ke cabang lewat serah terima RETURN.
func assertAssetNotHeld(asset *models.Asset) error {
	if asset.AssignedUserID == nil {
		return nil
	}
	holder := *asset.AssignedUserID
	if name := resolveUserFullname(holder); name != nil {
		holder = *name
	}
	return fmt.Errorf("asset %s is currently held by %s — return it to the branch first (Asset Handover → Return)",
		asset.AssetNumber, holder)
}
