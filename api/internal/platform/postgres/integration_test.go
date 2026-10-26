package postgres_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	accountingdomain "github.com/yannamsellem/gestion-immo/internal/modules/accounting/domain"
	accountingpg "github.com/yannamsellem/gestion-immo/internal/modules/accounting/infrastructure/postgres"
	authdomain "github.com/yannamsellem/gestion-immo/internal/modules/auth/domain"
	authpg "github.com/yannamsellem/gestion-immo/internal/modules/auth/infrastructure/postgres"
	documentdomain "github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	documentpg "github.com/yannamsellem/gestion-immo/internal/modules/document/infrastructure/postgres"
	inspectiondomain "github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	inspectionpg "github.com/yannamsellem/gestion-immo/internal/modules/inspection/infrastructure/postgres"
	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	leasepg "github.com/yannamsellem/gestion-immo/internal/modules/lease/infrastructure/postgres"
	paymentdomain "github.com/yannamsellem/gestion-immo/internal/modules/payment/domain"
	paymentpg "github.com/yannamsellem/gestion-immo/internal/modules/payment/infrastructure/postgres"
	portfoliodomain "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	portfoliopg "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/infrastructure/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestPostgresIntegration_MigrationsTransactionsAndRepositories(t *testing.T) {
	dbURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}

	if err := postgres.RunMigrations(dbURL, "../../../migrations", "up"); err != nil {
		t.Fatalf("RunMigrations up failed: %v", err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, secret.New(dbURL))
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	_, err = pool.Exec(ctx, `
		TRUNCATE TABLE
			annual_account_reports,
			account_balance_lines,
			journal_lines,
			journal_entries,
			incidents,
			inspection_items,
			inspections,
			inspection_templates,
			gli_dossiers,
			stored_files,
			document_templates,
			cash_journal_entries,
			rent_allocations,
			payments,
			rent_calls,
			leases,
			tenants,
			lots,
			properties,
			shareholders,
			legal_entities,
			mcp_api_tokens,
			refresh_tokens,
			users
		CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	// 1. Verify WithinTransaction commit and rollback on error
	userRepo := authpg.NewUserRepo(pool)
	sentinelErr := errors.New("simulated rollback error")
	err = postgres.WithinTransaction(ctx, pool, func(txCtx context.Context) error {
		txUserRepo := authpg.NewUserRepo(postgres.Conn(txCtx, pool))
		if err := txUserRepo.CreateUser(txCtx, authdomain.User{
			ID:           "usr_rolled_back",
			FullName:     "Rolled Back",
			Email:        "rollback@example.fr",
			PasswordHash: "hash",
			Role:         authdomain.RoleManager,
			Status:       authdomain.StatusPendingApproval,
			CreatedAt:    now,
			UpdatedAt:    now,
		}); err != nil {
			return err
		}
		return sentinelErr
	})
	if !errors.Is(err, sentinelErr) {
		t.Fatalf("expected sentinelErr from WithinTransaction, got %v", err)
	}
	if _, err := userRepo.FindUserByID(ctx, "usr_rolled_back"); !errors.Is(err, authdomain.ErrUserNotFound) {
		t.Fatalf("expected rolled back user to not exist, got err=%v", err)
	}

	// 2. Auth repositories (UserRepo, RefreshTokenRepo, MCPTokenRepo)
	admin := authdomain.User{
		ID:           "usr_admin_1",
		FullName:     "Yann Admin",
		Email:        "Yann.Admin@Example.fr",
		PasswordHash: "argon2id$hash",
		Role:         authdomain.RoleSuperAdmin,
		Status:       authdomain.StatusApproved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := userRepo.CreateUser(ctx, admin); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	foundByEmail, err := userRepo.FindUserByEmail(ctx, "YANN.ADMIN@EXAMPLE.FR")
	if err != nil || foundByEmail.ID != admin.ID {
		t.Fatalf("FindUserByEmail failed: got %+v (err=%v)", foundByEmail, err)
	}

	refreshRepo := authpg.NewRefreshTokenRepo(pool, pool)
	if err := refreshRepo.SaveRefreshToken(ctx, authdomain.RefreshTokenRecord{
		ID:        "rt_1",
		UserID:    admin.ID,
		FamilyID:  "fam_1",
		TokenHash: "hash_rt_1",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveRefreshToken failed: %v", err)
	}
	if err := refreshRepo.RotateRefreshToken(ctx, "rt_1", authdomain.RefreshTokenRecord{
		ID:        "rt_2",
		UserID:    admin.ID,
		FamilyID:  "fam_1",
		TokenHash: "hash_rt_2",
		ExpiresAt: now.Add(48 * time.Hour),
		CreatedAt: now,
	}, now); err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}

	mcpTokenRepo := authpg.NewMCPTokenRepo(pool)
	if err := mcpTokenRepo.SaveMCPToken(ctx, authdomain.MCPApiToken{
		ID:        "mcp_1",
		UserID:    admin.ID,
		Label:     "Antigravity",
		TokenHash: "hash_mcp_1",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveMCPToken failed: %v", err)
	}
	if _, err := mcpTokenRepo.FindMCPTokenByHash(ctx, "hash_mcp_1"); err != nil {
		t.Fatalf("FindMCPTokenByHash failed: %v", err)
	}

	// 3. Portfolio repositories (LegalEntityRepo, ShareholderRepo, PropertyRepo, LotRepo with ListLotsByPropertyIDs)
	entityRepo := portfoliopg.NewLegalEntityRepo(pool)
	shareholderRepo := portfoliopg.NewShareholderRepo(pool)
	propertyRepo := portfoliopg.NewPropertyRepo(pool)
	lotRepo := portfoliopg.NewLotRepo(pool)

	entity := portfoliodomain.LegalEntity{
		ID:                "ent_pg_1",
		Name:              "SCI POSTGRES",
		EntityType:        portfoliodomain.EntityTypeSCIIS,
		SIRET:             "88123456700011",
		ShareCapitalCents: 10000_00,
		TotalShares:       100,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := entityRepo.CreateLegalEntity(ctx, entity); err != nil {
		t.Fatalf("CreateLegalEntity failed: %v", err)
	}
	if _, err := shareholderRepo.UpsertShareholder(ctx, portfoliodomain.Shareholder{
		ID:                   "sh_pg_1",
		LegalEntityID:        entity.ID,
		FullName:             "Yann Associate",
		SharesCount:          100,
		DetentionBasisPoints: 10000,
		CurrentAccountCents:  5000_00,
		CreatedAt:            now,
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("UpsertShareholder failed: %v", err)
	}

	prop := portfoliodomain.Property{
		ID:                  "prop_pg_1",
		LegalEntityID:       entity.ID,
		Label:               "Immeuble Prado",
		PropertyType:        "BUILDING",
		AccountingCategory:  portfoliodomain.AccountingCategoryFixedAsset,
		AcquisitionCents:    300000_00,
		WorkStockCents:      50000_00,
		TotalBookValueCents: 350000_00,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := propertyRepo.CreateProperty(ctx, prop); err != nil {
		t.Fatalf("CreateProperty failed: %v", err)
	}

	lot := portfoliodomain.Lot{
		ID:         "lot_pg_1",
		PropertyID: prop.ID,
		Label:      "Lot A1",
		LotType:    "APARTMENT",
		SurfaceM2:  65,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := lotRepo.CreateLot(ctx, lot); err != nil {
		t.Fatalf("CreateLot failed: %v", err)
	}
	lotsByProp, err := lotRepo.ListLotsByPropertyIDs(ctx, []string{prop.ID})
	if err != nil || len(lotsByProp[prop.ID]) != 1 {
		t.Fatalf("ListLotsByPropertyIDs failed: got %+v (err=%v)", lotsByProp, err)
	}

	// 4. Lease repositories (TenantRepo, LeaseRepo, RentCallRepo with pgx.Batch)
	tenantRepo := leasepg.NewTenantRepo(pool)
	leaseRepo := leasepg.NewLeaseRepo(pool)
	rentCallRepo := leasepg.NewRentCallRepo(pool, pool)

	tenant := leasedomain.Tenant{
		ID:        "ten_pg_1",
		FullName:  "Alice Locataire",
		Email:     "alice@example.fr",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tenantRepo.CreateTenant(ctx, tenant); err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}

	lease := leasedomain.Lease{
		ID:            "lease_pg_1",
		LegalEntityID: entity.ID,
		LotID:         lot.ID,
		TenantID:      tenant.ID,
		LeaseType:     leasedomain.LeaseTypeResidentialUnfurnished,
		RentNetCents:  800_00,
		ChargesCents:  50_00,
		DueDayOfMonth: 5,
		StartDate:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Active:        true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := leaseRepo.CreateLease(ctx, lease); err != nil {
		t.Fatalf("CreateLease failed: %v", err)
	}

	calls := []leasedomain.RentCall{
		{
			ID:                "rc_pg_1",
			LeaseID:           lease.ID,
			LegalEntityID:     entity.ID,
			TenantID:          tenant.ID,
			PeriodMonth:       "2026-01",
			RentNetCents:      800_00,
			ChargesCents:      50_00,
			TotalDueCents:     850_00,
			RemainingDueCents: 850_00,
			Status:            leasedomain.StatusPending,
			DueDate:           time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "rc_pg_2",
			LeaseID:           lease.ID,
			LegalEntityID:     entity.ID,
			TenantID:          tenant.ID,
			PeriodMonth:       "2026-02",
			RentNetCents:      800_00,
			ChargesCents:      50_00,
			TotalDueCents:     850_00,
			RemainingDueCents: 850_00,
			Status:            leasedomain.StatusPending,
			DueDate:           time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC),
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}
	if err := rentCallRepo.SaveRentCalls(ctx, calls); err != nil {
		t.Fatalf("SaveRentCalls batch failed: %v", err)
	}
	unpaidCalls, err := rentCallRepo.ListUnpaidRentCallsByLease(ctx, lease.ID)
	if err != nil || len(unpaidCalls) != 2 {
		t.Fatalf("ListUnpaidRentCallsByLease failed: got %d (err=%v)", len(unpaidCalls), err)
	}

	// 5. Payment repositories (PaymentRepo with pgx.Batch + ANY($1), CashJournalRepo)
	paymentRepo := paymentpg.NewPaymentRepo(pool, pool)
	cashJournalRepo := paymentpg.NewCashJournalRepo(pool)

	pmt := paymentdomain.Payment{
		ID:               "pmt_pg_1",
		LegalEntityID:    entity.ID,
		LeaseID:          lease.ID,
		TenantID:         tenant.ID,
		AmountCents:      850_00,
		UnallocatedCents: 0,
		Method:           paymentdomain.MethodBankTransfer,
		Label:            "Virement Janvier",
		PaymentDate:      time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
		CreatedAt:        now,
		Allocations: []paymentdomain.RentAllocation{
			{
				ID:           "alloc_pg_1",
				PaymentID:    "pmt_pg_1",
				RentCallID:   "rc_pg_1",
				PeriodMonth:  "2026-01",
				AppliedCents: 850_00,
				CreatedAt:    now,
			},
		},
	}
	if err := paymentRepo.SavePayment(ctx, pmt); err != nil {
		t.Fatalf("SavePayment batch failed: %v", err)
	}
	listedPmts, err := paymentRepo.ListPaymentsByLease(ctx, lease.ID)
	if err != nil || len(listedPmts) != 1 || len(listedPmts[0].Allocations) != 1 {
		t.Fatalf("ListPaymentsByLease ANY($1) failed: got %+v (err=%v)", listedPmts, err)
	}
	if err := cashJournalRepo.SaveCashJournalEntry(ctx, paymentdomain.CashJournalEntry{
		ID:                     "cj_pg_1",
		LegalEntityID:          entity.ID,
		PaymentID:              &pmt.ID,
		AccountCode:            "530000",
		CounterpartAccountCode: "411000",
		EntryType:              paymentdomain.CashEntryReceipt,
		Label:                  "Espèces",
		DebitCents:             100_00,
		EntryDate:              time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC),
		CreatedAt:              now,
	}); err != nil {
		t.Fatalf("SaveCashJournalEntry failed: %v", err)
	}

	// 6. Document repositories (DocumentTemplateRepo, StoredFileRepo, GLIDossierRepo)
	docTplRepo := documentpg.NewDocumentTemplateRepo(pool)
	storedFileRepo := documentpg.NewStoredFileRepo(pool)
	gliRepo := documentpg.NewGLIDossierRepo(pool)

	if err := docTplRepo.SaveTemplate(ctx, documentdomain.DocumentTemplate{
		ID:           "dtpl_pg_1",
		Name:         "Quittance",
		TemplateType: documentdomain.TemplateTypeRentReceipt,
		Format:       documentdomain.FormatHTML,
		HTMLBody:     "<p>Quittance</p>",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("SaveTemplate failed: %v", err)
	}
	if err := storedFileRepo.SaveStoredFile(ctx, documentdomain.StoredFile{
		ID:            "sf_pg_1",
		LegalEntityID: entity.ID,
		LeaseID:       lease.ID,
		Category:      "QUITTANCE",
		FileName:      "quittance.pdf",
		ContentType:   "application/pdf",
		R2Key:         "leases/lease_pg_1/quittance.pdf",
		SizeBytes:     512,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("SaveStoredFile failed: %v", err)
	}
	if err := gliRepo.SaveGLIDossier(ctx, documentdomain.GLIDossier{
		ID:               "gli_pg_1",
		LeaseID:          lease.ID,
		LegalEntityID:    entity.ID,
		TotalUnpaidCents: 850_00,
		UnpaidMonths:     []string{"2026-02"},
		ArchiveR2Key:     "gli/gli_pg_1.zip",
		IncludedFiles:    []string{"01_declaration_gli.docx"},
		CreatedAt:        now,
	}); err != nil {
		t.Fatalf("SaveGLIDossier failed: %v", err)
	}

	// 7. Inspection repositories (InspectionTemplateRepo, InspectionRepo with pgx.Batch + ANY($1), IncidentRepo)
	inspTplRepo := inspectionpg.NewInspectionTemplateRepo(pool)
	inspRepo := inspectionpg.NewInspectionRepo(pool, pool)
	incRepo := inspectionpg.NewIncidentRepo(pool)

	if err := inspTplRepo.SaveTemplate(ctx, inspectiondomain.InspectionTemplate{
		ID:              "itpl_pg_1",
		Name:            "T2 Standard",
		Rooms:           []string{"Séjour", "Cuisine"},
		DefaultElements: []string{"Sol", "Murs"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("SaveInspectionTemplate failed: %v", err)
	}
	if err := inspRepo.SaveInspection(ctx, inspectiondomain.Inspection{
		ID:             "edl_pg_1",
		LeaseID:        lease.ID,
		TemplateID:     "itpl_pg_1",
		Type:           inspectiondomain.InspectionTypeMoveIn,
		InspectionDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("SaveInspection failed: %v", err)
	}
	if err := inspRepo.UpsertInspectionItems(ctx, "edl_pg_1", []inspectiondomain.InspectionItem{
		{
			ID:           "edli_pg_1",
			InspectionID: "edl_pg_1",
			RoomName:     "Séjour",
			ElementName:  "Sol",
			Condition:    inspectiondomain.ConditionNew,
			Observation:  "Parquet neuf",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "edli_pg_2",
			InspectionID: "edl_pg_1",
			RoomName:     "Cuisine",
			ElementName:  "Murs",
			Condition:    inspectiondomain.ConditionGood,
			Observation:  "RAS",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}); err != nil {
		t.Fatalf("UpsertInspectionItems batch failed: %v", err)
	}
	listedInsps, err := inspRepo.ListInspectionsByLease(ctx, lease.ID)
	if err != nil || len(listedInsps) != 1 || len(listedInsps[0].Items) != 2 {
		t.Fatalf("ListInspectionsByLease ANY($1) failed: got %+v (err=%v)", listedInsps, err)
	}
	if err := incRepo.SaveIncident(ctx, inspectiondomain.Incident{
		ID:                 "inc_pg_1",
		PropertyID:         prop.ID,
		LotID:              lot.ID,
		LeaseID:            lease.ID,
		Title:              "Fuite",
		Status:             inspectiondomain.IncidentStatusOpen,
		EstimatedCostCents: 300_00,
		OccurredAt:         time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("SaveIncident failed: %v", err)
	}

	// 8. Accounting repositories (JournalEntryRepo, AccountBalanceRepo, AnnualAccountReportRepo)
	journalRepo := accountingpg.NewJournalEntryRepo(pool, pool)
	balanceRepo := accountingpg.NewAccountBalanceRepo(pool, pool)
	reportRepo := accountingpg.NewAnnualAccountReportRepo(pool)

	entryDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := journalRepo.SaveJournalEntry(ctx, accountingdomain.JournalEntry{
		ID:             "je_pg_1",
		LegalEntityID:  entity.ID,
		FiscalYear:     2026,
		JournalCode:    "OD",
		JournalLabel:   "Opérations Diverses",
		EntryNumber:    "2026-00001",
		EntryDate:      entryDate,
		DocumentRef:    "DOC-1",
		DocumentDate:   entryDate,
		Label:          "Loyer Janvier",
		ValidationDate: entryDate,
		CreatedAt:      now,
		Lines: []accountingdomain.JournalLine{
			{
				ID:             "jl_pg_1",
				JournalEntryID: "je_pg_1",
				LegalEntityID:  entity.ID,
				FiscalYear:     2026,
				LineID:         "ACTIF_CREANCES_CLIENTS",
				AccountCode:    "411000",
				AccountLabel:   "Clients",
				DebitCents:     money.AmountCents(850_00),
				CurrencyCode:   "EUR",
				CreatedAt:      now,
			},
			{
				ID:             "jl_pg_2",
				JournalEntryID: "je_pg_1",
				LegalEntityID:  entity.ID,
				FiscalYear:     2026,
				LineID:         "PRODUITS_LOYERS",
				AccountCode:    "706000",
				AccountLabel:   "Loyers",
				CreditCents:    money.AmountCents(850_00),
				CurrencyCode:   "EUR",
				CreatedAt:      now,
			},
		},
	}); err != nil {
		t.Fatalf("SaveJournalEntry batch failed: %v", err)
	}
	listedEntries, err := journalRepo.ListJournalEntries(ctx, entity.ID, 2026)
	if err != nil || len(listedEntries) != 1 || len(listedEntries[0].Lines) != 2 {
		t.Fatalf("ListJournalEntries ANY($1) failed: got %+v (err=%v)", listedEntries, err)
	}

	if err := balanceRepo.UpsertAccountBalanceLines(ctx, entity.ID, 2026, []accountingdomain.AccountBalanceLine{
		{
			ID:          "abl_pg_1",
			LineID:      "PRODUITS_LOYERS",
			AccountCode: "706000",
			Label:       "Loyers",
			Section:     "INCOME_STATEMENT_REVENUE",
			NetNCents:   850_00,
		},
	}); err != nil {
		t.Fatalf("UpsertAccountBalanceLines batch failed: %v", err)
	}
	balances, err := balanceRepo.ListAccountBalanceLines(ctx, entity.ID, 2026)
	if err != nil || len(balances) != 1 {
		t.Fatalf("ListAccountBalanceLines failed: got %+v (err=%v)", balances, err)
	}

	if err := reportRepo.SaveAnnualAccountReport(ctx, accountingdomain.AnnualAccountReport{
		ID:              "aar_pg_1",
		LegalEntityID:   entity.ID,
		FiscalYear:      2026,
		NetIncomeNCents: 850_00,
		PDFR2Key:        "accounting/ent_pg_1/2026_annual.pdf",
		FileName:        "2026_annual.pdf",
		SizeBytes:       1024,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("SaveAnnualAccountReport failed: %v", err)
	}
}
