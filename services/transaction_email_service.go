package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ============================================================
// Email notifikasi aksi stage
//
// Alurnya di frontend: user menekan aksi → dialog email (preview) → aksi
// dijalankan → email dikirim. Email dikirim SETELAH aksi berhasil, jadi
// kegagalan SMTP tidak pernah menahan proses bisnis; email yang gagal tercatat
// di transaction_email_logs dan bisa dikirim ulang.
//
// Penerima To tidak dikonfigurasi — dihitung dari siapa yang akan memegang
// bola setelah aksi ini, memakai aturan yang sama dengan "Menunggu Saya"
// (waiting_service.go): permission di route stage + cabang pengajuan, atau
// giliran step approval.
// ============================================================

type emailFlowConfig struct {
	waiting        waitingConfig
	getTransaction func(string) (*models.Transaction, error)
	nextStage      func(*models.Transaction) string
	// stage approval → flow_code yang dipakai stage itu
	approvalFlows map[string]string
	finishedStage string
	detailPath    string
}

func emailFlowFor(transactionType string) (emailFlowConfig, error) {
	switch transactionType {
	case TxProcurement:
		return emailFlowConfig{
			waiting:        procurementWaiting,
			getTransaction: getProcurementTransaction,
			nextStage: func(t *models.Transaction) string {
				return nextInList([]string{
					models.StageDraft,
					models.StageAssetVerification,
					models.StageApproval,
					models.StageProcessBudget,
					models.StageExecuteAsset,
					models.StageGR,
					models.StageFinished,
				}, t.CurrentStage)
			},
			approvalFlows: map[string]string{models.StageApproval: "PROCUREMENT_APPROVAL"},
			finishedStage: models.StageFinished,
			detailPath:    "/dashboard/procurement/",
		}, nil
	case TxMutationFlow:
		return emailFlowConfig{
			waiting:        mutationWaiting,
			getTransaction: getMutationTransaction,
			nextStage: func(t *models.Transaction) string {
				return nextInList([]string{
					models.StageMutationDraft,
					models.StageMutationApproval,
					models.StageMutationReceiving,
					models.StageMutationExecute,
					models.StageMutationFinished,
				}, t.CurrentStage)
			},
			approvalFlows: map[string]string{models.StageMutationApproval: "MUTATION_APPROVAL"},
			finishedStage: models.StageMutationFinished,
			detailPath:    "/dashboard/mutation/",
		}, nil
	case TxDisposalFlow:
		return emailFlowConfig{
			waiting:        disposalWaiting,
			getTransaction: getDisposalTransaction,
			nextStage: func(t *models.Transaction) string {
				disposalType := models.DisposalTypeDispose
				if t.DisposalType != nil && *t.DisposalType != "" {
					disposalType = *t.DisposalType
				}
				next, err := nextStageForDisposal(disposalType, t.CurrentStage)
				if err != nil {
					return ""
				}
				return next
			},
			approvalFlows: map[string]string{models.StageDisposalApprovalRequest: "DISPOSAL_APPROVAL_REQUEST"},
			finishedStage: models.StageDisposalFinished,
			detailPath:    "/dashboard/disposal/",
		}, nil
	}
	return emailFlowConfig{}, fmt.Errorf("jenis transaksi %s tidak didukung", transactionType)
}

func nextInList(stages []string, current string) string {
	for i, s := range stages {
		if s == current && i+1 < len(stages) {
			return stages[i+1]
		}
	}
	return ""
}

// ============================================================
// Penentuan penerima
// ============================================================

// transactionBranchIDs — cabang pengajuan: homebase aktif pembuatnya, sama
// dengan yang dipakai validateApproverBranch. Kalau pembuat tidak punya
// homebase aktif, jatuh ke seluruh cabang miliknya.
func transactionBranchIDs(transaction *models.Transaction) []string {
	if homebase, err := GetUserActiveHomebase(transaction.CreatedBy); err == nil && homebase.BranchID != "" {
		return []string{homebase.BranchID}
	}
	return userBranchIDs(transaction.CreatedBy)
}

func branchIDsByCode(code string) []string {
	var branches []models.Branch
	config.DB.Where("branch_code = ?", code).Find(&branches)
	ids := make([]string, 0, len(branches))
	for _, b := range branches {
		ids = append(ids, b.ID)
	}
	return ids
}

