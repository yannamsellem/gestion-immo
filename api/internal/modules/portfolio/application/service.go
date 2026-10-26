package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
)

// CreateLegalEntityCmd carries the input for creating a portfolio legal entity.
type CreateLegalEntityCmd struct {
	Name              string            `json:"name"`
	EntityType        string            `json:"entity_type"`
	SIRET             string            `json:"siret"`
	NAFCode           string            `json:"naf_code"`
	Address           string            `json:"address"`
	ShareCapitalCents money.AmountCents `json:"share_capital_cents"`
	TotalShares       int64             `json:"total_shares"`
}

// LegalEntityDTO is an immutable projection of a domain.LegalEntity.
type LegalEntityDTO struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	EntityType        domain.EntityType `json:"entity_type"`
	SIRET             string            `json:"siret"`
	NAFCode           string            `json:"naf_code"`
	Address           string            `json:"address"`
	ShareCapitalCents money.AmountCents `json:"share_capital_cents"`
	TotalShares       int64             `json:"total_shares"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// UpsertShareholderCmd carries the input for creating or updating an associate (CERFA 2059-F).
type UpsertShareholderCmd struct {
	LegalEntityID         string            `json:"legal_entity_id"`
	FullName              string            `json:"full_name"`
	SharesCount           int64             `json:"shares_count"`
	CurrentAccountCents   money.AmountCents `json:"current_account_cents"`
	RemunerationPaidCents money.AmountCents `json:"remuneration_paid_cents"`
}

// ShareholderDTO is an immutable projection of a domain.Shareholder.
type ShareholderDTO struct {
	ID                    string            `json:"id"`
	LegalEntityID         string            `json:"legal_entity_id"`
	FullName              string            `json:"full_name"`
	SharesCount           int64             `json:"shares_count"`
	DetentionBasisPoints  int64             `json:"detention_basis_points"`
	CurrentAccountCents   money.AmountCents `json:"current_account_cents"`
	RemunerationPaidCents money.AmountCents `json:"remuneration_paid_cents"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// CreatePropertyCmd carries the input for adding a property asset to a legal entity.
type CreatePropertyCmd struct {
	LegalEntityID      string            `json:"legal_entity_id"`
	Label              string            `json:"label"`
	PropertyType       string            `json:"property_type"`
	AccountingCategory string            `json:"accounting_category"`
	Address            string            `json:"address"`
	AcquisitionCents   money.AmountCents `json:"acquisition_cents"`
	WorkStockCents     money.AmountCents `json:"work_stock_cents"`
}

