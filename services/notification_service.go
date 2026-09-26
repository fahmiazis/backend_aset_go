package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"sort"
	"time"
)

// ============================================================
// Lonceng notifikasi navbar
//
// Sengaja TIDAK memakai tabel notifikasi. Isinya adalah semua pengajuan yang
// "Menunggu Saya" (waiting_service.go) di keempat alur, dihitung ulang setiap
// diminta. Konsekuensinya sesuai kebutuhan:
//   - membuka detail tidak menghapus notif — baru hilang setelah aksinya
//     dikerjakan (oleh siapa pun yang berhak)
//   - tidak ada status "sudah dibaca" yang bisa tidak sinkron dengan
//     keadaan transaksi sebenarnya
// ============================================================

const notificationLimit = 50

func waitingTransactions(userID, txType string, cfg waitingConfig) []models.Transaction {
	var rows []models.Transaction
	applyWaitingFilter(
		config.DB.Model(&models.Transaction{}).Where("transaction_type = ?", txType),
		userID, cfg,
	).Find(&rows)
	return rows
}

// stageEnteredAt — waktu transaksi masuk ke stage saat ini (perpindahan
// stage terakhir), jatuh ke created_at untuk DRAFT yang belum pernah pindah.
func stageEnteredAt(transactionIDs []uint) map[uint]time.Time {
	result := map[uint]time.Time{}
	if len(transactionIDs) == 0 {
		return result
	}
	type row struct {
		TransactionID uint
		Latest        time.Time
	}
	var rows []row
	config.DB.Model(&models.TransactionStage{}).
		Select("transaction_id, MAX(created_at) AS latest").
		Where("transaction_id IN ?", transactionIDs).
		Group("transaction_id").
		Scan(&rows)
	for _, r := range rows {
		result[r.TransactionID] = r.Latest
	}
	return result
}

// lastApprovalAt — giliran approval berpindah saat step sebelumnya disetujui,
// yang tidak tercatat sebagai perpindahan stage.
func lastApprovalAt(numbers []string, txType string) map[string]time.Time {
	result := map[string]time.Time{}
	if len(numbers) == 0 {
		return result
	}
	type row struct {
		TransactionNumber string
		Latest            *time.Time
	}
	var rows []row
	config.DB.Model(&models.TransactionApproval{}).
		Select("transaction_number, MAX(approved_at) AS latest").
		Where("transaction_number IN ? AND transaction_type = ? AND status = ?", numbers, txType, "approved").
		Group("transaction_number").
		Scan(&rows)
	for _, r := range rows {
		if r.Latest != nil {
			result[r.TransactionNumber] = *r.Latest
		}
	}
	return result
}

func GetWaitingNotifications(userID string) (*dto.WaitingNotificationResponse, error) {
	flows := []struct {
		txType string
		cfg    waitingConfig
	}{
		{TxProcurement, procurementWaiting},
		{TxMutationFlow, mutationWaiting},
		{TxDisposalFlow, disposalWaiting},
	}

	items := []dto.WaitingNotification{}
	creatorIDs := []string{}
	creatorOf := map[int]string{}

	for _, flow := range flows {
		rows := waitingTransactions(userID, flow.txType, flow.cfg)

		ids := make([]uint, 0, len(rows))
		numbers := make([]string, 0, len(rows))
		for _, trx := range rows {
			ids = append(ids, trx.ID)
			numbers = append(numbers, trx.TransactionNumber)
		}
		entered := stageEnteredAt(ids)
		approved := lastApprovalAt(numbers, flow.txType)

		for _, trx := range rows {
			since := trx.CreatedAt
			if t, ok := entered[trx.ID]; ok && t.After(since) {
				since = t
			}
			if t, ok := approved[trx.TransactionNumber]; ok && t.After(since) {
				since = t
			}
			creatorOf[len(items)] = trx.CreatedBy
			creatorIDs = append(creatorIDs, trx.CreatedBy)
			items = append(items, dto.WaitingNotification{
				TransactionType:   flow.txType,
				TransactionNumber: trx.TransactionNumber,
				CurrentStage:      trx.CurrentStage,
				Since:             since,
			})
		}
	}

	// Disposal agreement: giliran step approval milik role user, dan user
	// memegang seluruh cabang anggotanya (tanpa itu approve-nya pasti ditolak).
	agreementNumbers := myTurnApprovalNumbers(userRoleIDs(userID), TxDisposalAgreement)
	if len(agreementNumbers) > 0 {
		var agreements []models.DisposalAgreement
		config.DB.
			Where("agreement_number IN ? AND current_stage = ?", agreementNumbers, models.StageAgreementApproval).
			Find(&agreements)
		approved := lastApprovalAt(agreementNumbers, TxDisposalAgreement)

		for _, agreement := range agreements {
			if validateAgreementApproverBranches(userID, agreement.AgreementNumber) != nil {
				continue
			}
			since := agreement.CreatedAt
			if t, ok := approved[agreement.AgreementNumber]; ok && t.After(since) {
				since = t
			}
			creatorOf[len(items)] = agreement.CreatedBy
			creatorIDs = append(creatorIDs, agreement.CreatedBy)
			items = append(items, dto.WaitingNotification{
				TransactionType:   TxDisposalAgreement,
				TransactionNumber: agreement.AgreementNumber,
				CurrentStage:      agreement.CurrentStage,
				Since:             since,
			})
		}
	}

	names := resolveUserFullnames(creatorIDs)
	for i := range items {
		if name, ok := names[creatorOf[i]]; ok {
			n := name
			items[i].CreatedByName = &n
		}
	}

	// yang paling baru masuk giliran user ditaruh paling atas
	sort.SliceStable(items, func(i, j int) bool { return items[i].Since.After(items[j].Since) })

	total := len(items)
	if len(items) > notificationLimit {
		items = items[:notificationLimit]
	}
	return &dto.WaitingNotificationResponse{Total: total, Items: items}, nil
}