// filterUsersInBranches menyisakan user yang punya salah satu cabang itu.
func filterUsersInBranches(userIDs, branchIDs []string) []string {
	if len(userIDs) == 0 || len(branchIDs) == 0 {
		return nil
	}
	var rows []models.UserBranch
	config.DB.Where("user_id IN ? AND branch_id IN ?", userIDs, branchIDs).Find(&rows)

	seen := map[string]bool{}
	result := make([]string, 0, len(rows))
	for _, ub := range rows {
		if !seen[ub.UserID] {
			seen[ub.UserID] = true
			result = append(result, ub.UserID)
		}
	}
	return result
}

func usersWithRoles(roleIDs []string) []string {
	if len(roleIDs) == 0 {
		return nil
	}
	var rows []models.UserRole
	config.DB.Where("role_id IN ?", roleIDs).Find(&rows)

	seen := map[string]bool{}
	result := make([]string, 0, len(rows))
	for _, ur := range rows {
		if !seen[ur.UserID] {
			seen[ur.UserID] = true
			result = append(result, ur.UserID)
		}
	}
	return result
}

// rolesWithMenuPermission — kebalikan userHasMenuPermission: role mana saja
// yang lolos RequirePermission di route itu.
func rolesWithMenuPermission(routePath string, permissions []string) []string {
	var menu models.Menu
	if err := config.DB.
		Where("route_path = ? AND deleted_at IS NULL", routePath).
		First(&menu).Error; err != nil {
		return nil
	}

	wanted := map[string]bool{}
	for _, p := range permissions {
		wanted[p] = true
	}

	var roleMenus []models.RoleMenu
	config.DB.Where("menu_id = ?", menu.ID).Find(&roleMenus)

	result := []string{}
	for _, rm := range roleMenus {
		for _, p := range rm.Permissions {
			if wanted[p] {
				result = append(result, rm.RoleID)
				break
			}
		}
	}
	return result
}

// pendingApprovalRows — baris approval pending milik satu flow, urut step.
// Baris pertama adalah step yang sedang berjalan.
func pendingApprovalRows(transactionNumber, transactionType, flowCode string) []models.TransactionApproval {
	var rows []models.TransactionApproval
	config.DB.
		Preload("ApprovalFlowStep").
		Where("transaction_number = ? AND transaction_type = ? AND status = ?",
			transactionNumber, transactionType, "pending").
		Where("flow_id IN (?)", config.DB.Model(&models.ApprovalFlow{}).
			Select("id").Where("flow_code = ?", flowCode)).
		Find(&rows)

	sort.SliceStable(rows, func(i, j int) bool {
		return stepOrderOf(rows[i]) < stepOrderOf(rows[j])
	})
	return rows
}

func stepOrderOf(row models.TransactionApproval) int {
	if row.ApprovalFlowStep == nil {
		return 1 << 30
	}
	return row.ApprovalFlowStep.StepOrder
}

func approverUsersOfRow(row models.TransactionApproval, branchIDs []string) []string {
	if row.ApproverUserID != nil && *row.ApproverUserID != "" {
		return []string{*row.ApproverUserID}
	}
	if row.ApproverRoleID == nil {
		return nil
	}
	return filterUsersInBranches(usersWithRoles([]string{*row.ApproverRoleID}), branchIDs)
}

// stageHandlers — siapa yang mengerjakan transaksi ini ketika ia berada di
// `stage`. depth mencegah rekursi tak berujung kalau seluruh step approval
// milik pembuat (auto-approve) dan kita melompat ke stage setelahnya.
func stageHandlers(cfg emailFlowConfig, transaction *models.Transaction, stage string, depth int) []string {
	if stage == "" || depth > 3 {
		return nil
	}

	if stage == cfg.waiting.draftStage || stage == cfg.finishedStage {
		return []string{transaction.CreatedBy}
	}

	branchIDs := transactionBranchIDs(transaction)

	if flowCode, isApproval := cfg.approvalFlows[stage]; isApproval {
		flow, err := GetApprovalFlowByCodeAndBranch(flowCode, GetCreatorBranchCode(transaction.CreatedBy))
		if err != nil {
			return nil
		}
		// Step di barisan depan yang approver-nya pembuat transaksi di-approve
		// otomatis oleh InitiateTransactionApproval, jadi dilewati.
		for _, step := range flow.FlowSteps {
			if step.RoleID == nil {
				continue
			}
			if userHasRole(transaction.CreatedBy, *step.RoleID) {
				continue
			}
			return filterUsersInBranches(usersWithRoles([]string{*step.RoleID}), branchIDs)
		}
		// semua step milik pembuat → approval langsung selesai
		afterApproval := *transaction
		afterApproval.CurrentStage = stage
		return stageHandlers(cfg, transaction, cfg.nextStage(&afterApproval), depth+1)
	}

	owner, ok := cfg.waiting.owners[stage]
	if !ok {
		return nil
	}

	roleUsers := usersWithRoles(rolesWithMenuPermission(owner.routePath, owner.permissions))
	if owner.byDestinationBranch {
		if transaction.MutationToBranchCode == nil {
			return nil
		}
		return filterUsersInBranches(roleUsers, branchIDsByCode(*transaction.MutationToBranchCode))
	}
	return filterUsersInBranches(roleUsers, branchIDs)
}