// PropertyDTO is an immutable projection of a domain.Property with its lots.
type PropertyDTO struct {
	ID                  string                    `json:"id"`
	LegalEntityID       string                    `json:"legal_entity_id"`
	Label               string                    `json:"label"`
	PropertyType        string                    `json:"property_type"`
	AccountingCategory  domain.AccountingCategory `json:"accounting_category"`
	Address             string                    `json:"address"`
	AcquisitionCents    money.AmountCents         `json:"acquisition_cents"`
	WorkStockCents      money.AmountCents         `json:"work_stock_cents"`
	TotalBookValueCents money.AmountCents         `json:"total_book_value_cents"`
	Lots                []LotDTO                  `json:"lots,omitempty"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
}

// CreateLotCmd carries the input for adding a rentable lot to a property.
type CreateLotCmd struct {
	PropertyID     string `json:"property_id"`
	Label          string `json:"label"`
	LotType        string `json:"lot_type"`
	SurfaceM2      int64  `json:"surface_m2"`
	OwnershipShare int64  `json:"ownership_share"`
	Floor          string `json:"floor"`
}

// LotDTO is an immutable projection of a domain.Lot.
type LotDTO struct {
	ID             string    `json:"id"`
	PropertyID     string    `json:"property_id"`
	Label          string    `json:"label"`
	LotType        string    `json:"lot_type"`
	SurfaceM2      int64     `json:"surface_m2"`
	OwnershipShare int64     `json:"ownership_share"`
	Floor          string    `json:"floor"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EntityPortfolioSummaryDTO aggregates a legal entity, its CERFA 2059-F shareholders,
// its properties (with lots), and total book value across Fixed Assets and Merchandise.
type EntityPortfolioSummaryDTO struct {
	Entity                LegalEntityDTO    `json:"entity"`
	Shareholders          []ShareholderDTO  `json:"shareholders"`
	Properties            []PropertyDTO     `json:"properties"`
	TotalAcquisitionCents money.AmountCents `json:"total_acquisition_cents"`
	TotalWorkStockCents   money.AmountCents `json:"total_work_stock_cents"`
	TotalBookValueCents   money.AmountCents `json:"total_book_value_cents"`
	TotalLotsCount        int               `json:"total_lots_count"`
}

// Service orchestrates legal entities, CERFA 2059-F shareholders, properties, and lots.
type Service struct {
	entityRepo      domain.LegalEntityRepository
	shareholderRepo domain.ShareholderRepository
	propertyRepo    domain.PropertyRepository
	lotRepo         domain.LotRepository
	uow             domain.UnitOfWork
}

// NewService constructs a portfolio application Service via constructor injection of per-aggregate repositories and UnitOfWork.
func NewService(
	entityRepo domain.LegalEntityRepository,
	shareholderRepo domain.ShareholderRepository,
	propertyRepo domain.PropertyRepository,
	lotRepo domain.LotRepository,
	uow domain.UnitOfWork,
) *Service {
	return &Service{
		entityRepo:      entityRepo,
		shareholderRepo: shareholderRepo,
		propertyRepo:    propertyRepo,
		lotRepo:         lotRepo,
		uow:             uow,
	}
}

// CreateLegalEntity validates and persists a new LegalEntity.
func (s *Service) CreateLegalEntity(ctx context.Context, cmd CreateLegalEntityCmd) (LegalEntityDTO, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return LegalEntityDTO{}, fmt.Errorf("%w: entity name is required", domain.ErrValidation)
	}

	entityType := domain.EntityType(strings.ToUpper(strings.TrimSpace(cmd.EntityType)))
	if !entityType.IsValid() {
		return LegalEntityDTO{}, fmt.Errorf("%w: unsupported entity_type %q", domain.ErrValidation, cmd.EntityType)
	}

	if cmd.ShareCapitalCents.IsNegative() {
		return LegalEntityDTO{}, fmt.Errorf("%w: share_capital_cents cannot be negative", domain.ErrValidation)
	}

	totalShares := cmd.TotalShares
	if totalShares < 0 {
		return LegalEntityDTO{}, fmt.Errorf("%w: total_shares cannot be negative", domain.ErrValidation)
	}
	if totalShares == 0 {
		totalShares = 100
	}

	entityID, err := id.Generate("ent")
	if err != nil {
		return LegalEntityDTO{}, err
	}

	now := time.Now().UTC()
	entity := domain.LegalEntity{
		ID:                entityID,
		Name:              name,
		EntityType:        entityType,
		SIRET:             strings.TrimSpace(cmd.SIRET),
		NAFCode:           strings.TrimSpace(cmd.NAFCode),
		Address:           strings.TrimSpace(cmd.Address),
		ShareCapitalCents: cmd.ShareCapitalCents,
		TotalShares:       totalShares,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.entityRepo.CreateLegalEntity(ctx, entity); err != nil {
		return LegalEntityDTO{}, err
	}

	return toLegalEntityDTO(entity), nil
}

// ListLegalEntities returns all portfolio legal entities ordered by creation date.
func (s *Service) ListLegalEntities(ctx context.Context) ([]LegalEntityDTO, error) {
	entities, err := s.entityRepo.ListLegalEntities(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LegalEntityDTO, 0, len(entities))
	for _, ent := range entities {
		out = append(out, toLegalEntityDTO(ent))
	}
	return out, nil
}

// GetLegalEntityByID returns a single portfolio legal entity by ID without loading shareholders, properties, or lots.
func (s *Service) GetLegalEntityByID(ctx context.Context, entityID string) (LegalEntityDTO, error) {
	trimmedID := strings.TrimSpace(entityID)
	if trimmedID == "" {
		return LegalEntityDTO{}, fmt.Errorf("%w: entity id is required", domain.ErrValidation)
	}
	ent, err := s.entityRepo.FindLegalEntityByID(ctx, trimmedID)
	if err != nil {
		return LegalEntityDTO{}, err
	}
	return toLegalEntityDTO(*ent), nil
}

// UpsertShareholder creates or updates a shareholder on a legal entity and computes
// DetentionBasisPoints using exact integer half-up rounding: (SharesCount * 10000 + TotalShares/2) / TotalShares.
func (s *Service) UpsertShareholder(ctx context.Context, cmd UpsertShareholderCmd) (ShareholderDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	fullName := strings.TrimSpace(cmd.FullName)
	if entityID == "" || fullName == "" {
		return ShareholderDTO{}, fmt.Errorf("%w: legal_entity_id and full_name are required", domain.ErrValidation)
	}

	entity, err := s.entityRepo.FindLegalEntityByID(ctx, entityID)
	if err != nil {
		return ShareholderDTO{}, err
	}

	if cmd.SharesCount < 0 || cmd.SharesCount > entity.TotalShares {
		return ShareholderDTO{}, fmt.Errorf("%w: shares_count (%d) must be between 0 and total_shares (%d)", domain.ErrValidation, cmd.SharesCount, entity.TotalShares)
	}
	if cmd.RemunerationPaidCents.IsNegative() {
		return ShareholderDTO{}, fmt.Errorf("%w: remuneration_paid_cents cannot be negative", domain.ErrValidation)
	}

	detentionBps := domain.ComputeDetentionBasisPoints(cmd.SharesCount, entity.TotalShares)

	shID, err := id.Generate("sh")
	if err != nil {
		return ShareholderDTO{}, err
	}

	now := time.Now().UTC()
	sh := domain.Shareholder{
		ID:                    shID,
		LegalEntityID:         entity.ID,
		FullName:              fullName,
		SharesCount:           cmd.SharesCount,
		DetentionBasisPoints:  detentionBps,
		CurrentAccountCents:   cmd.CurrentAccountCents,
		RemunerationPaidCents: cmd.RemunerationPaidCents,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	var saved *domain.Shareholder
	if err := s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		existingList, err := repos.Shareholders.ListShareholdersByEntity(txCtx, entity.ID)
		if err != nil {
			return err
		}
		var otherSharesSum int64
		for _, existing := range existingList {
			if existing.FullName != fullName {
				otherSharesSum += existing.SharesCount
			}
		}
		if otherSharesSum+cmd.SharesCount > entity.TotalShares {
			return fmt.Errorf("%w: total shares across shareholders (%d) would exceed entity total_shares (%d)", domain.ErrValidation, otherSharesSum+cmd.SharesCount, entity.TotalShares)
		}

		res, err := repos.Shareholders.UpsertShareholder(txCtx, sh)
		if err != nil {
			return err
		}
		saved = res
		return nil
	}); err != nil {
		return ShareholderDTO{}, err
	}

	return toShareholderDTO(*saved), nil
}

