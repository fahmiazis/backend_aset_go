package services

import (
	"backend-go/config"
	"backend-go/models"

	"gorm.io/gorm"
)

// ============================================================
// "Menunggu saya" — pengajuan yang bolanya ada di user ini
//
// Dipakai disposal, procurement, dan mutation. Aturannya satu, yang berbeda
// hanya pemetaan stage ke endpoint pengerjanya (lihat waitingConfig di
// *_waiting.go masing-masing flow).
// ============================================================

// stageOwner memetakan stage ke endpoint yang mengerjakannya, beserta
// permission yang dibutuhkan endpoint itu. Nilainya harus sama dengan
// RequirePermission di routes dan route_path di tabel menus — dari situlah hak
// akses user dibaca.
type stageOwner struct {
	routePath string
	// Cukup salah satu terpenuhi, sama dengan middleware.RequirePermission yang
	// meloloskan request kalau user punya salah satu permission yang diminta.
	permissions []string
	// true kalau stage ini dikerjakan cabang TUJUAN transaksi, bukan cabang
	// pengaju — mis. konfirmasi penerimaan mutasi.
	byDestinationBranch bool
	// opsional: subquery id transaksi di stage ini yang sudah BUKAN tugas
	// pemilik stage walau stage-nya belum berubah (mis. disposal yang sudah
	// masuk agreement aktif)
	excludeIDs func() *gorm.DB
}

type waitingConfig struct {
	// transaction_type pada tabel transactions dan transaction_approvals
	txType string
	// stage yang hanya bisa dilanjutkan pembuatnya
	draftStage string
	owners     map[string]stageOwner

	// Opsional: tugas yang dipegang user TERTENTU, bukan role — mis. penerima
	// serah terima aset. extraCondition dipakai filter daftar, extraCheck
	// dipakai halaman detail; keduanya harus mencerminkan aturan yang sama.
	extraCondition func(userID string) *gorm.DB
	extraCheck     func(userID string, transaction *models.Transaction) bool
}

func userRoleIDs(userID string) []string {
	var userRoles []models.UserRole
	config.DB.Where("user_id = ?", userID).Find(&userRoles)

	ids := make([]string, 0, len(userRoles))
	for _, ur := range userRoles {
		ids = append(ids, ur.RoleID)
	}
	return ids
}

// userHasMenuPermission — cerminan langkah 3-6 di middleware.RequirePermission:
// cari menu berdasarkan route_path, gabungkan permission dari semua role user
// pada menu itu, lalu cek apakah salah satu permission yang diminta dimiliki.
func userHasMenuPermission(roleIDs []string, routePath string, permissions []string) bool {
	if len(roleIDs) == 0 || len(permissions) == 0 {
		return false
	}

	var menu models.Menu
	if err := config.DB.
		Where("route_path = ? AND deleted_at IS NULL", routePath).
		First(&menu).Error; err != nil {
		// Menu tidak terdaftar — middleware pun akan menolak request-nya,
		// jadi stage ini tidak bisa dianggap tugas siapa pun.
		return false
	}

	wanted := make(map[string]bool, len(permissions))
	for _, p := range permissions {
		wanted[p] = true
	}

	var roleMenus []models.RoleMenu
	config.DB.Where("role_id IN ? AND menu_id = ?", roleIDs, menu.ID).Find(&roleMenus)

	for _, rm := range roleMenus {
		for _, p := range rm.Permissions {
			if wanted[p] {
				return true
			}
		}
	}
	return false
}

// userBranchIDs — cabang yang dimiliki user.
func userBranchIDs(userID string) []string {
	var myBranches []models.UserBranch
	config.DB.Where("user_id = ?", userID).Find(&myBranches)

	ids := make([]string, 0, len(myBranches))
	for _, ub := range myBranches {
		if ub.BranchID != "" {
			ids = append(ids, ub.BranchID)
		}
	}
	return ids
}