// resolveToRecipients — penerima To untuk aksi yang akan dilakukan pada
// kondisi transaksi SAAT INI (dipanggil sebelum aksinya dijalankan).
func resolveToRecipients(cfg emailFlowConfig, transaction *models.Transaction, action string) (userIDs []string, nextStage string) {
	stage := transaction.CurrentStage

	switch action {
	case models.EmailActionReject:
		return []string{transaction.CreatedBy}, models.StageRejected
	case models.EmailActionRevise:
		return []string{transaction.CreatedBy}, cfg.waiting.draftStage
	case models.EmailActionCancel:
		// dikabari: yang sedang memegang transaksi ini
		if flowCode, isApproval := cfg.approvalFlows[stage]; isApproval {
			if pending := pendingApprovalRows(transaction.TransactionNumber, cfg.waiting.txType, flowCode); len(pending) > 0 {
				return approverUsersOfRow(pending[0], transactionBranchIDs(transaction)), models.StageCancelled
			}
		}
		return stageHandlers(cfg, transaction, stage, 0), models.StageCancelled
	}

	// proceed
	if flowCode, isApproval := cfg.approvalFlows[stage]; isApproval {
		pending := pendingApprovalRows(transaction.TransactionNumber, cfg.waiting.txType, flowCode)
		// masih ada step setelah step berjalan → diteruskan ke approver berikutnya
		if len(pending) >= 2 {
			return approverUsersOfRow(pending[1], transactionBranchIDs(transaction)), stage
		}
	}

	next := cfg.nextStage(transaction)
	return stageHandlers(cfg, transaction, next, 0), next
}

// resolveCCRecipients — user dengan role CC template, di cabang pengajuan.
func resolveCCRecipients(transaction *models.Transaction, templateID uint) []string {
	var ccRoles []models.EmailTemplateCCRole
	config.DB.Where("email_template_id = ?", templateID).Find(&ccRoles)

	roleIDs := make([]string, 0, len(ccRoles))
	for _, cc := range ccRoles {
		roleIDs = append(roleIDs, cc.RoleID)
	}
	return filterUsersInBranches(usersWithRoles(roleIDs), transactionBranchIDs(transaction))
}

// toRecipients memuat user aktif yang punya email, membuang `exclude`.
func toRecipients(userIDs []string, exclude map[string]bool) []dto.EmailRecipient {
	result := []dto.EmailRecipient{}
	if len(userIDs) == 0 {
		return result
	}

	var users []models.User
	config.DB.Where("id IN ? AND status = ?", userIDs, "active").Order("fullname").Find(&users)

	for _, u := range users {
		if exclude[u.ID] || strings.TrimSpace(u.Email) == "" {
			continue
		}
		name := strings.TrimSpace(u.Fullname)
		if name == "" {
			name = u.Username
		}
		result = append(result, dto.EmailRecipient{UserID: u.ID, Name: name, Email: u.Email})
	}
	return result
}

// ============================================================
// Render template
// ============================================================

type emailRenderContext struct {
	transaction *models.Transaction
	stage       string
	nextStage   string
	action      string
	senderID    string
	link        string
	// diisi untuk agreement yang lintas cabang; kosong = cabang pembuat
	branchCode string
}

func appBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv("APP_BASE_URL")), "/")
}

