package services

import (
	"backend-go/config"
	"backend-go/dto"
	"backend-go/models"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ============================================================
// Email notifikasi Disposal Agreement
//
// Agreement bukan baris `transactions` dan lintas cabang, jadi aturan
// "satu cabang dengan pengajuan" diganti:
//   - To  : approver wajib punya akses ke SEMUA cabang anggota — sama dengan
//           validateAgreementApproverBranches, orang lain toh akan ditolak
//           saat approve
//   - CC  : user dengan role CC yang punya akses ke SALAH SATU cabang anggota
//           (kepala cabang tiap anggota ikut tahu). Untuk revisi, cabang yang
//           dihitung hanya cabang anggota yang dikeluarkan.
//
// Cabang anggota = cabang homebase pembuat tiap transaksi disposal anggota,
// sama dengan agreementMemberBranches.
// ============================================================

// EmailStageAgreementCreate — "stage" template untuk saat agreement dibuat.
// Agreement belum ada sebelum aksi itu, jadi tidak punya stage sungguhan.
const EmailStageAgreementCreate = "CREATE"

// placeholder nomor agreement di preview pembuatan — nomornya baru dibuat
// server saat disimpan, email yang benar-benar terkirim memakai nomor asli
const agreementNumberPending = "[nomor agreement otomatis]"

type agreementEmailScope struct {
	number        string
	creatorID     string
	stage         string // stage agreement saat ini ("" = belum dibuat)
	memberNumbers []string
	branchCodes   []string
	// revisi: anggota yang dikeluarkan — penerima & cabang dihitung dari sini
	revisedNumbers []string
}

// memberBranchCodes — cabang homebase pembuat transaksi anggota.
func memberBranchCodes(memberNumbers []string) (codes []string, creators []string) {
	if len(memberNumbers) == 0 {
		return nil, nil
	}
	var transactions []models.Transaction
	config.DB.Where("transaction_number IN ?", memberNumbers).Find(&transactions)

	seenCode := map[string]bool{}
	seenCreator := map[string]bool{}
	for _, trx := range transactions {
		if code := GetCreatorBranchCode(trx.CreatedBy); code != "" && code != "ALL" && !seenCode[code] {
			seenCode[code] = true
			codes = append(codes, code)
		}
		if trx.CreatedBy != "" && !seenCreator[trx.CreatedBy] {
			seenCreator[trx.CreatedBy] = true
			creators = append(creators, trx.CreatedBy)
		}
	}
	return codes, creators
}

func branchIDsByCodes(codes []string) [][]string {
	groups := make([][]string, 0, len(codes))
	for _, code := range codes {
		groups = append(groups, branchIDsByCode(code))
	}
	return groups
}

// usersInAllBranches — user yang punya akses ke SETIAP cabang anggota.
func usersInAllBranches(userIDs []string, branchCodes []string) []string {
	if len(branchCodes) == 0 {
		return nil
	}
	result := userIDs
	for _, ids := range branchIDsByCodes(branchCodes) {
		result = filterUsersInBranches(result, ids)
		if len(result) == 0 {
			return nil
		}
	}
	return result
}

// usersInAnyBranch — user yang punya akses ke salah satu cabang anggota.
func usersInAnyBranch(userIDs []string, branchCodes []string) []string {
	all := []string{}
	for _, ids := range branchIDsByCodes(branchCodes) {
		all = append(all, ids...)
	}
	return filterUsersInBranches(userIDs, all)
}

func loadAgreementScope(agreementNumber string, memberNumbers []string) (*agreementEmailScope, error) {
	if agreementNumber == "" {
		// pembuatan: anggota dari request, pembuat = pengirim (diisi caller)
		codes, _ := memberBranchCodes(memberNumbers)
		return &agreementEmailScope{memberNumbers: memberNumbers, branchCodes: codes}, nil
	}

	agreement, err := GetDisposalAgreementByNumber(agreementNumber)
	if err != nil {
		return nil, err
	}
	var items []models.DisposalAgreementItem
	config.DB.Where("agreement_id = ?", agreement.ID).Find(&items)

	numbers := make([]string, 0, len(items))
	for _, item := range items {
		numbers = append(numbers, item.TransactionNumber)
	}
	codes, _ := memberBranchCodes(numbers)

	return &agreementEmailScope{
		number:        agreement.AgreementNumber,
		creatorID:     agreement.CreatedBy,
		stage:         agreement.CurrentStage,
		memberNumbers: numbers,
		branchCodes:   codes,
	}, nil
}

// agreementApproverUsers — approver satu step, dibatasi yang memegang semua
// cabang anggota.
func agreementApproverUsers(roleID *string, userID *string, scope *agreementEmailScope) []string {
	if userID != nil && *userID != "" {
		return []string{*userID}
	}
	if roleID == nil {
		return nil
	}
	return usersInAllBranches(usersWithRoles([]string{*roleID}), scope.branchCodes)
}

// resolveAgreementTo — penerima To untuk aksi pada kondisi agreement saat ini.
func resolveAgreementTo(scope *agreementEmailScope, stage, action string) (userIDs []string, nextStage string) {
	if action == models.EmailActionReject {
		return []string{scope.creatorID}, models.StageAgreementRejected
	}

	if action == models.EmailActionRevise {
		// pembuat disposal yang dikeluarkan (merekalah yang memperbaiki) +
		// pembuat agreement (anggotanya berkurang). Disposal itu kembali ke
		// DRAFT, agreement-nya sendiri tetap di APPROVAL_AGREEMENT.
		_, revisedCreators := memberBranchCodes(scope.revisedNumbers)
		return append([]string{scope.creatorID}, revisedCreators...), models.StageDisposalDraft
	}

	if stage == EmailStageAgreementCreate {
		// step terdepan yang dipegang pembuat di-approve otomatis saat initiate
		flow, err := resolveAgreementApprovalFlow()
		if err != nil {
			return nil, models.StageAgreementApproval
		}
		for _, step := range flow.FlowSteps {
			if step.RoleID == nil || userHasRole(scope.creatorID, *step.RoleID) {
				continue
			}
			return agreementApproverUsers(step.RoleID, nil, scope), models.StageAgreementApproval
		}
		return nil, models.StageAgreementApproval
	}

	// approve step
	pending := pendingApprovalRows(scope.number, TxDisposalAgreement, models.FlowDisposalApprovalAgreement)
	if len(pending) >= 2 {
		return agreementApproverUsers(pending[1].ApproverRoleID, pending[1].ApproverUserID, scope), models.StageAgreementApproval
	}

	// step terakhir → agreement selesai, kabari pembuat agreement dan pembuat
	// tiap transaksi anggota (transaksinya lanjut ke stage berikutnya)
	_, memberCreators := memberBranchCodes(scope.memberNumbers)
	return append([]string{scope.creatorID}, memberCreators...), models.StageAgreementFinished
}

func resolveAgreementCC(templateID uint, scope *agreementEmailScope) []string {
	branchCodes := scope.branchCodes
	if len(scope.revisedNumbers) > 0 {
		// revisi hanya menyangkut cabang anggota yang dikeluarkan
		branchCodes, _ = memberBranchCodes(scope.revisedNumbers)
	}

	var ccRoles []models.EmailTemplateCCRole
	config.DB.Where("email_template_id = ?", templateID).Find(&ccRoles)

	roleIDs := make([]string, 0, len(ccRoles))
	for _, cc := range ccRoles {
		roleIDs = append(roleIDs, cc.RoleID)
	}
	return usersInAnyBranch(usersWithRoles(roleIDs), branchCodes)
}

func agreementLink(agreementNumber string) string {
	base := appBaseURL()
	if base == "" || agreementNumber == "" || agreementNumber == agreementNumberPending {
		return ""
	}
	segments := strings.Split(agreementNumber, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return base + "/dashboard/disposal-agreement/" + strings.Join(segments, "/")
}

func agreementRenderContext(scope *agreementEmailScope, number, stage, nextStage, action, senderID string) emailRenderContext {
	return emailRenderContext{
		// agreement tidak punya baris transactions — cukup field yang dipakai
		// renderEmailText
		transaction: &models.Transaction{
			TransactionNumber: number,
			TransactionType:   TxDisposalAgreement,
			CreatedBy:         scope.creatorID,
		},
		stage:      stage,
		nextStage:  nextStage,
		action:     action,
		senderID:   senderID,
		link:       agreementLink(number),
		branchCode: strings.Join(scope.branchCodes, ", "),
	}
}

func previewAgreementEmail(userID, agreementNumber, action string, memberNumbers []string) (*dto.EmailPreviewResponse, error) {
	switch action {
	case models.EmailActionProceed, models.EmailActionReject, models.EmailActionRevise:
	default:
		return nil, fmt.Errorf("aksi %s tidak tersedia untuk agreement", action)
	}
	if action == models.EmailActionRevise && (agreementNumber == "" || len(memberNumbers) == 0) {
		return nil, errors.New("revisi agreement butuh agreement_number dan member_numbers (anggota yang dikeluarkan)")
	}

	scope, err := loadAgreementScope(agreementNumber, memberNumbers)
	if err != nil {
		return nil, err
	}
	if action == models.EmailActionRevise {
		scope.revisedNumbers = memberNumbers
	}

	stage := scope.stage
	number := scope.number
	if agreementNumber == "" {
		stage = EmailStageAgreementCreate
		number = agreementNumberPending
		scope.creatorID = userID
	}

	response := &dto.EmailPreviewResponse{
		TransactionNumber: number,
		TransactionType:   TxDisposalAgreement,
		Stage:             stage,
		Action:            action,
		To:                []dto.EmailRecipient{},
		CC:                []dto.EmailRecipient{},
		MailerConfigured:  MailerConfigured(),
	}

	tpl, err := findActiveEmailTemplate(TxDisposalAgreement, stage, action)
	if err != nil {
		return nil, err
	}
	if tpl == nil {
		return response, nil
	}

	toIDs, nextStage := resolveAgreementTo(scope, stage, action)

	exclude := map[string]bool{userID: true}
	to := toRecipients(toIDs, exclude)
	for _, r := range to {
		exclude[r.UserID] = true
	}
	cc := toRecipients(resolveAgreementCC(tpl.ID, scope), exclude)

	ctx := agreementRenderContext(scope, number, stage, nextStage, action, userID)
	if len(scope.revisedNumbers) > 0 {
		codes, _ := memberBranchCodes(scope.revisedNumbers)
		ctx.branchCode = strings.Join(codes, ", ")
	}

	response.HasTemplate = true
	response.TemplateID = tpl.ID
	response.NextStage = nextStage
	response.Subject = renderEmailText(tpl.Subject, ctx)
	response.Body = renderEmailText(tpl.Body, ctx)

	// aset yang ditampilkan: anggota yang dikeluarkan (revisi) atau seluruh
	// anggota (termasuk calon anggota saat pembuatan)
	numbers := scope.memberNumbers
	if action == models.EmailActionRevise {
		numbers = scope.revisedNumbers
	}
	response.HTML = buildEmailHTML(emailLayout{
		action:     action,
		subject:    response.Subject,
		message:    response.Body,
		senderName: strOr(resolveUserFullname(userID), "-"),
		link:       ctx.link,
		summary:    buildAgreementSummary(scope, numbers, ctx, action == models.EmailActionRevise),
	})
	response.To = to
	response.CC = cc
	return response, nil
}

// agreementActedRecently — pengirim harus pembuat agreement ini (untuk email
// pembuatan) atau approver/penolak salah satu step-nya, dalam 30 menit.
func agreementActedRecently(userID string, scope *agreementEmailScope, templateStage, templateAction string) bool {
	since := time.Now().Add(-30 * time.Minute)

	if templateStage == EmailStageAgreementCreate {
		var count int64
		config.DB.Model(&models.DisposalAgreement{}).
			Where("agreement_number = ? AND created_by = ? AND created_at >= ?", scope.number, userID, since).
			Count(&count)
		return count > 0
	}

	if templateAction == models.EmailActionRevise {
		// revisi tidak mengubah baris approval agreement — jejaknya ada di
		// stage disposal yang dikeluarkan (agreementRevisionNote)
		var count int64
		config.DB.Model(&models.TransactionStage{}).
			Where("actor_id = ? AND action = ? AND notes LIKE ? AND created_at >= ?",
				userID, models.ActionRevise, agreementRevisionNote(scope.number, "")+"%", since).
			Count(&count)
		return count > 0
	}

	var count int64
	config.DB.Model(&models.TransactionApproval{}).
		Where("transaction_number = ? AND transaction_type = ? AND (approved_by = ? OR rejected_by = ?) AND updated_at >= ?",
			scope.number, TxDisposalAgreement, userID, userID, since).
		Count(&count)
	return count > 0
}

func sendAgreementEmail(userID string, req dto.SendTransactionEmailRequest) (*dto.TransactionEmailLogResponse, error) {
	var tpl models.EmailTemplate
	if err := config.DB.First(&tpl, req.TemplateID).Error; err != nil {
		return nil, errors.New("email template not found")
	}
	if tpl.TransactionType != TxDisposalAgreement {
		return nil, errors.New("template tidak sesuai dengan jenis transaksi")
	}

	scope, err := loadAgreementScope(req.TransactionNumber, nil)
	if err != nil {
		return nil, err
	}

	if !agreementActedRecently(userID, scope, tpl.Stage, tpl.Action) {
		return nil, errors.New("email hanya bisa dikirim oleh user yang baru saja memproses agreement ini")
	}

	to := normalizeEmails(req.To)
	cc := normalizeEmails(req.CC)
	if len(to) == 0 || len(cc) == 0 {
		return nil, errors.New("penerima To dan CC masing-masing minimal 1")
	}

	// aksi sudah dijalankan: stage agreement sekarang adalah tujuan sebenarnya
	nextStage := scope.stage
	if tpl.Action == models.EmailActionRevise {
		// yang pindah stage adalah disposal yang dikeluarkan, bukan agreement
		nextStage = models.StageDisposalDraft
	}
	ctx := agreementRenderContext(scope, scope.number, tpl.Stage, nextStage, tpl.Action, userID)
	// Anggota yang dikeluarkan sudah tidak tercatat di agreement — ambil dari
	// jejak stage revisi yang baru saja dibuat user ini.
	numbers := scope.memberNumbers
	isRevise := tpl.Action == models.EmailActionRevise
	if isRevise {
		numbers = recentlyRevisedMembers(userID, scope.number)
	}

	return deliverAndLog(userID, &tpl, ctx, to, cc, req.AdditionalMessage,
		buildAgreementSummary(scope, numbers, ctx, isRevise))
}

func recentlyRevisedMembers(userID, agreementNumber string) []string {
	var numbers []string
	config.DB.Model(&models.TransactionStage{}).
		Where("actor_id = ? AND action = ? AND notes LIKE ? AND created_at >= ?",
			userID, models.ActionRevise, agreementRevisionNote(agreementNumber, "")+"%",
			time.Now().Add(-30*time.Minute)).
		Distinct().Pluck("transaction_number", &numbers)
	return numbers
}