// userBranchCodes — kode cabang milik user, dipakai membandingkan dengan
// mutation_to_branch_code yang menyimpan kode, bukan ID.
func userBranchCodes(userID string) []string {
	branchIDs := userBranchIDs(userID)
	if len(branchIDs) == 0 {
		return nil
	}

	var branches []models.Branch
	config.DB.Where("id IN ?", branchIDs).Find(&branches)

	codes := make([]string, 0, len(branches))
	for _, b := range branches {
		if b.BranchCode != "" {
			codes = append(codes, b.BranchCode)
		}
	}
	return codes
}

// branchPeerUserIDs — user yang punya irisan cabang dengan user ini.
//
// Dibandingkan di Go, bukan lewat JOIN SQL, karena transactions.created_by
// memakai collation utf8mb4_unicode_ci sementara user_branchs.user_id
// utf8mb4_general_ci — menjodohkannya langsung di SQL berujung
// "Illegal mix of collations".
func branchPeerUserIDs(userID string) []string {
	branchIDs := userBranchIDs(userID)
	if len(branchIDs) == 0 {
		return nil
	}

	var peers []models.UserBranch
	config.DB.Where("branch_id IN ?", branchIDs).Find(&peers)

	seen := map[string]bool{}
	userIDs := make([]string, 0, len(peers))
	for _, ub := range peers {
		if !seen[ub.UserID] {
			seen[ub.UserID] = true
			userIDs = append(userIDs, ub.UserID)
		}
	}
	return userIDs
}

// myTurnApprovalNumbers — nomor transaksi yang step approval BERJALANNYA milik
// role user ini.
//
// "Berjalan" berarti step pending dengan step_order terkecil — approver di
// urutan belakang belum gilirannya walau barisnya sudah pending.
func myTurnApprovalNumbers(roleIDs []string, transactionType string) []string {
	if len(roleIDs) == 0 {
		return nil
	}

	var pending []models.TransactionApproval
	config.DB.
		Preload("ApprovalFlowStep").
		Where("transaction_type = ? AND status = ?", transactionType, "pending").
		Find(&pending)

	// kelompokkan per (nomor transaksi, flow) lalu ambil step_order terkecil
	type key struct{ number, flow string }
	lowest := map[key]*models.TransactionApproval{}

	for i := range pending {
		row := &pending[i]
		if row.ApprovalFlowStep == nil {
			continue
		}
		k := key{row.TransactionNumber, row.FlowID}
		if current, ok := lowest[k]; !ok ||
			row.ApprovalFlowStep.StepOrder < current.ApprovalFlowStep.StepOrder {
			lowest[k] = row
		}
	}

	roleSet := map[string]bool{}
	for _, id := range roleIDs {
		roleSet[id] = true
	}

	seen := map[string]bool{}
	numbers := make([]string, 0)
	for _, row := range lowest {
		if row.ApproverRoleID == nil || !roleSet[*row.ApproverRoleID] {
			continue
		}
		if !seen[row.TransactionNumber] {
			seen[row.TransactionNumber] = true
			numbers = append(numbers, row.TransactionNumber)
		}
	}
	return numbers
}

// permittedStages memisahkan stage yang boleh dikerjakan user menjadi dua
// kelompok: yang dinilai dari cabang pengaju, dan yang dinilai dari cabang
// tujuan transaksi.
func permittedStages(roleIDs []string, cfg waitingConfig) (byCreator, byDestination []string) {
	for stage, owner := range cfg.owners {
		if !userHasMenuPermission(roleIDs, owner.routePath, owner.permissions) {
			continue
		}
		if owner.byDestinationBranch {
			byDestination = append(byDestination, stage)
		} else {
			byCreator = append(byCreator, stage)
		}
	}
	return byCreator, byDestination
}