// transactionLink — tautan halaman detail. Nomor transaksi mengandung "/" dan
// spasi, jadi tiap segmen di-escape tapi "/"-nya dibiarkan (rute detail di
// frontend memakai splat).
func transactionLink(cfg emailFlowConfig, transactionNumber string) string {
	base := appBaseURL()
	if base == "" {
		return ""
	}
	segments := strings.Split(transactionNumber, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return base + cfg.detailPath + strings.Join(segments, "/")
}

func renderEmailText(text string, ctx emailRenderContext) string {
	creatorName := ""
	if name := resolveUserFullname(ctx.transaction.CreatedBy); name != nil {
		creatorName = *name
	}
	senderName := ""
	if name := resolveUserFullname(ctx.senderID); name != nil {
		senderName = *name
	}

	branchCode := ctx.branchCode
	if branchCode == "" {
		branchCode = GetCreatorBranchCode(ctx.transaction.CreatedBy)
	}

	return strings.NewReplacer(
		"{{transaction_number}}", ctx.transaction.TransactionNumber,
		"{{transaction_type}}", ctx.transaction.TransactionType,
		"{{stage}}", ctx.stage,
		"{{next_stage}}", ctx.nextStage,
		"{{action}}", ctx.action,
		"{{creator_name}}", creatorName,
		"{{sender_name}}", senderName,
		"{{branch_code}}", branchCode,
		"{{link}}", ctx.link,
		"{{date}}", time.Now().Format("02 Jan 2006 15:04"),
	).Replace(text)
}

// ============================================================
// Preview
// ============================================================

func findActiveEmailTemplate(transactionType, stage, action string) (*models.EmailTemplate, error) {
	var tpl models.EmailTemplate
	err := config.DB.
		Where("transaction_type = ? AND stage = ? AND action = ? AND is_active = ?",
			transactionType, stage, action, true).
		First(&tpl).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tpl, nil
}

func PreviewTransactionEmail(userID, transactionType, transactionNumber, action string, memberNumbers []string) (*dto.EmailPreviewResponse, error) {
	// agreement bukan baris transactions — penerima & cabangnya dihitung
	// dari transaksi anggotanya (agreement_email_service.go)
	if transactionType == TxDisposalAgreement {
		return previewAgreementEmail(userID, transactionNumber, action, memberNumbers)
	}

	cfg, err := emailFlowFor(transactionType)
	if err != nil {
		return nil, err
	}
	switch action {
	case models.EmailActionProceed, models.EmailActionReject, models.EmailActionRevise, models.EmailActionCancel:
	default:
		return nil, fmt.Errorf("aksi %s tidak dikenal", action)
	}

	transaction, err := cfg.getTransaction(transactionNumber)
	if err != nil {
		return nil, err
	}

	response := &dto.EmailPreviewResponse{
		TransactionNumber: transaction.TransactionNumber,
		TransactionType:   transactionType,
		Stage:             transaction.CurrentStage,
		Action:            action,
		To:                []dto.EmailRecipient{},
		CC:                []dto.EmailRecipient{},
		MailerConfigured:  MailerConfigured(),
	}

	tpl, err := findActiveEmailTemplate(transactionType, transaction.CurrentStage, action)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		// tidak dikonfigurasi → frontend langsung menjalankan aksinya
		return response, nil
	}

	toIDs, nextStage := resolveToRecipients(cfg, transaction, action)

	// pengirim tidak perlu mengirimi dirinya sendiri
	exclude := map[string]bool{userID: true}
	to := toRecipients(toIDs, exclude)
	for _, r := range to {
		exclude[r.UserID] = true
	}
	cc := toRecipients(resolveCCRecipients(transaction, tpl.ID), exclude)

	ctx := emailRenderContext{
		transaction: transaction,
		stage:       transaction.CurrentStage,
		nextStage:   nextStage,
		action:      action,
		senderID:    userID,
		link:        transactionLink(cfg, transaction.TransactionNumber),
	}

	response.HasTemplate = true
	response.TemplateID = tpl.ID
	response.NextStage = nextStage
	response.Subject = renderEmailText(tpl.Subject, ctx)
	response.Body = renderEmailText(tpl.Body, ctx)
	response.HTML = buildEmailHTML(emailLayout{
		action:     action,
		subject:    response.Subject,
		message:    response.Body,
		senderName: strOr(resolveUserFullname(userID), "-"),
		link:       ctx.link,
		summary:    buildTransactionSummary(transaction, ctx),
	})
	response.To = to
	response.CC = cc
	return response, nil
}

// ============================================================
// Kirim & kirim ulang
// ============================================================

