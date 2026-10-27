package application

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// LeasePort defines the lease application operations consumed by the payment bounded context.
type LeasePort interface {
	GetLeaseByID(ctx context.Context, leaseID string) (leaseapp.LeaseDTO, error)
	GetTenantByID(ctx context.Context, tenantID string) (leaseapp.TenantDTO, error)
	ListLeasesByEntity(ctx context.Context, entityID string) ([]leaseapp.LeaseDTO, error)
	ListUnpaidRentCallsByLease(ctx context.Context, leaseID string) ([]leasedomain.RentCall, error)
	SaveRentCalls(ctx context.Context, calls []leasedomain.RentCall) error
}

// RecordCashPaymentCmd carries the input for recording a tenant cash rent payment (PCG 530 Caisse).
type RecordCashPaymentCmd struct {
	LeaseID       string            `json:"lease_id"`
	AmountCents   money.AmountCents `json:"amount_cents"`
	Label         string            `json:"label"`
	ReceiptNumber string            `json:"receipt_number,omitempty"`
	PaymentDate   string            `json:"payment_date,omitempty"`
}

// RecordBankPaymentCmd carries the input for recording a bank transfer, check, or direct debit rent payment.
type RecordBankPaymentCmd struct {
	LeaseID     string            `json:"lease_id"`
	AmountCents money.AmountCents `json:"amount_cents"`
	Method      string            `json:"method,omitempty"`
	Label       string            `json:"label"`
	ExternalRef string            `json:"external_ref,omitempty"`
	PaymentDate string            `json:"payment_date,omitempty"`
}

// RecordCashDepositCmd carries the input for depositing cash from PCG 530 (Caisse) to PCG 580/512 (Banque).
type RecordCashDepositCmd struct {
	LegalEntityID string            `json:"legal_entity_id"`
	AmountCents   money.AmountCents `json:"amount_cents"`
	Label         string            `json:"label"`
	EntryDate     string            `json:"entry_date,omitempty"`
}