// applyWaitingFilter mempersempit query ke pengajuan yang menunggu tindakan
// user ini:
//
//   - DRAFT milik sendiri (hanya pembuat yang bisa submit)
//   - stage yang endpoint-nya boleh dia akses, dan pengajuannya berasal dari
//     cabang yang dia punya
//   - stage yang dikerjakan cabang tujuan, dan tujuannya cabang dia
//   - stage approval yang gilirannya ada di role dia
func applyWaitingFilter(query *gorm.DB, userID string, cfg waitingConfig) *gorm.DB {
	roleIDs := userRoleIDs(userID)
	byCreator, byDestination := permittedStages(roleIDs, cfg)

	conditions := config.DB.
		Where("created_by = ? AND current_stage = ?", userID, cfg.draftStage)

	if len(byCreator) > 0 {
		if peers := branchPeerUserIDs(userID); len(peers) > 0 {
			// stage dengan pengecualian dapat kondisinya sendiri
			plain := make([]string, 0, len(byCreator))
			for _, stage := range byCreator {
				owner := cfg.owners[stage]
				if owner.excludeIDs == nil {
					plain = append(plain, stage)
					continue
				}
				conditions = conditions.Or(
					config.DB.Where("current_stage = ?", stage).
						Where("created_by IN ?", peers).
						Where("id NOT IN (?)", owner.excludeIDs()),
				)
			}
			if len(plain) > 0 {
				conditions = conditions.Or(
					config.DB.Where("current_stage IN ?", plain).
						Where("created_by IN ?", peers),
				)
			}
		}
	}

	if len(byDestination) > 0 {
		if codes := userBranchCodes(userID); len(codes) > 0 {
			conditions = conditions.Or(
				config.DB.Where("current_stage IN ?", byDestination).
					Where("mutation_to_branch_code IN ?", codes),
			)
		}
	}

	if numbers := myTurnApprovalNumbers(roleIDs, cfg.txType); len(numbers) > 0 {
		conditions = conditions.Or("transaction_number IN ?", numbers)
	}

	if cfg.extraCondition != nil {
		conditions = conditions.Or(cfg.extraCondition(userID))
	}

	return query.Where(conditions)
}

// isWaitingForUser — apakah satu transaksi sedang menunggu tindakan user ini.
// Aturannya sama persis dengan filter daftar di atas, dipakai halaman detail
// untuk menyembunyikan tombol aksi yang bukan bagiannya.
func isWaitingForUser(userID string, transaction *models.Transaction, cfg waitingConfig) bool {
	if userID == "" || transaction == nil {
		return false
	}

	stage := transaction.CurrentStage

	if cfg.extraCheck != nil && cfg.extraCheck(userID, transaction) {
		return true
	}

	// DRAFT hanya bisa dilanjutkan pembuatnya
	if stage == cfg.draftStage {
		return transaction.CreatedBy == userID
	}

	roleIDs := userRoleIDs(userID)

	owner, ok := cfg.owners[stage]
	if !ok {
		// Stage tanpa pemilik endpoint berarti tahap approval: giliran role user
		// pada step yang sedang berjalan.
		for _, number := range myTurnApprovalNumbers(roleIDs, cfg.txType) {
			if number == transaction.TransactionNumber {
				return true
			}
		}
		return false
	}

	if !userHasMenuPermission(roleIDs, owner.routePath, owner.permissions) {
		return false
	}

	if owner.excludeIDs != nil {
		var excluded int64
		config.DB.Model(&models.Transaction{}).
			Where("id = ? AND id IN (?)", transaction.ID, owner.excludeIDs()).
			Count(&excluded)
		if excluded > 0 {
			return false
		}
	}

	if owner.byDestinationBranch {
		if transaction.MutationToBranchCode == nil {
			return false
		}
		for _, code := range userBranchCodes(userID) {
			if code == *transaction.MutationToBranchCode {
				return true
			}
		}
		return false
	}

	// pengaju harus berada di cabang yang juga dimiliki user ini
	for _, peer := range branchPeerUserIDs(userID) {
		if peer == transaction.CreatedBy {
			return true
		}
	}
	return false
}
