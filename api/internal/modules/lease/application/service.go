package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// CreateTenantCmd carries the input for creating a tenant.
type CreateTenantCmd struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
}

// TenantDTO is an immutable projection of a domain.Tenant.
type TenantDTO struct {
	ID         string     `json:"id"`
	FullName   string     `json:"full_name"`
	Email      string     `json:"email"`
	Phone      string     `json:"phone"`
	Address    string     `json:"address"`
	Archived   bool       `json:"archived"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CreateLeaseCmd carries the input for creating a residential or commercial lease.
type CreateLeaseCmd struct {
	LegalEntityID        string            `json:"legal_entity_id"`
	LotID                string            `json:"lot_id"`
	TenantID             string            `json:"tenant_id"`
	LeaseType            string            `json:"lease_type"`
	RentNetCents         money.AmountCents `json:"rent_net_cents"`
	ChargesCents         money.AmountCents `json:"charges_cents"`
	SubjectToVAT         bool              `json:"subject_to_vat"`
	VATRateBasisPoints   int64             `json:"vat_rate_basis_points"`
	SecurityDepositCents money.AmountCents `json:"security_deposit_cents"`
	DueDayOfMonth        int               `json:"due_day_of_month"`
	GLIPolicyNumber      string            `json:"gli_policy_number"`
	GuarantorName        string            `json:"guarantor_name"`
	StartDate            string            `json:"start_date,omitempty"`
	EndDate              string            `json:"end_date,omitempty"`
}

// LeaseDTO is an immutable projection of a domain.Lease.
type LeaseDTO struct {
	ID                   string            `json:"id"`
	LegalEntityID        string            `json:"legal_entity_id"`
	LotID                string            `json:"lot_id"`
	TenantID             string            `json:"tenant_id"`
	LeaseType            domain.LeaseType  `json:"lease_type"`
	RentNetCents         money.AmountCents `json:"rent_net_cents"`
	ChargesCents         money.AmountCents `json:"charges_cents"`
	SubjectToVAT         bool              `json:"subject_to_vat"`
	VATRateBasisPoints   int64             `json:"vat_rate_basis_points"`
	SecurityDepositCents money.AmountCents `json:"security_deposit_cents"`
	DueDayOfMonth        int               `json:"due_day_of_month"`
	GLIPolicyNumber      string            `json:"gli_policy_number"`
	GuarantorName        string            `json:"guarantor_name"`
	StartDate            time.Time         `json:"start_date"`
	EndDate              *time.Time        `json:"end_date,omitempty"`
	Active               bool              `json:"active"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

// RentCallDTO is an immutable projection of a domain.RentCall.
type RentCallDTO struct {
	ID                string                `json:"id"`
	LeaseID           string                `json:"lease_id"`
	LegalEntityID     string                `json:"legal_entity_id"`
	TenantID          string                `json:"tenant_id"`
	PeriodMonth       string                `json:"period_month"`
	RentNetCents      money.AmountCents     `json:"rent_net_cents"`
	VATCents          money.AmountCents     `json:"vat_cents"`
	ChargesCents      money.AmountCents     `json:"charges_cents"`
	TotalDueCents     money.AmountCents     `json:"total_due_cents"`
	RemainingDueCents money.AmountCents     `json:"remaining_due_cents"`
	Status            domain.RentCallStatus `json:"status"`
	DueDate           time.Time             `json:"due_date"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
}

// AccountingRecorder defines the optional port for posting automatic PCG journal entries when rent calls are generated.
type AccountingRecorder interface {
	RecordRentCallEntry(
		ctx context.Context,
		entityID, rentCallID, leaseID, tenantID, periodMonth string,
		dueDate time.Time,
		rentNetCents, vatCents, chargesCents, totalDueCents money.AmountCents,
	) error
}

// Service orchestrates tenants, leases, and monthly rent calls.
type Service struct {
	tenantRepo         domain.TenantRepository
	leaseRepo          domain.LeaseRepository
	rentCallRepo       domain.RentCallRepository
	uow                domain.UnitOfWork
	accountingRecorder AccountingRecorder
}

// NewService constructs a lease application Service via constructor injection of per-aggregate repositories and UnitOfWork.
func NewService(
	tenantRepo domain.TenantRepository,
	leaseRepo domain.LeaseRepository,
	rentCallRepo domain.RentCallRepository,
	uow domain.UnitOfWork,
) *Service {
	return &Service{
		tenantRepo:   tenantRepo,
		leaseRepo:    leaseRepo,
		rentCallRepo: rentCallRepo,
		uow:          uow,
	}
}

// WithAccountingRecorder attaches an optional AccountingRecorder to automatically post PCG VE journal entries on rent call generation.
func (s *Service) WithAccountingRecorder(recorder AccountingRecorder) *Service {
	s.accountingRecorder = recorder
	return s
}

// CreateTenant validates and persists a new Tenant.
func (s *Service) CreateTenant(ctx context.Context, cmd CreateTenantCmd) (TenantDTO, error) {
	fullName := strings.TrimSpace(cmd.FullName)
	if fullName == "" {
		return TenantDTO{}, fmt.Errorf("%w: full_name is required", domain.ErrValidation)
	}

	tenantID, err := id.Generate("ten")
	if err != nil {
		return TenantDTO{}, err
	}

	now := time.Now().UTC()
	tenant := domain.Tenant{
		ID:        tenantID,
		FullName:  fullName,
		Email:     strings.TrimSpace(cmd.Email),
		Phone:     strings.TrimSpace(cmd.Phone),
		Address:   strings.TrimSpace(cmd.Address),
		Archived:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.tenantRepo.CreateTenant(ctx, tenant); err != nil {
		return TenantDTO{}, err
	}

	return toTenantDTO(tenant), nil
}

// ArchiveTenant marks an existing tenant as archived while preserving historical lease and payment records.
func (s *Service) ArchiveTenant(ctx context.Context, tenantID string) error {
	trimmed := strings.TrimSpace(tenantID)
	if trimmed == "" {
		return fmt.Errorf("%w: tenant_id is required", domain.ErrValidation)
	}
	if _, err := s.tenantRepo.FindTenantByID(ctx, trimmed); err != nil {
		return err
	}
	return s.tenantRepo.ArchiveTenant(ctx, trimmed, time.Now().UTC())
}

// GetTenantByID retrieves a tenant by ID.
func (s *Service) GetTenantByID(ctx context.Context, tenantID string) (TenantDTO, error) {
	trimmed := strings.TrimSpace(tenantID)
	if trimmed == "" {
		return TenantDTO{}, fmt.Errorf("%w: tenant_id is required", domain.ErrValidation)
	}
	t, err := s.tenantRepo.FindTenantByID(ctx, trimmed)
	if err != nil {
		return TenantDTO{}, err
	}
	return toTenantDTO(*t), nil
}

// ListTenants lists present tenants, optionally including archived tenants.
func (s *Service) ListTenants(ctx context.Context, includeArchived bool) ([]TenantDTO, error) {
	tenants, err := s.tenantRepo.ListTenants(ctx, includeArchived)
	if err != nil {
		return nil, err
	}
	out := make([]TenantDTO, 0, len(tenants))
	for _, t := range tenants {
		out = append(out, toTenantDTO(t))
	}
	return out, nil
}

// CreateLease validates and persists a new residential or commercial lease.
func (s *Service) CreateLease(ctx context.Context, cmd CreateLeaseCmd) (LeaseDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	lotID := strings.TrimSpace(cmd.LotID)
	tenantID := strings.TrimSpace(cmd.TenantID)
	if entityID == "" || lotID == "" || tenantID == "" {
		return LeaseDTO{}, fmt.Errorf("%w: legal_entity_id, lot_id, and tenant_id are required", domain.ErrValidation)
	}

	if _, err := s.tenantRepo.FindTenantByID(ctx, tenantID); err != nil {
		return LeaseDTO{}, err
	}

	leaseType := domain.LeaseType(strings.ToUpper(strings.TrimSpace(cmd.LeaseType)))
	if !leaseType.IsValid() {
		return LeaseDTO{}, fmt.Errorf("%w: unsupported lease_type %q", domain.ErrValidation, cmd.LeaseType)
	}

	if cmd.RentNetCents.IsNegative() || cmd.ChargesCents.IsNegative() || cmd.SecurityDepositCents.IsNegative() {
		return LeaseDTO{}, fmt.Errorf("%w: monetary lease amounts cannot be negative", domain.ErrValidation)
	}

	vatRateBps := cmd.VATRateBasisPoints
	if cmd.SubjectToVAT {
		if vatRateBps <= 0 || vatRateBps > 10000 {
			return LeaseDTO{}, fmt.Errorf("%w: vat_rate_basis_points must be between 1 and 10000 when subject_to_vat is true", domain.ErrValidation)
		}
	} else {
		vatRateBps = 0
	}

	dueDay := cmd.DueDayOfMonth
	if dueDay == 0 {
		dueDay = 1
	}
	if dueDay < 1 || dueDay > 31 {
		return LeaseDTO{}, fmt.Errorf("%w: due_day_of_month must be between 1 and 31", domain.ErrValidation)
	}

	now := time.Now().UTC()
	startDate := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	if strings.TrimSpace(cmd.StartDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(cmd.StartDate))
		if err != nil {
			return LeaseDTO{}, fmt.Errorf("%w: invalid start_date (expected YYYY-MM-DD): %v", domain.ErrValidation, err)
		}
		startDate = parsed.UTC()
	}

	var endDate *time.Time
	if strings.TrimSpace(cmd.EndDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(cmd.EndDate))
		if err != nil {
			return LeaseDTO{}, fmt.Errorf("%w: invalid end_date (expected YYYY-MM-DD): %v", domain.ErrValidation, err)
		}
		endUTC := parsed.UTC()
		endDate = &endUTC
	}

	leaseID, err := id.Generate("lse")
	if err != nil {
		return LeaseDTO{}, err
	}

	lease := domain.Lease{
		ID:                   leaseID,
		LegalEntityID:        entityID,
		LotID:                lotID,
		TenantID:             tenantID,
		LeaseType:            leaseType,
		RentNetCents:         cmd.RentNetCents,
		ChargesCents:         cmd.ChargesCents,
		SubjectToVAT:         cmd.SubjectToVAT,
		VATRateBasisPoints:   vatRateBps,
		SecurityDepositCents: cmd.SecurityDepositCents,
		DueDayOfMonth:        dueDay,
		GLIPolicyNumber:      strings.TrimSpace(cmd.GLIPolicyNumber),
		GuarantorName:        strings.TrimSpace(cmd.GuarantorName),
		StartDate:            startDate,
		EndDate:              endDate,
		Active:               true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.leaseRepo.CreateLease(ctx, lease); err != nil {
		return LeaseDTO{}, err
	}

	return toLeaseDTO(lease), nil
}

// GetLeaseByID retrieves a lease by ID.
func (s *Service) GetLeaseByID(ctx context.Context, leaseID string) (LeaseDTO, error) {
	trimmed := strings.TrimSpace(leaseID)
	if trimmed == "" {
		return LeaseDTO{}, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	l, err := s.leaseRepo.FindLeaseByID(ctx, trimmed)
	if err != nil {
		return LeaseDTO{}, err
	}
	return toLeaseDTO(*l), nil
}

// ListLeases returns all leases across all entities.
func (s *Service) ListLeases(ctx context.Context) ([]LeaseDTO, error) {
	leases, err := s.leaseRepo.ListLeases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LeaseDTO, 0, len(leases))
	for _, l := range leases {
		out = append(out, toLeaseDTO(l))
	}
	return out, nil
}

// ListLeasesByEntity returns all leases for a specific legal entity.
func (s *Service) ListLeasesByEntity(ctx context.Context, entityID string) ([]LeaseDTO, error) {
	trimmed := strings.TrimSpace(entityID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	leases, err := s.leaseRepo.ListLeasesByEntity(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]LeaseDTO, 0, len(leases))
	for _, l := range leases {
		out = append(out, toLeaseDTO(l))
	}
	return out, nil
}

// GenerateMonthlyRentCalls generates (or returns existing) monthly rent calls for all active leases
// belonging to entityID for periodMonth (YYYY-MM).
func (s *Service) GenerateMonthlyRentCalls(ctx context.Context, entityID string, periodMonth string) ([]RentCallDTO, error) {
	trimmedEntityID := strings.TrimSpace(entityID)
	trimmedPeriod := strings.TrimSpace(periodMonth)
	if trimmedEntityID == "" || trimmedPeriod == "" {
		return nil, fmt.Errorf("%w: entity_id and period_month are required", domain.ErrValidation)
	}

	periodTime, err := time.Parse("2006-01", trimmedPeriod)
	if err != nil {
		return nil, fmt.Errorf("%w: period_month must be in YYYY-MM format: %v", domain.ErrValidation, err)
	}

	var (
		result []RentCallDTO
		toSave []domain.RentCall
	)
	err = s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		leases, err := repos.Leases.ListLeasesByEntity(txCtx, trimmedEntityID)
		if err != nil {
			return err
		}

		now := time.Now().UTC()

		for _, lease := range leases {
			if !lease.Active {
				continue
			}
			if !lease.StartDate.IsZero() {
				startMonth := lease.StartDate.UTC().Format("2006-01")
				if trimmedPeriod < startMonth {
					continue
				}
			}
			if lease.EndDate != nil && !lease.EndDate.IsZero() {
				endMonth := lease.EndDate.UTC().Format("2006-01")
				if trimmedPeriod > endMonth {
					continue
				}
			}

			existing, err := repos.RentCalls.FindRentCallByLeaseAndPeriod(txCtx, lease.ID, trimmedPeriod)
			if err == nil && existing != nil {
				result = append(result, toRentCallDTO(*existing))
				continue
			}
			if err != nil && !errors.Is(err, domain.ErrRentCallNotFound) {
				return err
			}

			rcID, err := id.Generate("rc")
			if err != nil {
				return err
			}

			vatCents, totalDueCents := lease.ComputeRentCallAmounts()
			dueDate := computeDueDate(periodTime.Year(), periodTime.Month(), lease.DueDayOfMonth)

			rc := domain.RentCall{
				ID:                rcID,
				LeaseID:           lease.ID,
				LegalEntityID:     lease.LegalEntityID,
				TenantID:          lease.TenantID,
				PeriodMonth:       trimmedPeriod,
				RentNetCents:      lease.RentNetCents,
				VATCents:          vatCents,
				ChargesCents:      lease.ChargesCents,
				TotalDueCents:     totalDueCents,
				RemainingDueCents: totalDueCents,
				Status:            domain.StatusPending,
				DueDate:           dueDate,
				CreatedAt:         now,
				UpdatedAt:         now,
			}

			toSave = append(toSave, rc)
			result = append(result, toRentCallDTO(rc))
		}

		if len(toSave) > 0 {
			if err := repos.RentCalls.SaveRentCalls(txCtx, toSave); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.accountingRecorder != nil {
		for _, rc := range toSave {
			if err := s.accountingRecorder.RecordRentCallEntry(
				ctx,
				rc.LegalEntityID,
				rc.ID,
				rc.LeaseID,
				rc.TenantID,
				rc.PeriodMonth,
				rc.DueDate,
				rc.RentNetCents,
				rc.VATCents,
				rc.ChargesCents,
				rc.TotalDueCents,
			); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// ListUnpaidRentCallsByLease returns all unpaid/partially paid rent calls for a lease ordered oldest first.
func (s *Service) ListUnpaidRentCallsByLease(ctx context.Context, leaseID string) ([]domain.RentCall, error) {
	trimmed := strings.TrimSpace(leaseID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	return s.rentCallRepo.ListUnpaidRentCallsByLease(ctx, trimmed)
}

// SaveRentCalls persists or updates a batch of rent calls (used by payment allocation and GLI workflows).
func (s *Service) SaveRentCalls(ctx context.Context, calls []domain.RentCall) error {
	return s.rentCallRepo.SaveRentCalls(ctx, calls)
}

// ListRentCallsByLease lists all rent calls for a lease.
func (s *Service) ListRentCallsByLease(ctx context.Context, leaseID string) ([]RentCallDTO, error) {
	trimmed := strings.TrimSpace(leaseID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	calls, err := s.rentCallRepo.ListRentCallsByLease(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]RentCallDTO, 0, len(calls))
	for _, rc := range calls {
		out = append(out, toRentCallDTO(rc))
	}
	return out, nil
}

// ListRentCallsByEntity lists all rent calls for a legal entity.
func (s *Service) ListRentCallsByEntity(ctx context.Context, entityID string) ([]RentCallDTO, error) {
	trimmed := strings.TrimSpace(entityID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}
	calls, err := s.rentCallRepo.ListRentCallsByEntity(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]RentCallDTO, 0, len(calls))
	for _, rc := range calls {
		out = append(out, toRentCallDTO(rc))
	}
	return out, nil
}

func computeDueDate(year int, month time.Month, dueDay int) time.Time {
	if dueDay < 1 {
		dueDay = 1
	}
	lastDayOfMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if dueDay > lastDayOfMonth {
		dueDay = lastDayOfMonth
	}
	return time.Date(year, month, dueDay, 0, 0, 0, 0, time.UTC)
}

func toTenantDTO(t domain.Tenant) TenantDTO {
	return TenantDTO{
		ID:         t.ID,
		FullName:   t.FullName,
		Email:      t.Email,
		Phone:      t.Phone,
		Address:    t.Address,
		Archived:   t.Archived,
		ArchivedAt: t.ArchivedAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func toLeaseDTO(l domain.Lease) LeaseDTO {
	return LeaseDTO{
		ID:                   l.ID,
		LegalEntityID:        l.LegalEntityID,
		LotID:                l.LotID,
		TenantID:             l.TenantID,
		LeaseType:            l.LeaseType,
		RentNetCents:         l.RentNetCents,
		ChargesCents:         l.ChargesCents,
		SubjectToVAT:         l.SubjectToVAT,
		VATRateBasisPoints:   l.VATRateBasisPoints,
		SecurityDepositCents: l.SecurityDepositCents,
		DueDayOfMonth:        l.DueDayOfMonth,
		GLIPolicyNumber:      l.GLIPolicyNumber,
		GuarantorName:        l.GuarantorName,
		StartDate:            l.StartDate,
		EndDate:              l.EndDate,
		Active:               l.Active,
		CreatedAt:            l.CreatedAt,
		UpdatedAt:            l.UpdatedAt,
	}
}

func toRentCallDTO(rc domain.RentCall) RentCallDTO {
	return RentCallDTO{
		ID:                rc.ID,
		LeaseID:           rc.LeaseID,
		LegalEntityID:     rc.LegalEntityID,
		TenantID:          rc.TenantID,
		PeriodMonth:       rc.PeriodMonth,
		RentNetCents:      rc.RentNetCents,
		VATCents:          rc.VATCents,
		ChargesCents:      rc.ChargesCents,
		TotalDueCents:     rc.TotalDueCents,
		RemainingDueCents: rc.RemainingDueCents,
		Status:            rc.Status,
		DueDate:           rc.DueDate,
		CreatedAt:         rc.CreatedAt,
		UpdatedAt:         rc.UpdatedAt,
	}
}