// userActedRecently — pengirim harus orang yang baru saja menjalankan aksi
// pada transaksi ini. Tanpa ini endpoint kirim bisa dipakai siapa saja untuk
// mengirim email bebas lewat akun SMTP perusahaan.
func userActedRecently(userID string, transaction *models.Transaction) bool {
	since := time.Now().Add(-30 * time.Minute)

	var stageCount int64
	config.DB.Model(&models.TransactionStage{}).
		Where("transaction_number = ? AND actor_id = ? AND created_at >= ?",
			transaction.TransactionNumber, userID, since).
		Count(&stageCount)
	if stageCount > 0 {
		return true
	}

	// approve/reject step di tengah flow tidak mencatat perpindahan stage
	var approvalCount int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND (approved_by = ? OR rejected_by = ?) AND updated_at >= ?",
			transaction.TransactionNumber, userID, userID, since).
		Count(&approvalCount)
	if approvalCount > 0 {
		return true
	}

	// GR sebagian aset juga tidak memindahkan stage — baru pindah ke FINISHED
	// setelah seluruh aset diterima
	var grCount int64
	config.DB.Model(&models.AssetGR{}).
		Where("transaction_number = ? AND gr_by = ? AND created_at >= ?",
			transaction.TransactionNumber, userID, since).
		Count(&grCount)
	return grCount > 0
}

func normalizeEmails(list []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(list))
	for _, e := range list {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		result = append(result, e)
	}
	return result
}

func SendTransactionEmail(userID string, req dto.SendTransactionEmailRequest) (*dto.TransactionEmailLogResponse, error) {
	if req.TransactionType == TxDisposalAgreement {
		return sendAgreementEmail(userID, req)
	}

	cfg, err := emailFlowFor(req.TransactionType)
	if err != nil {
		return nil, err
	}

	transaction, err := cfg.getTransaction(req.TransactionNumber)
	if err != nil {
		return nil, err
	}

	var tpl models.EmailTemplate
	if err := config.DB.First(&tpl, req.TemplateID).Error; err != nil {
		return nil, errors.New("email template not found")
	}
	if tpl.TransactionType != req.TransactionType {
		return nil, errors.New("template tidak sesuai dengan jenis transaksi")
	}

	if !userActedRecently(userID, transaction) {
		return nil, errors.New("email hanya bisa dikirim oleh user yang baru saja memproses transaksi ini")
	}

	to := normalizeEmails(req.To)
	cc := normalizeEmails(req.CC)
	if len(to) == 0 || len(cc) == 0 {
		return nil, errors.New("penerima To dan CC masing-masing minimal 1")
	}

	// Aksi sudah dijalankan, jadi current_stage sekarang adalah stage tujuan
	// yang sebenarnya.
	ctx := emailRenderContext{
		transaction: transaction,
		stage:       tpl.Stage,
		nextStage:   transaction.CurrentStage,
		action:      tpl.Action,
		senderID:    userID,
		link:        transactionLink(cfg, transaction.TransactionNumber),
	}
	return deliverAndLog(userID, &tpl, ctx, to, cc, req.AdditionalMessage, buildTransactionSummary(transaction, ctx))
}

// deliverAndLog merender template, mengirim, lalu mencatat hasilnya — gagal
// maupun berhasil.
func deliverAndLog(userID string, tpl *models.EmailTemplate, ctx emailRenderContext, to, cc []string, additional string, summary emailSummary) (*dto.TransactionEmailLogResponse, error) {
	subject := renderEmailText(tpl.Subject, ctx)
	body := buildEmailHTML(emailLayout{
		action:     tpl.Action,
		subject:    subject,
		message:    renderEmailText(tpl.Body, ctx),
		additional: additional,
		senderName: strOr(resolveUserFullname(userID), "-"),
		link:       ctx.link,
		summary:    summary,
	})

	toJSON, _ := json.Marshal(to)
	ccJSON, _ := json.Marshal(cc)

	templateID := tpl.ID
	logRow := models.TransactionEmailLog{
		EmailTemplateID:   &templateID,
		TransactionNumber: ctx.transaction.TransactionNumber,
		TransactionType:   tpl.TransactionType,
		Stage:             tpl.Stage,
		Action:            tpl.Action,
		Subject:           subject,
		Body:              body,
		ToEmails:          string(toJSON),
		CCEmails:          string(ccJSON),
		Attempts:          1,
		SentBy:            userID,
	}
	applySendResult(&logRow, sendMail(outgoingMail{to: to, cc: cc, subject: subject, htmlBody: body}))

	if err := config.DB.Create(&logRow).Error; err != nil {
		return nil, err
	}

	response := mapEmailLogToResponse(logRow, nil)
	return &response, nil
}