// CreateProperty validates and persists a property asset under a LegalEntity,
// computing TotalBookValueCents = AcquisitionCents + WorkStockCents.
func (s *Service) CreateProperty(ctx context.Context, cmd CreatePropertyCmd) (PropertyDTO, error) {
	entityID := strings.TrimSpace(cmd.LegalEntityID)
	label := strings.TrimSpace(cmd.Label)
	if entityID == "" || label == "" {
		return PropertyDTO{}, fmt.Errorf("%w: legal_entity_id and label are required", domain.ErrValidation)
	}

	entity, err := s.entityRepo.FindLegalEntityByID(ctx, entityID)
	if err != nil {
		return PropertyDTO{}, err
	}

	propType := strings.ToUpper(strings.TrimSpace(cmd.PropertyType))
	if propType == "" {
		propType = "BUILDING"
	}

	categoryStr := strings.ToUpper(strings.TrimSpace(cmd.AccountingCategory))
	if categoryStr == "" {
		categoryStr = string(domain.AccountingCategoryFixedAsset)
	}
	category := domain.AccountingCategory(categoryStr)
	if !category.IsValid() {
		return PropertyDTO{}, fmt.Errorf("%w: unsupported accounting_category %q", domain.ErrValidation, cmd.AccountingCategory)
	}

	if cmd.AcquisitionCents.IsNegative() || cmd.WorkStockCents.IsNegative() {
		return PropertyDTO{}, fmt.Errorf("%w: acquisition_cents and work_stock_cents cannot be negative", domain.ErrValidation)
	}

	propID, err := id.Generate("prop")
	if err != nil {
		return PropertyDTO{}, err
	}

	now := time.Now().UTC()
	prop := domain.Property{
		ID:                 propID,
		LegalEntityID:      entity.ID,
		Label:              label,
		PropertyType:       propType,
		AccountingCategory: category,
		Address:            strings.TrimSpace(cmd.Address),
		AcquisitionCents:   cmd.AcquisitionCents,
		WorkStockCents:     cmd.WorkStockCents,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	prop.TotalBookValueCents = prop.ComputeTotalBookValue()

	if err := s.propertyRepo.CreateProperty(ctx, prop); err != nil {
		return PropertyDTO{}, err
	}

	return toPropertyDTO(prop, nil), nil
}

// ListProperties returns all properties (with their lots) belonging to entityID.
func (s *Service) ListProperties(ctx context.Context, entityID string) ([]PropertyDTO, error) {
	trimmedID := strings.TrimSpace(entityID)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: legal_entity_id is required", domain.ErrValidation)
	}

	if _, err := s.entityRepo.FindLegalEntityByID(ctx, trimmedID); err != nil {
		return nil, err
	}

	props, err := s.propertyRepo.ListPropertiesByEntity(ctx, trimmedID)
	if err != nil {
		return nil, err
	}

	lotsByProp, err := s.listLotsByProperties(ctx, props)
	if err != nil {
		return nil, err
	}

	out := make([]PropertyDTO, 0, len(props))
	for _, p := range props {
		lots := lotsByProp[p.ID]
		lotDTOs := make([]LotDTO, 0, len(lots))
		for _, l := range lots {
			lotDTOs = append(lotDTOs, toLotDTO(l))
		}
		out = append(out, toPropertyDTO(p, lotDTOs))
	}
	return out, nil
}