// PaymentReceiptDTO is an immutable projection of a recorded Payment and its RentAllocations.
type PaymentReceiptDTO struct {
	ID                 string                  `json:"id"`
	LegalEntityID      string                  `json:"legal_entity_id"`
	LeaseID            string                  `json:"lease_id"`
	TenantID           string                  `json:"tenant_id"`
	AmountCents        money.AmountCents       `json:"amount_cents"`
	UnallocatedCents   money.AmountCents       `json:"unallocated_cents"`
	Method             domain.PaymentMethod    `json:"method"`
	Label              string                  `json:"label"`
	ExternalRef        string                  `json:"external_ref,omitempty"`
	ReceiptNumber      string                  `json:"receipt_number,omitempty"`
	PaymentDate        time.Time               `json:"payment_date"`
	Allocations        []domain.RentAllocation `json:"allocations"`
	CashJournalEntryID string                  `json:"cash_journal_entry_id,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
}

// CashJournalEntryDTO is an immutable projection of a PCG 530 Cash Journal entry.
type CashJournalEntryDTO struct {
	ID                     string               `json:"id"`
	LegalEntityID          string               `json:"legal_entity_id"`
	PaymentID              *string              `json:"payment_id,omitempty"`
	AccountCode            string               `json:"account_code"`
	CounterpartAccountCode string               `json:"counterpart_account_code"`
	EntryType              domain.CashEntryType `json:"entry_type"`
	Label                  string               `json:"label"`
	DebitCents             money.AmountCents    `json:"debit_cents"`
	CreditCents            money.AmountCents    `json:"credit_cents"`
	ReceiptNumber          string               `json:"receipt_number,omitempty"`
	EntryDate              time.Time            `json:"entry_date"`
	CreatedAt              time.Time            `json:"created_at"`
}

// BatchImportResultDTO summarizes a CSV/OFX or Open Banking DSP2 batch import.
type BatchImportResultDTO struct {
	EntityID           string              `json:"entity_id"`
	ImportedCount      int                 `json:"imported_count"`
	TotalImportedCents money.AmountCents   `json:"total_imported_cents"`
	Payments           []PaymentReceiptDTO `json:"payments"`
}

// NegativePositionDTO represents a lease with unpaid rent calls (situation locative débitrice).
type NegativePositionDTO struct {
	LeaseID          string            `json:"lease_id"`
	LegalEntityID    string            `json:"legal_entity_id"`
	LotID            string            `json:"lot_id"`
	TenantID         string            `json:"tenant_id"`
	TenantName       string            `json:"tenant_name"`
	TotalUnpaidCents money.AmountCents `json:"total_unpaid_cents"`
	UnpaidMonths     []string          `json:"unpaid_months"`
	OldestDueDate    time.Time         `json:"oldest_due_date"`
	GLIPolicyNumber  string            `json:"gli_policy_number"`
	GLIEligible      bool              `json:"gli_eligible"`
}

// AccountingRecorder defines the optional port for posting automatic PCG journal entries on payments and cash deposits.
type AccountingRecorder interface {
	RecordPaymentCollectionEntry(
		ctx context.Context,
		entityID, paymentID, tenantID, method, label string,
		paymentDate time.Time,
		amountCents money.AmountCents,
	) error
	RecordCashDepositEntry(
		ctx context.Context,
		entityID, depositID, label string,
		depositDate time.Time,
		amountCents money.AmountCents,
	) error
}

// Service orchestrates cash/bank payment allocation, PCG 530 Cash Journal, CSV/OFX batch imports, and negative positions.
type Service struct {
	payments           domain.PaymentRepository
	cashJournal        domain.CashJournalRepository
	uow                domain.UnitOfWork
	leaseSvc           LeasePort
	allocSvc           *domain.AllocationService
	csvImporter        domain.StatementImporter
	bankProvider       domain.BankProvider
	accountingRecorder AccountingRecorder
}

// NewService constructs a payment application Service via constructor injection.
func NewService(
	payments domain.PaymentRepository,
	cashJournal domain.CashJournalRepository,
	uow domain.UnitOfWork,
	leaseSvc LeasePort,
	bankProvider domain.BankProvider,
	csvImporter domain.StatementImporter,
) *Service {
	return &Service{
		payments:     payments,
		cashJournal:  cashJournal,
		uow:          uow,
		leaseSvc:     leaseSvc,
		allocSvc:     domain.NewAllocationService(),
		csvImporter:  csvImporter,
		bankProvider: bankProvider,
	}
}

// WithAccountingRecorder attaches an optional AccountingRecorder to automatically post PCG journal entries on payments and cash deposits.
func (s *Service) WithAccountingRecorder(recorder AccountingRecorder) *Service {
	s.accountingRecorder = recorder
	return s
}

// RecordCashPayment records a physical cash rent payment, allocates it across unpaid rent calls
// (explicit month first, FIFO fallback), and posts a debit entry to the PCG 530 Cash Journal
// inside a single Unit of Work transaction across the Payment, RentCall, and CashJournalEntry aggregates.
func (s *Service) RecordCashPayment(ctx context.Context, cmd RecordCashPaymentCmd) (PaymentReceiptDTO, error) {
	leaseID := strings.TrimSpace(cmd.LeaseID)
	if leaseID == "" {
		return PaymentReceiptDTO{}, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	if cmd.AmountCents <= 0 {
		return PaymentReceiptDTO{}, fmt.Errorf("%w: amount_cents must be strictly positive", domain.ErrValidation)
	}

	paymentDate, err := parsePaymentDateOrNow(cmd.PaymentDate)
	if err != nil {
		return PaymentReceiptDTO{}, err
	}

	lease, err := s.leaseSvc.GetLeaseByID(ctx, leaseID)
	if err != nil {
		return PaymentReceiptDTO{}, err
	}

	receiptNum := strings.TrimSpace(cmd.ReceiptNumber)
	if receiptNum == "" {
		suffix, err := id.Generate("rcu")
		if err != nil {
			return PaymentReceiptDTO{}, err
		}
		receiptNum = fmt.Sprintf("CAISSE-%s-%s", paymentDate.Format("20060102"), suffix[len(suffix)-6:])
	}

	label := strings.TrimSpace(cmd.Label)
	if label == "" {
		label = "Encaissement loyer espèces"
	}

	var result PaymentReceiptDTO
	err = s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		dto, err := s.allocateAndPersistPayment(
			txCtx,
			repos.Payments,
			lease,
			cmd.AmountCents,
			domain.MethodCash,
			label,
			"",
			receiptNum,
			paymentDate,
		)
		if err != nil {
			return err
		}

		cashEntryID, err := id.Generate("cje")
		if err != nil {
			return err
		}
		paymentID := dto.ID
		cashEntry := domain.CashJournalEntry{
			ID:                     cashEntryID,
			LegalEntityID:          lease.LegalEntityID,
			PaymentID:              &paymentID,
			AccountCode:            "530",
			CounterpartAccountCode: "411",
			EntryType:              domain.CashEntryReceipt,
			Label:                  label,
			DebitCents:             cmd.AmountCents,
			CreditCents:            0,
			ReceiptNumber:          receiptNum,
			EntryDate:              paymentDate,
			CreatedAt:              dto.CreatedAt,
		}
		if err := repos.CashJournal.SaveCashJournalEntry(txCtx, cashEntry); err != nil {
			return err
		}

		dto.CashJournalEntryID = cashEntry.ID
		result = dto
		return nil
	})
	if err != nil {
		return PaymentReceiptDTO{}, err
	}
	if s.accountingRecorder != nil {
		if err := s.accountingRecorder.RecordPaymentCollectionEntry(
			ctx,
			result.LegalEntityID,
			result.ID,
			result.TenantID,
			string(result.Method),
			result.Label,
			result.PaymentDate,
			result.AmountCents,
		); err != nil {
			return PaymentReceiptDTO{}, err
		}
	}
	return result, nil
}

// RecordBankPayment records a bank transfer, check, or direct debit rent payment and allocates it
// across unpaid rent calls (explicit month first, FIFO fallback) inside a Unit of Work transaction.
func (s *Service) RecordBankPayment(ctx context.Context, cmd RecordBankPaymentCmd) (PaymentReceiptDTO, error) {
	leaseID := strings.TrimSpace(cmd.LeaseID)
	if leaseID == "" {
		return PaymentReceiptDTO{}, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	if cmd.AmountCents <= 0 {
		return PaymentReceiptDTO{}, fmt.Errorf("%w: amount_cents must be strictly positive", domain.ErrValidation)
	}

	methodStr := strings.ToUpper(strings.TrimSpace(cmd.Method))
	if methodStr == "" {
		methodStr = string(domain.MethodBankTransfer)
	}
	method := domain.PaymentMethod(methodStr)
	if !method.IsValid() || method == domain.MethodCash {
		return PaymentReceiptDTO{}, fmt.Errorf("%w: invalid bank payment method %q", domain.ErrValidation, cmd.Method)
	}

	paymentDate, err := parsePaymentDateOrNow(cmd.PaymentDate)
	if err != nil {
		return PaymentReceiptDTO{}, err
	}

	lease, err := s.leaseSvc.GetLeaseByID(ctx, leaseID)
	if err != nil {
		return PaymentReceiptDTO{}, err
	}

	label := strings.TrimSpace(cmd.Label)
	if label == "" {
		label = "Virement loyer"
	}

	var result PaymentReceiptDTO
	err = s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		dto, err := s.allocateAndPersistPayment(
			txCtx,
			repos.Payments,
			lease,
			cmd.AmountCents,
			method,
			label,
			strings.TrimSpace(cmd.ExternalRef),
			"",
			paymentDate,
		)
		if err != nil {
			return err
		}
		result = dto
		return nil
	})
	if err != nil {
		return PaymentReceiptDTO{}, err
	}
	if s.accountingRecorder != nil {
		if err := s.accountingRecorder.RecordPaymentCollectionEntry(
			ctx,
			result.LegalEntityID,
			result.ID,
			result.TenantID,
			string(result.Method),
			result.Label,
			result.PaymentDate,
			result.AmountCents,
		); err != nil {
			return PaymentReceiptDTO{}, err
		}
	}
	return result, nil
}

// RecordCashDepositToBank records a cash deposit from PCG 530 (Caisse credit) to PCG 580/512 (Banque debit).
func (s *Service) RecordCashDepositToBank(ctx context.Context, cmd RecordCashDepositCmd) (CashJournalEntryDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	if entityID == "" {
		return CashJournalEntryDTO{}, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	if cmd.AmountCents <= 0 {
		return CashJournalEntryDTO{}, fmt.Errorf("%w: amount_cents must be strictly positive", domain.ErrValidation)
	}

	entryDate, err := parsePaymentDateOrNow(cmd.EntryDate)
	if err != nil {
		return CashJournalEntryDTO{}, err
	}

	entryID, err := id.Generate("cje")
	if err != nil {
		return CashJournalEntryDTO{}, err
	}

	label := strings.TrimSpace(cmd.Label)
	if label == "" {
		label = "Remise d'espèces en banque"
	}

	entry := domain.CashJournalEntry{
		ID:                     entryID,
		LegalEntityID:          entityID,
		AccountCode:            "530",
		CounterpartAccountCode: "580",
		EntryType:              domain.CashEntryBankDeposit,
		Label:                  label,
		DebitCents:             0,
		CreditCents:            cmd.AmountCents,
		EntryDate:              entryDate,
		CreatedAt:              time.Now().UTC(),
	}
	if err := s.cashJournal.SaveCashJournalEntry(ctx, entry); err != nil {
		return CashJournalEntryDTO{}, err
	}
	if s.accountingRecorder != nil {
		if err := s.accountingRecorder.RecordCashDepositEntry(
			ctx,
			entry.LegalEntityID,
			entry.ID,
			entry.Label,
			entry.EntryDate,
			entry.CreditCents,
		); err != nil {
			return CashJournalEntryDTO{}, err
		}
	}
	return toCashJournalEntryDTO(entry), nil
}

// ListCashJournalByEntity returns all PCG 530 Cash Journal entries for a legal entity.
func (s *Service) ListCashJournalByEntity(ctx context.Context, entityID string) ([]CashJournalEntryDTO, error) {
	trimmed := strings.TrimSpace(entityID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	entries, err := s.cashJournal.ListCashJournalByEntity(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]CashJournalEntryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toCashJournalEntryDTO(e))
	}
	return out, nil
}

// ListPaymentsByLease returns all recorded payments for a lease.
func (s *Service) ListPaymentsByLease(ctx context.Context, leaseID string) ([]PaymentReceiptDTO, error) {
	trimmed := strings.TrimSpace(leaseID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	payments, err := s.payments.ListPaymentsByLease(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentReceiptDTO, 0, len(payments))
	for _, p := range payments {
		out = append(out, toPaymentReceiptDTO(p))
	}
	return out, nil
}

// ImportPaymentsCSV parses a CSV or OFX bank statement stream and allocates each transaction
// for the given legal entity.
func (s *Service) ImportPaymentsCSV(ctx context.Context, entityID string, r io.Reader) (BatchImportResultDTO, error) {
	trimmedEntityID := strings.TrimSpace(entityID)
	if trimmedEntityID == "" {
		return BatchImportResultDTO{}, fmt.Errorf("%w: entity_id is required", domain.ErrValidation)
	}
	if s.csvImporter == nil {
		return BatchImportResultDTO{}, fmt.Errorf("%w: statement importer is not configured", domain.ErrValidation)
	}

	rows, err := s.csvImporter.ParseTransactions(r)
	if err != nil {
		return BatchImportResultDTO{}, err
	}

	return s.processBankRows(ctx, trimmedEntityID, rows)
}

// SyncOpenBankingDSP2 synchronizes bank transactions via the optional DSP2 Open Banking provider.
func (s *Service) SyncOpenBankingDSP2(ctx context.Context, entityID string, since time.Time) (BatchImportResultDTO, error) {
	trimmedEntityID := strings.TrimSpace(entityID)
	if trimmedEntityID == "" {
		return BatchImportResultDTO{}, fmt.Errorf("%w: entity_id is required", domain.ErrValidation)
	}
	if s.bankProvider == nil || !s.bankProvider.Enabled() {
		return BatchImportResultDTO{}, domain.ErrOpenBankingDisabled
	}
	rows, err := s.bankProvider.FetchTransactions(ctx, trimmedEntityID, since)
	if err != nil {
		return BatchImportResultDTO{}, err
	}
	return s.processBankRows(ctx, trimmedEntityID, rows)
}

// ListNegativePositions returns all leases belonging to entityID that have unpaid rent calls,
// including total unpaid amount, unpaid period months, oldest due date, and GLI policy eligibility.
func (s *Service) ListNegativePositions(ctx context.Context, entityID string) ([]NegativePositionDTO, error) {
	trimmedEntityID := strings.TrimSpace(entityID)
	if trimmedEntityID == "" {
		return nil, fmt.Errorf("%w: entity_id is required", domain.ErrValidation)
	}

	leases, err := s.leaseSvc.ListLeasesByEntity(ctx, trimmedEntityID)
	if err != nil {
		return nil, err
	}

	var positions []NegativePositionDTO
	for _, lease := range leases {
		unpaidCalls, err := s.leaseSvc.ListUnpaidRentCallsByLease(ctx, lease.ID)
		if err != nil {
			return nil, err
		}
		if len(unpaidCalls) == 0 {
			continue
		}

		sort.SliceStable(unpaidCalls, func(i, j int) bool {
			return unpaidCalls[i].PeriodMonth < unpaidCalls[j].PeriodMonth
		})

		var totalUnpaid money.AmountCents
		months := make([]string, 0, len(unpaidCalls))
		oldestDue := unpaidCalls[0].DueDate

		for _, rc := range unpaidCalls {
			if rc.RemainingDueCents <= 0 {
				continue
			}
			totalUnpaid = totalUnpaid.Add(rc.RemainingDueCents)
			months = append(months, rc.PeriodMonth)
			if oldestDue.IsZero() || (!rc.DueDate.IsZero() && rc.DueDate.Before(oldestDue)) {
				oldestDue = rc.DueDate
			}
		}

		if totalUnpaid <= 0 {
			continue
		}

		tenantName := ""
		if tenant, err := s.leaseSvc.GetTenantByID(ctx, lease.TenantID); err == nil {
			tenantName = tenant.FullName
		}

		positions = append(positions, NegativePositionDTO{
			LeaseID:          lease.ID,
			LegalEntityID:    lease.LegalEntityID,
			LotID:            lease.LotID,
			TenantID:         lease.TenantID,
			TenantName:       tenantName,
			TotalUnpaidCents: totalUnpaid,
			UnpaidMonths:     months,
			OldestDueDate:    oldestDue,
			GLIPolicyNumber:  lease.GLIPolicyNumber,
			GLIEligible:      strings.TrimSpace(lease.GLIPolicyNumber) != "",
		})
	}

	return positions, nil
}

func (s *Service) processBankRows(
	ctx context.Context,
	entityID string,
	rows []domain.BankTransactionRow,
) (BatchImportResultDTO, error) {
	leases, err := s.leaseSvc.ListLeasesByEntity(ctx, entityID)
	if err != nil {
		return BatchImportResultDTO{}, err
	}

	leasesByID := make(map[string]leaseapp.LeaseDTO, len(leases))
	tenantNamesByLease := make(map[string]string, len(leases))
	for _, l := range leases {
		leasesByID[l.ID] = l
		if t, err := s.leaseSvc.GetTenantByID(ctx, l.TenantID); err == nil {
			tenantNamesByLease[l.ID] = strings.ToLower(t.FullName)
		}
	}

	type resolvedBankRow struct {
		row   domain.BankTransactionRow
		lease leaseapp.LeaseDTO
	}

	var resolvedRows []resolvedBankRow
	for _, row := range rows {
		if row.AmountCents <= 0 {
			continue
		}
		leaseID := strings.TrimSpace(row.LeaseID)
		if leaseID == "" {
			lowerLabel := strings.ToLower(row.Label)
			for _, l := range leases {
				if tName := tenantNamesByLease[l.ID]; tName != "" && strings.Contains(lowerLabel, tName) {
					leaseID = l.ID
					break
				}
			}
		}
		if leaseID == "" {
			return BatchImportResultDTO{}, fmt.Errorf("%w: could not resolve lease_id for bank transaction %q", domain.ErrValidation, row.Label)
		}
		lease, ok := leasesByID[leaseID]
		if !ok {
			loadedLease, err := s.leaseSvc.GetLeaseByID(ctx, leaseID)
			if err != nil {
				return BatchImportResultDTO{}, err
			}
			lease = loadedLease
			leasesByID[leaseID] = lease
		}
		resolvedRows = append(resolvedRows, resolvedBankRow{
			row:   row,
			lease: lease,
		})
	}

	var (
		receipts      []PaymentReceiptDTO
		totalImported money.AmountCents
	)

	err = s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		existingPayments, err := repos.Payments.ListPaymentsByEntity(txCtx, entityID)
		if err != nil {
			return err
		}
		seenRefs := make(map[string]bool, len(existingPayments))
		for _, p := range existingPayments {
			if ref := strings.TrimSpace(p.ExternalRef); ref != "" {
				seenRefs[ref] = true
			}
		}

		for _, item := range resolvedRows {
			row := item.row
			extRef := strings.TrimSpace(row.ExternalRef)
			if extRef != "" {
				if seenRefs[extRef] {
					continue
				}
				seenRefs[extRef] = true
			}

			label := strings.TrimSpace(row.Label)
			if label == "" {
				label = "Virement loyer"
			}
			paymentDate := row.PaymentDate.UTC()
			if paymentDate.IsZero() {
				paymentDate = time.Now().UTC()
			}

			dto, err := s.allocateAndPersistPayment(
				txCtx,
				repos.Payments,
				item.lease,
				row.AmountCents,
				domain.MethodBankTransfer,
				label,
				extRef,
				"",
				paymentDate,
			)
			if err != nil {
				return err
			}
			receipts = append(receipts, dto)
			totalImported = totalImported.Add(row.AmountCents)
		}
		return nil
	})
	if err != nil {
		return BatchImportResultDTO{}, err
	}

	if s.accountingRecorder != nil {
		for _, receipt := range receipts {
			if err := s.accountingRecorder.RecordPaymentCollectionEntry(
				ctx,
				receipt.LegalEntityID,
				receipt.ID,
				receipt.TenantID,
				string(receipt.Method),
				receipt.Label,
				receipt.PaymentDate,
				receipt.AmountCents,
			); err != nil {
				return BatchImportResultDTO{}, err
			}
		}
	}

	if receipts == nil {
		receipts = []PaymentReceiptDTO{}
	}

	return BatchImportResultDTO{
		EntityID:           entityID,
		ImportedCount:      len(receipts),
		TotalImportedCents: totalImported,
		Payments:           receipts,
	}, nil
}

func (s *Service) allocateAndPersistPayment(
	ctx context.Context,
	paymentsRepo domain.PaymentRepository,
	lease leaseapp.LeaseDTO,
	amountCents money.AmountCents,
	method domain.PaymentMethod,
	label string,
	externalRef string,
	receiptNumber string,
	paymentDate time.Time,
) (PaymentReceiptDTO, error) {
	unpaidCalls, err := s.leaseSvc.ListUnpaidRentCallsByLease(ctx, lease.ID)
	if err != nil {
		return PaymentReceiptDTO{}, err
	}

	allocs, updatedCalls := s.allocSvc.AllocatePayment(unpaidCalls, amountCents, label, paymentDate)

	if len(allocs) > 0 {
		touched := make(map[string]bool, len(allocs))
		for _, a := range allocs {
			touched[a.RentCallID] = true
		}
		modifiedCalls := make([]leasedomain.RentCall, 0, len(allocs))
		for _, rc := range updatedCalls {
			if touched[rc.ID] {
				modifiedCalls = append(modifiedCalls, rc)
			}
		}
		if len(modifiedCalls) > 0 {
			if err := s.leaseSvc.SaveRentCalls(ctx, modifiedCalls); err != nil {
				return PaymentReceiptDTO{}, err
			}
		}
	}

	paymentID, err := id.Generate("pay")
	if err != nil {
		return PaymentReceiptDTO{}, err
	}

	now := time.Now().UTC()
	var totalApplied money.AmountCents
	for i := range allocs {
		allocID, err := id.Generate("alc")
		if err != nil {
			return PaymentReceiptDTO{}, err
		}
		allocs[i].ID = allocID
		allocs[i].PaymentID = paymentID
		allocs[i].CreatedAt = now
		totalApplied = totalApplied.Add(allocs[i].AppliedCents)
	}

	unallocated := amountCents.Sub(totalApplied)
	if unallocated < 0 {
		unallocated = 0
	}

	payment := domain.Payment{
		ID:               paymentID,
		LegalEntityID:    lease.LegalEntityID,
		LeaseID:          lease.ID,
		TenantID:         lease.TenantID,
		AmountCents:      amountCents,
		UnallocatedCents: unallocated,
		Method:           method,
		Label:            label,
		ExternalRef:      externalRef,
		ReceiptNumber:    receiptNumber,
		PaymentDate:      paymentDate,
		Allocations:      allocs,
		CreatedAt:        now,
	}

	if err := paymentsRepo.SavePayment(ctx, payment); err != nil {
		return PaymentReceiptDTO{}, err
	}

	return toPaymentReceiptDTO(payment), nil
}

func toPaymentReceiptDTO(p domain.Payment) PaymentReceiptDTO {
	allocs := p.Allocations
	if allocs == nil {
		allocs = []domain.RentAllocation{}
	}
	return PaymentReceiptDTO{
		ID:               p.ID,
		LegalEntityID:    p.LegalEntityID,
		LeaseID:          p.LeaseID,
		TenantID:         p.TenantID,
		AmountCents:      p.AmountCents,
		UnallocatedCents: p.UnallocatedCents,
		Method:           p.Method,
		Label:            p.Label,
		ExternalRef:      p.ExternalRef,
		ReceiptNumber:    p.ReceiptNumber,
		PaymentDate:      p.PaymentDate,
		Allocations:      allocs,
		CreatedAt:        p.CreatedAt,
	}
}

func toCashJournalEntryDTO(e domain.CashJournalEntry) CashJournalEntryDTO {
	return CashJournalEntryDTO{
		ID:                     e.ID,
		LegalEntityID:          e.LegalEntityID,
		PaymentID:              e.PaymentID,
		AccountCode:            e.AccountCode,
		CounterpartAccountCode: e.CounterpartAccountCode,
		EntryType:              e.EntryType,
		Label:                  e.Label,
		DebitCents:             e.DebitCents,
		CreditCents:            e.CreditCents,
		ReceiptNumber:          e.ReceiptNumber,
		EntryDate:              e.EntryDate,
		CreatedAt:              e.CreatedAt,
	}
}

func parsePaymentDateOrNow(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Now().UTC(), nil
	}
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"02/01/2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: invalid payment_date %q (expected YYYY-MM-DD)", domain.ErrValidation, raw)
}