func applySendResult(logRow *models.TransactionEmailLog, sendErr error) {
	if sendErr != nil {
		msg := sendErr.Error()
		logRow.Status = models.EmailStatusFailed
		logRow.ErrorMessage = &msg
		return
	}
	now := time.Now()
	logRow.Status = models.EmailStatusSent
	logRow.ErrorMessage = nil
	logRow.SentAt = &now
}

// ResendTransactionEmail mengirim ulang email yang gagal, dengan isi dan
// penerima persis seperti percobaan pertama.
func ResendTransactionEmail(userID string, isAdmin bool, logID uint) (*dto.TransactionEmailLogResponse, error) {
	var logRow models.TransactionEmailLog
	if err := config.DB.First(&logRow, logID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("email log not found")
		}
		return nil, err
	}

	if !isAdmin && logRow.SentBy != userID {
		return nil, errors.New("hanya pengirim atau admin yang bisa mengirim ulang email ini")
	}
	if logRow.Status == models.EmailStatusSent {
		return nil, errors.New("email ini sudah terkirim")
	}

	var to, cc []string
	_ = json.Unmarshal([]byte(logRow.ToEmails), &to)
	_ = json.Unmarshal([]byte(logRow.CCEmails), &cc)

	applySendResult(&logRow, sendMail(outgoingMail{to: to, cc: cc, subject: logRow.Subject, htmlBody: logRow.Body}))
	logRow.Attempts++

	if err := config.DB.Model(&logRow).Updates(map[string]interface{}{
		"status":        logRow.Status,
		"error_message": logRow.ErrorMessage,
		"sent_at":       logRow.SentAt,
		"attempts":      logRow.Attempts,
	}).Error; err != nil {
		return nil, err
	}

	response := mapEmailLogToResponse(logRow, nil)
	return &response, nil
}

type EmailLogFilter struct {
	Status            string
	TransactionNumber string
	// kosong = semua (admin); selain admin hanya melihat kirimannya sendiri
	SentBy string
	Limit  int
}

func GetTransactionEmailLogs(filter EmailLogFilter) ([]dto.TransactionEmailLogResponse, error) {
	query := config.DB.Model(&models.TransactionEmailLog{}).Order("created_at DESC")
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.TransactionNumber != "" {
		query = query.Where("transaction_number LIKE ?", "%"+filter.TransactionNumber+"%")
	}
	if filter.SentBy != "" {
		query = query.Where("sent_by = ?", filter.SentBy)
	}
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var rows []models.TransactionEmailLog
	if err := query.Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}

	senderIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		senderIDs = append(senderIDs, r.SentBy)
	}
	names := resolveUserFullnames(senderIDs)

	result := make([]dto.TransactionEmailLogResponse, 0, len(rows))
	for _, r := range rows {
		result = append(result, mapEmailLogToResponse(r, names))
	}
	return result, nil
}

func mapEmailLogToResponse(row models.TransactionEmailLog, names map[string]string) dto.TransactionEmailLogResponse {
	var to, cc []string
	_ = json.Unmarshal([]byte(row.ToEmails), &to)
	_ = json.Unmarshal([]byte(row.CCEmails), &cc)

	var senderName *string
	if names != nil {
		if n, ok := names[row.SentBy]; ok {
			senderName = &n
		}
	} else {
		senderName = resolveUserFullname(row.SentBy)
	}

	return dto.TransactionEmailLogResponse{
		ID:                row.ID,
		EmailTemplateID:   row.EmailTemplateID,
		TransactionNumber: row.TransactionNumber,
		TransactionType:   row.TransactionType,
		Stage:             row.Stage,
		Action:            row.Action,
		Subject:           row.Subject,
		To:                to,
		CC:                cc,
		Status:            row.Status,
		ErrorMessage:      row.ErrorMessage,
		Attempts:          row.Attempts,
		SentBy:            row.SentBy,
		SentByName:        senderName,
		SentAt:            row.SentAt,
		CreatedAt:         row.CreatedAt,
	}
}