// CreateLot validates and persists a rentable lot under a Property.
func (s *Service) CreateLot(ctx context.Context, cmd CreateLotCmd) (LotDTO, error) {
	propID := strings.TrimSpace(cmd.PropertyID)
	label := strings.TrimSpace(cmd.Label)
	if propID == "" || label == "" {
		return LotDTO{}, fmt.Errorf("%w: property_id and label are required", domain.ErrValidation)
	}

	prop, err := s.propertyRepo.FindPropertyByID(ctx, propID)
	if err != nil {
		return LotDTO{}, err
	}

	if cmd.SurfaceM2 < 0 || cmd.OwnershipShare < 0 {
		return LotDTO{}, fmt.Errorf("%w: surface_m2 and ownership_share cannot be negative", domain.ErrValidation)
	}

	lotType := strings.ToUpper(strings.TrimSpace(cmd.LotType))
	if lotType == "" {
		lotType = "APARTMENT"
	}

	lotID, err := id.Generate("lot")
	if err != nil {
		return LotDTO{}, err
	}

	now := time.Now().UTC()
	lot := domain.Lot{
		ID:             lotID,
		PropertyID:     prop.ID,
		Label:          label,
		LotType:        lotType,
		SurfaceM2:      cmd.SurfaceM2,
		OwnershipShare: cmd.OwnershipShare,
		Floor:          strings.TrimSpace(cmd.Floor),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.lotRepo.CreateLot(ctx, lot); err != nil {
		return LotDTO{}, err
	}

	return toLotDTO(lot), nil
}

// GetEntityPortfolioSummary returns the full hierarchy and valuation summary for a legal entity.
func (s *Service) GetEntityPortfolioSummary(ctx context.Context, entityID string) (EntityPortfolioSummaryDTO, error) {
	trimmedID := strings.TrimSpace(entityID)
	if trimmedID == "" {
		return EntityPortfolioSummaryDTO{}, fmt.Errorf("%w: entity id is required", domain.ErrValidation)
	}

	entity, err := s.entityRepo.FindLegalEntityByID(ctx, trimmedID)
	if err != nil {
		return EntityPortfolioSummaryDTO{}, err
	}

	shareholders, err := s.shareholderRepo.ListShareholdersByEntity(ctx, entity.ID)
	if err != nil {
		return EntityPortfolioSummaryDTO{}, err
	}
	shDTOs := make([]ShareholderDTO, 0, len(shareholders))
	for _, sh := range shareholders {
		shDTOs = append(shDTOs, toShareholderDTO(sh))
	}

	props, err := s.propertyRepo.ListPropertiesByEntity(ctx, entity.ID)
	if err != nil {
		return EntityPortfolioSummaryDTO{}, err
	}

	lotsByProp, err := s.listLotsByProperties(ctx, props)
	if err != nil {
		return EntityPortfolioSummaryDTO{}, err
	}

	var totalAcquisition money.AmountCents
	var totalWorkStock money.AmountCents
	var totalBookValue money.AmountCents
	totalLotsCount := 0

	propDTOs := make([]PropertyDTO, 0, len(props))
	for _, p := range props {
		lots := lotsByProp[p.ID]
		lotDTOs := make([]LotDTO, 0, len(lots))
		for _, l := range lots {
			lotDTOs = append(lotDTOs, toLotDTO(l))
		}
		totalLotsCount += len(lotDTOs)
		totalAcquisition = totalAcquisition.Add(p.AcquisitionCents)
		totalWorkStock = totalWorkStock.Add(p.WorkStockCents)
		totalBookValue = totalBookValue.Add(p.TotalBookValueCents)
		propDTOs = append(propDTOs, toPropertyDTO(p, lotDTOs))
	}

	return EntityPortfolioSummaryDTO{
		Entity:                toLegalEntityDTO(*entity),
		Shareholders:          shDTOs,
		Properties:            propDTOs,
		TotalAcquisitionCents: totalAcquisition,
		TotalWorkStockCents:   totalWorkStock,
		TotalBookValueCents:   totalBookValue,
		TotalLotsCount:        totalLotsCount,
	}, nil
}

func (s *Service) listLotsByProperties(ctx context.Context, props []domain.Property) (map[string][]domain.Lot, error) {
	if len(props) == 0 {
		return map[string][]domain.Lot{}, nil
	}
	propertyIDs := make([]string, len(props))
	for i := range props {
		propertyIDs[i] = props[i].ID
	}
	if batchRepo, ok := s.lotRepo.(domain.LotBatchRepository); ok {
		return batchRepo.ListLotsByPropertyIDs(ctx, propertyIDs)
	}
	grouped := make(map[string][]domain.Lot, len(props))
	for _, propID := range propertyIDs {
		lots, err := s.lotRepo.ListLotsByProperty(ctx, propID)
		if err != nil {
			return nil, err
		}
		grouped[propID] = lots
	}
	return grouped, nil
}

func toLegalEntityDTO(e domain.LegalEntity) LegalEntityDTO {
	return LegalEntityDTO{
		ID:                e.ID,
		Name:              e.Name,
		EntityType:        e.EntityType,
		SIRET:             e.SIRET,
		NAFCode:           e.NAFCode,
		Address:           e.Address,
		ShareCapitalCents: e.ShareCapitalCents,
		TotalShares:       e.TotalShares,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

func toShareholderDTO(sh domain.Shareholder) ShareholderDTO {
	return ShareholderDTO{
		ID:                    sh.ID,
		LegalEntityID:         sh.LegalEntityID,
		FullName:              sh.FullName,
		SharesCount:           sh.SharesCount,
		DetentionBasisPoints:  sh.DetentionBasisPoints,
		CurrentAccountCents:   sh.CurrentAccountCents,
		RemunerationPaidCents: sh.RemunerationPaidCents,
		CreatedAt:             sh.CreatedAt,
		UpdatedAt:             sh.UpdatedAt,
	}
}

func toPropertyDTO(p domain.Property, lots []LotDTO) PropertyDTO {
	return PropertyDTO{
		ID:                  p.ID,
		LegalEntityID:       p.LegalEntityID,
		Label:               p.Label,
		PropertyType:        p.PropertyType,
		AccountingCategory:  p.AccountingCategory,
		Address:             p.Address,
		AcquisitionCents:    p.AcquisitionCents,
		WorkStockCents:      p.WorkStockCents,
		TotalBookValueCents: p.TotalBookValueCents,
		Lots:                lots,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

func toLotDTO(l domain.Lot) LotDTO {
	return LotDTO{
		ID:             l.ID,
		PropertyID:     l.PropertyID,
		Label:          l.Label,
		LotType:        l.LotType,
		SurfaceM2:      l.SurfaceM2,
		OwnershipShare: l.OwnershipShare,
		Floor:          l.Floor,
		CreatedAt:      l.CreatedAt,
		UpdatedAt:      l.UpdatedAt,
	}
}


