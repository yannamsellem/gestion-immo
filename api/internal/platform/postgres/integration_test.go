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
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/postgres"
	"github.com/yannamsellem/gestion-immo/internal/platform/secret"
)

func TestPostgresIntegration_TransactionsAndRepositories(t *testing.T) {
	dbURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
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
	rolledBackUserID := id.MustNewV7()
	sentinelErr := errors.New("simulated rollback error")
	err = postgres.WithinTransaction(ctx, pool, func(txCtx context.Context) error {
		txUserRepo := authpg.NewUserRepo(postgres.Conn(txCtx, pool))
		if err := txUserRepo.CreateUser(txCtx, authdomain.User{
			ID:           rolledBackUserID,
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
	if _, err := userRepo.FindUserByID(ctx, rolledBackUserID); !errors.Is(err, authdomain.ErrUserNotFound) {
		t.Fatalf("expected rolled back user to not exist, got err=%v", err)
	}

	// 2. Auth repositories (UserRepo, RefreshTokenRepo, MCPTokenRepo)
	admin := authdomain.User{
		ID:           id.MustNewV7(),
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
	rt1ID := id.MustNewV7()
	rt2ID := id.MustNewV7()
	familyID := id.MustNewV7()
	if err := refreshRepo.SaveRefreshToken(ctx, authdomain.RefreshTokenRecord{
		ID:        rt1ID,
		UserID:    admin.ID,
		FamilyID:  familyID,
		TokenHash: "hash_rt_1",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SaveRefreshToken failed: %v", err)
	}
	if err := refreshRepo.RotateRefreshToken(ctx, rt1ID, authdomain.RefreshTokenRecord{
		ID:        rt2ID,
		UserID:    admin.ID,
		FamilyID:  familyID,
		TokenHash: "hash_rt_2",
		ExpiresAt: now.Add(48 * time.Hour),
		CreatedAt: now,
	}, now); err != nil {
		t.Fatalf("RotateRefreshToken failed: %v", err)
	}

	mcpTokenRepo := authpg.NewMCPTokenRepo(pool)
	if err := mcpTokenRepo.SaveMCPToken(ctx, authdomain.MCPApiToken{
		ID:        id.MustNewV7(),
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
		ID:                id.MustNewV7(),
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
		ID:                   id.MustNewV7(),
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
		ID:                  id.MustNewV7(),
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
		ID:         id.MustNewV7(),
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
		ID:        id.MustNewV7(),
		FullName:  "Alice Locataire",
		Email:     "alice@example.fr",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := tenantRepo.CreateTenant(ctx, tenant); err != nil {
		t.Fatalf("CreateTenant failed: %v", err)
	}

	lease := leasedomain.Lease{
		ID:            id.MustNewV7(),
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

	rc1ID := id.MustNewV7()
	rc2ID := id.MustNewV7()
	calls := []leasedomain.RentCall{
		{
			ID:                rc1ID,
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
			ID:                rc2ID,
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

	// 5. Payment repositories (PaymentRepo with pgx.Batch + ANY($1::uuid[]), CashJournalRepo)
	paymentRepo := paymentpg.NewPaymentRepo(pool, pool)
	cashJournalRepo := paymentpg.NewCashJournalRepo(pool)

	pmtID := id.MustNewV7()
	pmt := paymentdomain.Payment{
		ID:               pmtID,
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
				ID:           id.MustNewV7(),
				PaymentID:    pmtID,
				RentCallID:   rc1ID,
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
		t.Fatalf("ListPaymentsByLease ANY($1::uuid[]) failed: got %+v (err=%v)", listedPmts, err)
	}
	if err := cashJournalRepo.SaveCashJournalEntry(ctx, paymentdomain.CashJournalEntry{
		ID:                     id.MustNewV7(),
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

	// 6. Document repositories (DocumentTemplateRepo, StoredFileRepo, GLIDossierRepo) + nullable UUID round-trips
	docTplRepo := documentpg.NewDocumentTemplateRepo(pool)
	storedFileRepo := documentpg.NewStoredFileRepo(pool)
	gliRepo := documentpg.NewGLIDossierRepo(pool)

	docTplID := id.MustNewV7()
	if err := docTplRepo.SaveTemplate(ctx, documentdomain.DocumentTemplate{
		ID:           docTplID,
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
		ID:            id.MustNewV7(),
		LegalEntityID: entity.ID,
		LeaseID:       lease.ID,
		Category:      "QUITTANCE",
		FileName:      "quittance.pdf",
		ContentType:   "application/pdf",
		R2Key:         "leases/" + lease.ID + "/quittance.pdf",
		SizeBytes:     512,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("SaveStoredFile failed: %v", err)
	}
	// Nullable lease_id ("") <-> SQL NULL round-trip on stored_files
	entityScopedFileID := id.MustNewV7()
	if err := storedFileRepo.SaveStoredFile(ctx, documentdomain.StoredFile{
		ID:            entityScopedFileID,
		LegalEntityID: entity.ID,
		LeaseID:       "",
		Category:      "TAX",
		FileName:      "taxe_fonciere.pdf",
		ContentType:   "application/pdf",
		R2Key:         "entities/" + entity.ID + "/taxe_fonciere.pdf",
		SizeBytes:     1024,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("SaveStoredFile with empty LeaseID failed: %v", err)
	}
	loadedEntityFile, err := storedFileRepo.FindStoredFileByID(ctx, entityScopedFileID)
	if err != nil || loadedEntityFile.LeaseID != "" || loadedEntityFile.LegalEntityID != entity.ID {
		t.Fatalf("expected nullable LeaseID round-trip to \"\", got %+v (err=%v)", loadedEntityFile, err)
	}

	// Nullable template_id ("") <-> SQL NULL round-trip on gli_dossiers
	gliID := id.MustNewV7()
	if err := gliRepo.SaveGLIDossier(ctx, documentdomain.GLIDossier{
		ID:               gliID,
		LeaseID:          lease.ID,
		LegalEntityID:    entity.ID,
		TemplateID:       "",
		TotalUnpaidCents: 850_00,
		UnpaidMonths:     []string{"2026-02"},
		ArchiveR2Key:     "gli/" + gliID + ".zip",
		IncludedFiles:    []string{"01_declaration_gli.docx"},
		CreatedAt:        now,
	}); err != nil {
		t.Fatalf("SaveGLIDossier with empty TemplateID failed: %v", err)
	}
	listedGLI, err := gliRepo.ListGLIDossiersByLease(ctx, lease.ID)
	if err != nil || len(listedGLI) != 1 || listedGLI[0].TemplateID != "" {
		t.Fatalf("expected nullable TemplateID round-trip to \"\" on gli_dossiers, got %+v (err=%v)", listedGLI, err)
	}

	// 7. Inspection repositories (InspectionTemplateRepo, InspectionRepo with pgx.Batch + ANY($1::uuid[]), IncidentRepo)
	inspTplRepo := inspectionpg.NewInspectionTemplateRepo(pool)
	inspRepo := inspectionpg.NewInspectionRepo(pool, pool)
	incRepo := inspectionpg.NewIncidentRepo(pool)

	inspTplID := id.MustNewV7()
	if err := inspTplRepo.SaveTemplate(ctx, inspectiondomain.InspectionTemplate{
		ID:              inspTplID,
		Name:            "T2 Standard",
		Rooms:           []string{"Séjour", "Cuisine"},
		DefaultElements: []string{"Sol", "Murs"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("SaveInspectionTemplate failed: %v", err)
	}
	edlID := id.MustNewV7()
	if err := inspRepo.SaveInspection(ctx, inspectiondomain.Inspection{
		ID:             edlID,
		LeaseID:        lease.ID,
		TemplateID:     inspTplID,
		Type:           inspectiondomain.InspectionTypeMoveIn,
		InspectionDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("SaveInspection failed: %v", err)
	}
	if err := inspRepo.UpsertInspectionItems(ctx, edlID, []inspectiondomain.InspectionItem{
		{
			ID:           id.MustNewV7(),
			InspectionID: edlID,
			RoomName:     "Séjour",
			ElementName:  "Sol",
			Condition:    inspectiondomain.ConditionNew,
			Observation:  "Parquet neuf",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           id.MustNewV7(),
			InspectionID: edlID,
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
		t.Fatalf("ListInspectionsByLease ANY($1::uuid[]) failed: got %+v (err=%v)", listedInsps, err)
	}
	if err := incRepo.SaveIncident(ctx, inspectiondomain.Incident{
		ID:                 id.MustNewV7(),
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
	// Nullable lot_id/lease_id ("") <-> SQL NULL round-trip on incidents + ListIncidents filters
	roofIncID := id.MustNewV7()
	if err := incRepo.SaveIncident(ctx, inspectiondomain.Incident{
		ID:                 roofIncID,
		PropertyID:         prop.ID,
		LotID:              "",
		LeaseID:            "",
		Title:              "Toiture",
		Status:             inspectiondomain.IncidentStatusOpen,
		EstimatedCostCents: 1200_00,
		OccurredAt:         time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("SaveIncident with empty LotID/LeaseID failed: %v", err)
	}
	loadedRoofInc, err := incRepo.FindIncidentByID(ctx, roofIncID)
	if err != nil || loadedRoofInc.LotID != "" || loadedRoofInc.LeaseID != "" || loadedRoofInc.PropertyID != prop.ID {
		t.Fatalf("expected nullable LotID/LeaseID round-trip to \"\", got %+v (err=%v)", loadedRoofInc, err)
	}
	allIncidents, err := incRepo.ListIncidents(ctx, "", "")
	if err != nil || len(allIncidents) != 2 {
		t.Fatalf("expected 2 incidents from ListIncidents(\"\", \"\"), got %d (err=%v)", len(allIncidents), err)
	}
	propIncidents, err := incRepo.ListIncidents(ctx, prop.ID, "")
	if err != nil || len(propIncidents) != 2 {
		t.Fatalf("expected 2 incidents from ListIncidents(prop.ID, \"\"), got %d (err=%v)", len(propIncidents), err)
	}

	// 8. Accounting repositories (JournalEntryRepo, AccountBalanceRepo, AnnualAccountReportRepo)
	journalRepo := accountingpg.NewJournalEntryRepo(pool, pool)
	balanceRepo := accountingpg.NewAccountBalanceRepo(pool, pool)
	reportRepo := accountingpg.NewAnnualAccountReportRepo(pool)

	jeID := id.MustNewV7()
	entryDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if err := journalRepo.SaveJournalEntry(ctx, accountingdomain.JournalEntry{
		ID:             jeID,
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
				ID:             id.MustNewV7(),
				JournalEntryID: jeID,
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
				ID:             id.MustNewV7(),
				JournalEntryID: jeID,
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
		t.Fatalf("ListJournalEntries ANY($1::uuid[]) failed: got %+v (err=%v)", listedEntries, err)
	}

	if err := balanceRepo.UpsertAccountBalanceLines(ctx, entity.ID, 2026, []accountingdomain.AccountBalanceLine{
		{
			ID:          id.MustNewV7(),
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
		ID:              id.MustNewV7(),
		LegalEntityID:   entity.ID,
		FiscalYear:      2026,
		NetIncomeNCents: 850_00,
		PDFR2Key:        "accounting/" + entity.ID + "/2026_annual.pdf",
		FileName:        "2026_annual.pdf",
		SizeBytes:       1024,
		CreatedAt:       now,
	}); err != nil {
		t.Fatalf("SaveAnnualAccountReport failed: %v", err)
	}

	// 9. LSP malformed non-UUID guards & SQLSTATE 23503 foreign-key violation mapping
	if _, err := userRepo.FindUserByID(ctx, "not-a-uuid"); !errors.Is(err, authdomain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound on malformed UUID, got %v", err)
	}
	if _, err := entityRepo.FindLegalEntityByID(ctx, "not-a-uuid"); !errors.Is(err, portfoliodomain.ErrLegalEntityNotFound) {
		t.Fatalf("expected ErrLegalEntityNotFound on malformed UUID, got %v", err)
	}
	if lotsMap, err := lotRepo.ListLotsByPropertyIDs(ctx, []string{"not-a-uuid"}); err != nil || len(lotsMap) != 0 {
		t.Fatalf("expected empty map without error on malformed UUID in ListLotsByPropertyIDs, got %+v (err=%v)", lotsMap, err)
	}
	if incs, err := incRepo.ListIncidents(ctx, "not-a-uuid", ""); err != nil || len(incs) != 0 {
		t.Fatalf("expected empty slice without error on malformed UUID in ListIncidents, got %+v (err=%v)", incs, err)
	}
	if err := inspRepo.SaveInspection(ctx, inspectiondomain.Inspection{
		ID:             id.MustNewV7(),
		LeaseID:        id.MustNewV7(), // valid UUIDv7, non-existent lease -> SQLSTATE 23503
		TemplateID:     inspTplID,
		Type:           inspectiondomain.InspectionTypeMoveIn,
		InspectionDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); !errors.Is(err, inspectiondomain.ErrValidation) {
		t.Fatalf("expected inspectiondomain.ErrValidation on SQLSTATE 23503 foreign-key violation, got %v", err)
	}
}
