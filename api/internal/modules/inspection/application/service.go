package application

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/inspection/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// CreateInspectionTemplateCmd carries the input for creating a reusable room/element inspection template.
type CreateInspectionTemplateCmd struct {
	Name            string   `json:"name"`
	Rooms           []string `json:"rooms"`
	DefaultElements []string `json:"default_elements,omitempty"`
}

// InspectionTemplateDTO is an immutable projection of a domain.InspectionTemplate.
type InspectionTemplateDTO struct {
	ID              string    `json:"id" format:"uuid"`
	Name            string    `json:"name"`
	Rooms           []string  `json:"rooms"`
	DefaultElements []string  `json:"default_elements"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateInspectionCmd carries the input for starting a Move-In (ENTRANT) or Move-Out (SORTANT) inspection.
type CreateInspectionCmd struct {
	LeaseID                  string               `json:"lease_id" format:"uuid"`
	TemplateID               string               `json:"template_id,omitempty" format:"uuid"`
	Type                     string               `json:"type"`
	EntryInspectionID        *string              `json:"entry_inspection_id,omitempty"`
	InspectionDate           string               `json:"inspection_date,omitempty"`
	MeterReadings            domain.MeterReadings `json:"meter_readings,omitempty"`
	KeysHandedOverCount      int                  `json:"keys_handed_over_count,omitempty"`
	KeysDescription          string               `json:"keys_description,omitempty"`
	EstimatedRepairCostCents money.AmountCents    `json:"estimated_repair_cost_cents,omitempty"`
}

// InspectionItemInput carries a single room element's condition rating, observation, and R2 photo keys.
type InspectionItemInput struct {
	RoomName    string   `json:"room_name"`
	ElementName string   `json:"element_name"`
	Condition   string   `json:"condition"`
	Observation string   `json:"observation,omitempty"`
	PhotoR2Keys []string `json:"photo_r2_keys,omitempty"`
}

// InspectionItemDTO is an immutable projection of a domain.InspectionItem.
type InspectionItemDTO struct {
	ID           string                `json:"id" format:"uuid"`
	InspectionID string                `json:"inspection_id" format:"uuid"`
	RoomName     string                `json:"room_name"`
	ElementName  string                `json:"element_name"`
	Condition    domain.ConditionState `json:"condition"`
	Observation  string                `json:"observation,omitempty"`
	PhotoR2Keys  []string              `json:"photo_r2_keys"`
}

// InspectionDTO is an immutable projection of a domain.Inspection with its items.
type InspectionDTO struct {
	ID                       string                `json:"id" format:"uuid"`
	LeaseID                  string                `json:"lease_id" format:"uuid"`
	TemplateID               string                `json:"template_id" format:"uuid"`
	Type                     domain.InspectionType `json:"type"`
	EntryInspectionID        *string               `json:"entry_inspection_id,omitempty" format:"uuid"`
	Items                    []InspectionItemDTO   `json:"items"`
	MeterReadings            domain.MeterReadings  `json:"meter_readings,omitempty"`
	KeysHandedOverCount      int                   `json:"keys_handed_over_count,omitempty"`
	KeysDescription          string                `json:"keys_description,omitempty"`
	EstimatedRepairCostCents money.AmountCents     `json:"estimated_repair_cost_cents,omitempty"`
	ReportPDFR2Key           string                `json:"report_pdf_r2_key,omitempty"`
	ReportDownloadURL        string                `json:"report_download_url,omitempty"`
	InspectionDate           string                `json:"inspection_date"`
	CreatedAt                time.Time             `json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
}

// ElementComparisonDTO represents the comparison of a single room element between entry and exit inspections.
type ElementComparisonDTO struct {
	RoomName         string                `json:"room_name"`
	ElementName      string                `json:"element_name"`
	EntryCondition   domain.ConditionState `json:"entry_condition"`
	ExitCondition    domain.ConditionState `json:"exit_condition"`
	EntryObservation string                `json:"entry_observation,omitempty"`
	ExitObservation  string                `json:"exit_observation,omitempty"`
	Degraded         bool                  `json:"degraded"`
}

// InspectionReportDTO represents a generated PDF inspection report (standalone or with entry vs. exit comparison).
type InspectionReportDTO struct {
	InspectionID       string                 `json:"inspection_id" format:"uuid"`
	LeaseID            string                 `json:"lease_id" format:"uuid"`
	Type               domain.InspectionType  `json:"type"`
	EntryInspectionID  *string                `json:"entry_inspection_id,omitempty" format:"uuid"`
	HasEntryComparison bool                   `json:"has_entry_comparison"`
	Comparisons        []ElementComparisonDTO `json:"comparisons,omitempty"`
	Degradations       []ElementComparisonDTO `json:"degradations,omitempty"`
	Items              []InspectionItemDTO    `json:"items"`
	PDFR2Key           string                 `json:"pdf_r2_key"`
	DownloadURL        string                 `json:"download_url"`
	GeneratedAt        time.Time              `json:"generated_at"`
}

// CreateIncidentCmd carries the input for opening a property/lot/lease incident or insurance claim.
type CreateIncidentCmd struct {
	PropertyID           string            `json:"property_id,omitempty"`
	LotID                string            `json:"lot_id,omitempty"`
	LeaseID              string            `json:"lease_id,omitempty"`
	Title                string            `json:"title"`
	Status               string            `json:"status,omitempty"`
	OccurredAt           string            `json:"occurred_at,omitempty"`
	InsuranceClaimNumber string            `json:"insurance_claim_number,omitempty"`
	EstimatedCostCents   money.AmountCents `json:"estimated_cost_cents,omitempty"`
	Observations         []string          `json:"observations,omitempty"`
	PhotoR2Keys          []string          `json:"photo_r2_keys,omitempty"`
}

// UpdateIncidentCmd carries the input for updating an incident's status, claim number, cost, observations, or photos.
type UpdateIncidentCmd struct {
	IncidentID           string             `json:"incident_id,omitempty"`
	Status               string             `json:"status,omitempty"`
	InsuranceClaimNumber *string            `json:"insurance_claim_number,omitempty"`
	EstimatedCostCents   *money.AmountCents `json:"estimated_cost_cents,omitempty"`
	AddObservation       string             `json:"add_observation,omitempty"`
	Observations         []string           `json:"observations,omitempty"`
	AddPhotoR2Keys       []string           `json:"add_photo_r2_keys,omitempty"`
	PhotoR2Keys          []string           `json:"photo_r2_keys,omitempty"`
}

// IncidentDTO is an immutable projection of a domain.Incident.
type IncidentDTO struct {
	ID                   string                `json:"id" format:"uuid"`
	PropertyID           string                `json:"property_id,omitempty"`
	LotID                string                `json:"lot_id,omitempty"`
	LeaseID              string                `json:"lease_id,omitempty"`
	Title                string                `json:"title"`
	Status               domain.IncidentStatus `json:"status"`
	OccurredAt           string                `json:"occurred_at"`
	InsuranceClaimNumber string                `json:"insurance_claim_number,omitempty"`
	EstimatedCostCents   money.AmountCents     `json:"estimated_cost_cents"`
	Observations         []string              `json:"observations"`
	PhotoR2Keys          []string              `json:"photo_r2_keys"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

// Service orchestrates inspection templates, move-in/move-out inspections, entry/exit comparisons, and incidents.
type Service struct {
	templates   domain.InspectionTemplateRepository
	inspections domain.InspectionRepository
	incidents   domain.IncidentRepository
	uow         domain.UnitOfWork
	storage     r2.ObjectStorage
	pdfEngine   templating.PdfEngine
}

// NewService constructs an inspection application Service via constructor injection.
func NewService(
	templates domain.InspectionTemplateRepository,
	inspections domain.InspectionRepository,
	incidents domain.IncidentRepository,
	uow domain.UnitOfWork,
	storage r2.ObjectStorage,
	pdfEngine templating.PdfEngine,
) *Service {
	return &Service{
		templates:   templates,
		inspections: inspections,
		incidents:   incidents,
		uow:         uow,
		storage:     storage,
		pdfEngine:   pdfEngine,
	}
}

// CreateTemplate validates and persists a reusable room/element inspection template.
func (s *Service) CreateTemplate(ctx context.Context, cmd CreateInspectionTemplateCmd) (InspectionTemplateDTO, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return InspectionTemplateDTO{}, fmt.Errorf("%w: template name is required", domain.ErrValidation)
	}

	rooms := cleanStringSlice(cmd.Rooms)
	if len(rooms) == 0 {
		return InspectionTemplateDTO{}, fmt.Errorf("%w: at least one room is required", domain.ErrValidation)
	}

	elements := cleanStringSlice(cmd.DefaultElements)
	if len(elements) == 0 {
		elements = append([]string(nil), domain.DefaultRoomElements...)
	}

	tplID, err := id.NewV7()
	if err != nil {
		return InspectionTemplateDTO{}, err
	}

	now := time.Now().UTC()
	tpl := domain.InspectionTemplate{
		ID:              tplID,
		Name:            name,
		Rooms:           rooms,
		DefaultElements: elements,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.templates.SaveTemplate(ctx, tpl); err != nil {
		return InspectionTemplateDTO{}, err
	}

	return toTemplateDTO(tpl), nil
}

// GetTemplate retrieves an inspection template by ID.
func (s *Service) GetTemplate(ctx context.Context, tplID string) (InspectionTemplateDTO, error) {
	trimmed := strings.TrimSpace(tplID)
	if trimmed == "" {
		return InspectionTemplateDTO{}, fmt.Errorf("%w: template_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmed) {
		return InspectionTemplateDTO{}, domain.ErrTemplateNotFound
	}
	tpl, err := s.templates.FindTemplateByID(ctx, trimmed)
	if err != nil {
		return InspectionTemplateDTO{}, err
	}
	return toTemplateDTO(*tpl), nil
}

// ListTemplates returns all registered inspection templates.
func (s *Service) ListTemplates(ctx context.Context) ([]InspectionTemplateDTO, error) {
	list, err := s.templates.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]InspectionTemplateDTO, 0, len(list))
	for _, tpl := range list {
		out = append(out, toTemplateDTO(tpl))
	}
	return out, nil
}

// CreateInspection initializes a Move-In (MOVE_IN) or Move-Out (MOVE_OUT) inspection.
// EntryInspectionID is optional: when nil on a MOVE_OUT inspection (e.g. legacy lease with a scanned PDF entry report),
// the exit inspection is created as a standalone structured inspection without error.
func (s *Service) CreateInspection(ctx context.Context, cmd CreateInspectionCmd) (InspectionDTO, error) {
	leaseID := strings.TrimSpace(cmd.LeaseID)
	if leaseID == "" {
		return InspectionDTO{}, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(leaseID) {
		return InspectionDTO{}, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}

	templateID := strings.TrimSpace(cmd.TemplateID)
	if templateID == "" {
		return InspectionDTO{}, fmt.Errorf("%w: template_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(templateID) {
		return InspectionDTO{}, fmt.Errorf("%w: template_id must be a valid UUID", domain.ErrValidation)
	}
	if _, err := s.templates.FindTemplateByID(ctx, templateID); err != nil {
		return InspectionDTO{}, err
	}

	inspType := domain.InspectionType(strings.ToUpper(strings.TrimSpace(cmd.Type)))
	if !inspType.IsValid() {
		return InspectionDTO{}, fmt.Errorf("%w: invalid inspection type %q (expected MOVE_IN or MOVE_OUT)", domain.ErrValidation, cmd.Type)
	}

	var entryIDPtr *string
	if cmd.EntryInspectionID != nil {
		trimmedEntry := strings.TrimSpace(*cmd.EntryInspectionID)
		if trimmedEntry != "" {
			if !id.IsValidUUID(trimmedEntry) {
				return InspectionDTO{}, fmt.Errorf("%w: entry_inspection_id must be a valid UUID", domain.ErrValidation)
			}
			if inspType != domain.InspectionTypeMoveOut {
				return InspectionDTO{}, fmt.Errorf("%w: entry_inspection_id is only allowed on MOVE_OUT inspections", domain.ErrValidation)
			}
			entryInsp, err := s.inspections.FindInspectionByID(ctx, trimmedEntry)
			if err != nil {
				return InspectionDTO{}, err
			}
			if entryInsp.Type != domain.InspectionTypeMoveIn {
				return InspectionDTO{}, fmt.Errorf("%w: referenced entry_inspection_id %q is not a MOVE_IN inspection", domain.ErrValidation, trimmedEntry)
			}
			if entryInsp.LeaseID != leaseID {
				return InspectionDTO{}, fmt.Errorf("%w: referenced entry_inspection_id %q belongs to lease %q, not %q", domain.ErrValidation, trimmedEntry, entryInsp.LeaseID, leaseID)
			}
			entryIDPtr = &trimmedEntry
		}
	}

	inspDate, err := parseDateOrToday(cmd.InspectionDate)
	if err != nil {
		return InspectionDTO{}, fmt.Errorf("%w: invalid inspection_date %q: %v", domain.ErrValidation, cmd.InspectionDate, err)
	}

	inspID, err := id.NewV7()
	if err != nil {
		return InspectionDTO{}, err
	}

	now := time.Now().UTC()
	insp := domain.Inspection{
		ID:                       inspID,
		LeaseID:                  leaseID,
		TemplateID:               templateID,
		Type:                     inspType,
		EntryInspectionID:        entryIDPtr,
		Items:                    []domain.InspectionItem{},
		MeterReadings:            cmd.MeterReadings,
		KeysHandedOverCount:      cmd.KeysHandedOverCount,
		KeysDescription:          strings.TrimSpace(cmd.KeysDescription),
		EstimatedRepairCostCents: cmd.EstimatedRepairCostCents,
		InspectionDate:           inspDate,
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	if err := s.inspections.SaveInspection(ctx, insp); err != nil {
		return InspectionDTO{}, err
	}

	return s.toInspectionDTO(ctx, insp), nil
}

// GetInspection retrieves an inspection by ID along with its items.
func (s *Service) GetInspection(ctx context.Context, inspID string) (InspectionDTO, error) {
	trimmed := strings.TrimSpace(inspID)
	if trimmed == "" {
		return InspectionDTO{}, fmt.Errorf("%w: inspection_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmed) {
		return InspectionDTO{}, domain.ErrInspectionNotFound
	}
	insp, err := s.inspections.FindInspectionByID(ctx, trimmed)
	if err != nil {
		return InspectionDTO{}, err
	}
	return s.toInspectionDTO(ctx, *insp), nil
}

// ListInspectionsByLease returns all inspections recorded for a lease.
func (s *Service) ListInspectionsByLease(ctx context.Context, leaseID string) ([]InspectionDTO, error) {
	trimmed := strings.TrimSpace(leaseID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmed) {
		return nil, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}
	list, err := s.inspections.ListInspectionsByLease(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]InspectionDTO, 0, len(list))
	for _, insp := range list {
		out = append(out, s.toInspectionDTO(ctx, insp))
	}
	return out, nil
}

// UpsertInspectionItems populates or updates room element conditions, observations, and R2 photo keys for an inspection.
func (s *Service) UpsertInspectionItems(ctx context.Context, inspectionID string, items []InspectionItemInput) (InspectionDTO, error) {
	trimmedID := strings.TrimSpace(inspectionID)
	if trimmedID == "" {
		return InspectionDTO{}, fmt.Errorf("%w: inspection_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmedID) {
		return InspectionDTO{}, fmt.Errorf("%w: inspection_id must be a valid UUID", domain.ErrValidation)
	}

	insp, err := s.inspections.FindInspectionByID(ctx, trimmedID)
	if err != nil {
		return InspectionDTO{}, err
	}

	existingByKey := make(map[string]domain.InspectionItem, len(insp.Items))
	for _, it := range insp.Items {
		existingByKey[itemCompositeKey(it.RoomName, it.ElementName)] = it
	}

	now := time.Now().UTC()
	domainItems := make([]domain.InspectionItem, 0, len(items))
	for _, in := range items {
		room := strings.TrimSpace(in.RoomName)
		elem := strings.TrimSpace(in.ElementName)
		if room == "" || elem == "" {
			return InspectionDTO{}, fmt.Errorf("%w: room_name and element_name are required", domain.ErrValidation)
		}

		cond := domain.ConditionState(strings.ToUpper(strings.TrimSpace(in.Condition)))
		if !cond.IsValid() {
			return InspectionDTO{}, fmt.Errorf("%w: invalid condition %q", domain.ErrValidation, in.Condition)
		}

		key := itemCompositeKey(room, elem)
		var itemID string
		createdAt := now
		if prev, ok := existingByKey[key]; ok {
			itemID = prev.ID
			createdAt = prev.CreatedAt
		} else {
			itemID, err = id.NewV7()
			if err != nil {
				return InspectionDTO{}, err
			}
		}

		domainItems = append(domainItems, domain.InspectionItem{
			ID:           itemID,
			InspectionID: insp.ID,
			RoomName:     room,
			ElementName:  elem,
			Condition:    cond,
			Observation:  strings.TrimSpace(in.Observation),
			PhotoR2Keys:  cleanStringSlice(in.PhotoR2Keys),
			CreatedAt:    createdAt,
			UpdatedAt:    now,
		})
	}

	if err := s.inspections.UpsertInspectionItems(ctx, insp.ID, domainItems); err != nil {
		return InspectionDTO{}, err
	}

	indexByKey := make(map[string]int, len(insp.Items))
	for i, existing := range insp.Items {
		indexByKey[itemCompositeKey(existing.RoomName, existing.ElementName)] = i
	}
	for _, item := range domainItems {
		key := itemCompositeKey(item.RoomName, item.ElementName)
		if idx, exists := indexByKey[key]; exists {
			insp.Items[idx] = item
		} else {
			indexByKey[key] = len(insp.Items)
			insp.Items = append(insp.Items, item)
		}
	}
	insp.UpdatedAt = now

	return s.toInspectionDTO(ctx, *insp), nil
}

// GenerateInspectionReport renders a PDF report for the inspection, uploads it to Cloudflare R2,
// and performs an automatic element-by-element comparison when EntryInspectionID != nil on a SORTANT inspection.
// When EntryInspectionID == nil on a SORTANT inspection, it generates a standalone exit report with HasEntryComparison=false.
func (s *Service) GenerateInspectionReport(ctx context.Context, inspectionID string) (InspectionReportDTO, error) {
	trimmedID := strings.TrimSpace(inspectionID)
	if trimmedID == "" {
		return InspectionReportDTO{}, fmt.Errorf("%w: inspection_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmedID) {
		return InspectionReportDTO{}, fmt.Errorf("%w: inspection_id must be a valid UUID", domain.ErrValidation)
	}

	insp, err := s.inspections.FindInspectionByID(ctx, trimmedID)
	if err != nil {
		return InspectionReportDTO{}, err
	}

	tpl, err := s.templates.FindTemplateByID(ctx, insp.TemplateID)
	if err != nil {
		return InspectionReportDTO{}, err
	}

	var (
		hasEntryComparison bool
		comparisons        []ElementComparisonDTO
		degradations       []ElementComparisonDTO
	)

	if insp.Type == domain.InspectionTypeMoveOut && insp.EntryInspectionID != nil && strings.TrimSpace(*insp.EntryInspectionID) != "" {
		entryInsp, entryErr := s.inspections.FindInspectionByID(ctx, strings.TrimSpace(*insp.EntryInspectionID))
		if entryErr != nil {
			return InspectionReportDTO{}, entryErr
		}
		hasEntryComparison = true

		entryByKey := make(map[string]domain.InspectionItem, len(entryInsp.Items))
		for _, it := range entryInsp.Items {
			entryByKey[itemCompositeKey(it.RoomName, it.ElementName)] = it
		}

		for _, exitItem := range insp.Items {
			entryItem, ok := entryByKey[itemCompositeKey(exitItem.RoomName, exitItem.ElementName)]
			if !ok {
				continue
			}
			degraded := domain.IsDegraded(entryItem.Condition, exitItem.Condition)
			comp := ElementComparisonDTO{
				RoomName:         exitItem.RoomName,
				ElementName:      exitItem.ElementName,
				EntryCondition:   entryItem.Condition,
				ExitCondition:    exitItem.Condition,
				EntryObservation: entryItem.Observation,
				ExitObservation:  exitItem.Observation,
				Degraded:         degraded,
			}
			comparisons = append(comparisons, comp)
			if degraded {
				degradations = append(degradations, comp)
			}
		}
	}

	htmlReport := buildInspectionReportHTML(*insp, *tpl, hasEntryComparison, comparisons, degradations)
	pdfBytes, err := s.pdfEngine.RenderHTMLToPDF(ctx, htmlReport)
	if err != nil {
		return InspectionReportDTO{}, fmt.Errorf("inspection: render report pdf: %w", err)
	}

	pdfR2Key := fmt.Sprintf("inspections/%s/%s_edl_%s.pdf", insp.LeaseID, insp.ID, strings.ToLower(string(insp.Type)))
	if err := s.storage.Upload(ctx, pdfR2Key, "application/pdf", bytes.NewReader(pdfBytes)); err != nil {
		return InspectionReportDTO{}, err
	}

	now := time.Now().UTC()
	insp.ReportPDFR2Key = pdfR2Key
	insp.UpdatedAt = now
	if err := s.inspections.SaveInspection(ctx, *insp); err != nil {
		return InspectionReportDTO{}, err
	}

	downloadURL, _ := s.storage.PresignGetURL(ctx, pdfR2Key, 24*time.Hour)

	return InspectionReportDTO{
		InspectionID:       insp.ID,
		LeaseID:            insp.LeaseID,
		Type:               insp.Type,
		EntryInspectionID:  copyStringPtr(insp.EntryInspectionID),
		HasEntryComparison: hasEntryComparison,
		Comparisons:        comparisons,
		Degradations:       degradations,
		Items:              toItemDTOs(insp.Items),
		PDFR2Key:           pdfR2Key,
		DownloadURL:        downloadURL,
		GeneratedAt:        now,
	}, nil
}

// CreateIncident validates and records a new property/lot/lease incident or insurance claim.
func (s *Service) CreateIncident(ctx context.Context, cmd CreateIncidentCmd) (IncidentDTO, error) {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return IncidentDTO{}, fmt.Errorf("%w: incident title is required", domain.ErrValidation)
	}
	if cmd.EstimatedCostCents < 0 {
		return IncidentDTO{}, fmt.Errorf("%w: estimated_cost_cents cannot be negative", domain.ErrValidation)
	}

	propID := strings.TrimSpace(cmd.PropertyID)
	if propID != "" && !id.IsValidUUID(propID) {
		return IncidentDTO{}, fmt.Errorf("%w: property_id must be a valid UUID", domain.ErrValidation)
	}
	lotID := strings.TrimSpace(cmd.LotID)
	if lotID != "" && !id.IsValidUUID(lotID) {
		return IncidentDTO{}, fmt.Errorf("%w: lot_id must be a valid UUID", domain.ErrValidation)
	}
	leaseID := strings.TrimSpace(cmd.LeaseID)
	if leaseID != "" && !id.IsValidUUID(leaseID) {
		return IncidentDTO{}, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}

	statusStr := strings.ToUpper(strings.TrimSpace(cmd.Status))
	if statusStr == "" {
		statusStr = string(domain.IncidentStatusOpen)
	}
	status := domain.IncidentStatus(statusStr)
	if !status.IsValid() {
		return IncidentDTO{}, fmt.Errorf("%w: invalid incident status %q", domain.ErrValidation, cmd.Status)
	}

	occurredAt, err := parseDateOrToday(cmd.OccurredAt)
	if err != nil {
		return IncidentDTO{}, fmt.Errorf("%w: invalid occurred_at %q: %v", domain.ErrValidation, cmd.OccurredAt, err)
	}

	incID, err := id.NewV7()
	if err != nil {
		return IncidentDTO{}, err
	}

	now := time.Now().UTC()
	inc := domain.Incident{
		ID:                   incID,
		PropertyID:           propID,
		LotID:                lotID,
		LeaseID:              leaseID,
		Title:                title,
		Status:               status,
		InsuranceClaimNumber: strings.TrimSpace(cmd.InsuranceClaimNumber),
		EstimatedCostCents:   cmd.EstimatedCostCents,
		Observations:         cleanStringSlice(cmd.Observations),
		PhotoR2Keys:          cleanStringSlice(cmd.PhotoR2Keys),
		OccurredAt:           occurredAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.incidents.SaveIncident(ctx, inc); err != nil {
		return IncidentDTO{}, err
	}

	return toIncidentDTO(inc), nil
}

// UpdateIncident updates an existing incident's status, insurance claim number, estimated cost, observations, or photos.
func (s *Service) UpdateIncident(ctx context.Context, cmd UpdateIncidentCmd) (IncidentDTO, error) {
	trimmedID := strings.TrimSpace(cmd.IncidentID)
	if trimmedID == "" {
		return IncidentDTO{}, fmt.Errorf("%w: incident_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmedID) {
		return IncidentDTO{}, fmt.Errorf("%w: incident_id must be a valid UUID", domain.ErrValidation)
	}

	inc, err := s.incidents.FindIncidentByID(ctx, trimmedID)
	if err != nil {
		return IncidentDTO{}, err
	}

	if strings.TrimSpace(cmd.Status) != "" {
		nextStatus := domain.IncidentStatus(strings.ToUpper(strings.TrimSpace(cmd.Status)))
		if !nextStatus.IsValid() {
			return IncidentDTO{}, fmt.Errorf("%w: invalid incident status %q", domain.ErrValidation, cmd.Status)
		}
		inc.Status = nextStatus
	}

	if cmd.InsuranceClaimNumber != nil {
		inc.InsuranceClaimNumber = strings.TrimSpace(*cmd.InsuranceClaimNumber)
	}

	if cmd.EstimatedCostCents != nil {
		if *cmd.EstimatedCostCents < 0 {
			return IncidentDTO{}, fmt.Errorf("%w: estimated_cost_cents cannot be negative", domain.ErrValidation)
		}
		inc.EstimatedCostCents = *cmd.EstimatedCostCents
	}

	if cmd.Observations != nil {
		inc.Observations = cleanStringSlice(cmd.Observations)
	}
	if strings.TrimSpace(cmd.AddObservation) != "" {
		inc.Observations = append(inc.Observations, strings.TrimSpace(cmd.AddObservation))
	}

	if cmd.PhotoR2Keys != nil {
		inc.PhotoR2Keys = cleanStringSlice(cmd.PhotoR2Keys)
	}
	if len(cmd.AddPhotoR2Keys) > 0 {
		inc.PhotoR2Keys = append(inc.PhotoR2Keys, cleanStringSlice(cmd.AddPhotoR2Keys)...)
	}

	inc.UpdatedAt = time.Now().UTC()
	if err := s.incidents.SaveIncident(ctx, *inc); err != nil {
		return IncidentDTO{}, err
	}

	return toIncidentDTO(*inc), nil
}

// GetIncident retrieves an incident by ID.
func (s *Service) GetIncident(ctx context.Context, incID string) (IncidentDTO, error) {
	trimmed := strings.TrimSpace(incID)
	if trimmed == "" {
		return IncidentDTO{}, fmt.Errorf("%w: incident_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmed) {
		return IncidentDTO{}, domain.ErrIncidentNotFound
	}
	inc, err := s.incidents.FindIncidentByID(ctx, trimmed)
	if err != nil {
		return IncidentDTO{}, err
	}
	return toIncidentDTO(*inc), nil
}

// ListIncidents lists incidents filtered optionally by property_id and/or lease_id.
func (s *Service) ListIncidents(ctx context.Context, propertyID, leaseID string) ([]IncidentDTO, error) {
	trimmedProp := strings.TrimSpace(propertyID)
	if trimmedProp != "" && !id.IsValidUUID(trimmedProp) {
		return nil, fmt.Errorf("%w: property_id must be a valid UUID", domain.ErrValidation)
	}
	trimmedLease := strings.TrimSpace(leaseID)
	if trimmedLease != "" && !id.IsValidUUID(trimmedLease) {
		return nil, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}
	list, err := s.incidents.ListIncidents(ctx, trimmedProp, trimmedLease)
	if err != nil {
		return nil, err
	}
	out := make([]IncidentDTO, 0, len(list))
	for _, inc := range list {
		out = append(out, toIncidentDTO(inc))
	}
	return out, nil
}

func (s *Service) toInspectionDTO(ctx context.Context, insp domain.Inspection) InspectionDTO {
	var downloadURL string
	if insp.ReportPDFR2Key != "" && s.storage != nil {
		downloadURL, _ = s.storage.PresignGetURL(ctx, insp.ReportPDFR2Key, 24*time.Hour)
	}
	return InspectionDTO{
		ID:                       insp.ID,
		LeaseID:                  insp.LeaseID,
		TemplateID:               insp.TemplateID,
		Type:                     insp.Type,
		EntryInspectionID:        copyStringPtr(insp.EntryInspectionID),
		Items:                    toItemDTOs(insp.Items),
		MeterReadings:            insp.MeterReadings,
		KeysHandedOverCount:      insp.KeysHandedOverCount,
		KeysDescription:          insp.KeysDescription,
		EstimatedRepairCostCents: insp.EstimatedRepairCostCents,
		ReportPDFR2Key:           insp.ReportPDFR2Key,
		ReportDownloadURL:        downloadURL,
		InspectionDate:           insp.InspectionDate.Format("2006-01-02"),
		CreatedAt:                insp.CreatedAt,
		UpdatedAt:                insp.UpdatedAt,
	}
}

func toTemplateDTO(tpl domain.InspectionTemplate) InspectionTemplateDTO {
	return InspectionTemplateDTO{
		ID:              tpl.ID,
		Name:            tpl.Name,
		Rooms:           append([]string(nil), tpl.Rooms...),
		DefaultElements: append([]string(nil), tpl.DefaultElements...),
		CreatedAt:       tpl.CreatedAt,
		UpdatedAt:       tpl.UpdatedAt,
	}
}

func toItemDTOs(items []domain.InspectionItem) []InspectionItemDTO {
	out := make([]InspectionItemDTO, 0, len(items))
	for _, it := range items {
		photos := append([]string(nil), it.PhotoR2Keys...)
		if photos == nil {
			photos = []string{}
		}
		out = append(out, InspectionItemDTO{
			ID:           it.ID,
			InspectionID: it.InspectionID,
			RoomName:     it.RoomName,
			ElementName:  it.ElementName,
			Condition:    it.Condition,
			Observation:  it.Observation,
			PhotoR2Keys:  photos,
		})
	}
	return out
}

func toIncidentDTO(inc domain.Incident) IncidentDTO {
	obs := append([]string(nil), inc.Observations...)
	if obs == nil {
		obs = []string{}
	}
	photos := append([]string(nil), inc.PhotoR2Keys...)
	if photos == nil {
		photos = []string{}
	}
	return IncidentDTO{
		ID:                   inc.ID,
		PropertyID:           inc.PropertyID,
		LotID:                inc.LotID,
		LeaseID:              inc.LeaseID,
		Title:                inc.Title,
		Status:               inc.Status,
		OccurredAt:           inc.OccurredAt.Format("2006-01-02"),
		InsuranceClaimNumber: inc.InsuranceClaimNumber,
		EstimatedCostCents:   inc.EstimatedCostCents,
		Observations:         obs,
		PhotoR2Keys:          photos,
		CreatedAt:            inc.CreatedAt,
		UpdatedAt:            inc.UpdatedAt,
	}
}

func translateInspectionTypeFr(t domain.InspectionType) string {
	switch t {
	case domain.InspectionTypeMoveIn:
		return "Entrant"
	case domain.InspectionTypeMoveOut:
		return "Sortant"
	default:
		return string(t)
	}
}

func translateConditionFr(c domain.ConditionState) string {
	switch c {
	case domain.ConditionNew:
		return "Neuf"
	case domain.ConditionGood:
		return "Bon état"
	case domain.ConditionFairWear:
		return "État d'usage"
	case domain.ConditionPoor:
		return "Mauvais état"
	case domain.ConditionOutOfService:
		return "Hors service"
	default:
		return string(c)
	}
}

func buildInspectionReportHTML(
	insp domain.Inspection,
	tpl domain.InspectionTemplate,
	hasComparison bool,
	comparisons []ElementComparisonDTO,
	degradations []ElementComparisonDTO,
) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html><html lang=\"fr\"><head><meta charset=\"UTF-8\"><title>État des Lieux</title></head><body>")
	sb.WriteString(fmt.Sprintf("<h1>État des Lieux %s — %s</h1>",
		html.EscapeString(translateInspectionTypeFr(insp.Type)),
		html.EscapeString(tpl.Name),
	))
	sb.WriteString(fmt.Sprintf("<p><strong>Bail :</strong> %s | <strong>Date :</strong> %s</p>",
		html.EscapeString(insp.LeaseID),
		insp.InspectionDate.Format("2006-01-02"),
	))

	sb.WriteString("<h2>Relevé par Pièce et Élément</h2><table><thead><tr><th>Pièce</th><th>Élément</th><th>État</th><th>Observation</th><th>Photos</th></tr></thead><tbody>")
	for _, it := range insp.Items {
		photoCell := "0"
		if len(it.PhotoR2Keys) > 0 {
			photoCell = fmt.Sprintf("%d (%s)", len(it.PhotoR2Keys), html.EscapeString(strings.Join(it.PhotoR2Keys, ", ")))
		}
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			html.EscapeString(it.RoomName),
			html.EscapeString(it.ElementName),
			html.EscapeString(translateConditionFr(it.Condition)),
			html.EscapeString(it.Observation),
			photoCell,
		))
	}
	sb.WriteString("</tbody></table>")

	if insp.MeterReadings != (domain.MeterReadings{}) {
		sb.WriteString("<h2>Relevés de Compteurs</h2>")
		sb.WriteString(fmt.Sprintf("<p>Eau : %d m³ | Électricité (PDL %s) : %d kWh | Gaz (PCE %s) : %d m³</p>",
			insp.MeterReadings.WaterM3,
			html.EscapeString(insp.MeterReadings.ElectricityPDL),
			insp.MeterReadings.ElectricityKWh,
			html.EscapeString(insp.MeterReadings.GasPCE),
			insp.MeterReadings.GasM3,
		))
	}

	if insp.KeysHandedOverCount > 0 || strings.TrimSpace(insp.KeysDescription) != "" {
		sb.WriteString("<h2>Remise des Clés</h2>")
		sb.WriteString(fmt.Sprintf("<p>Nombre : %d — %s</p>",
			insp.KeysHandedOverCount,
			html.EscapeString(insp.KeysDescription),
		))
	}

	if insp.EstimatedRepairCostCents > 0 {
		sb.WriteString("<h2>Retenues pour Réparations Locatives</h2>")
		sb.WriteString(fmt.Sprintf("<p>Montant estimé : %s</p>",
			html.EscapeString(insp.EstimatedRepairCostCents.FormatEUR()),
		))
	}

	if hasComparison {
		sb.WriteString(fmt.Sprintf("<h2>Comparatif Entrée / Sortie (%d dégradation(s) constatée(s))</h2>", len(degradations)))
		sb.WriteString("<table><thead><tr><th>Pièce</th><th>Élément</th><th>Entrée</th><th>Sortie</th><th>Dégradation</th></tr></thead><tbody>")
		for _, comp := range comparisons {
			degLabel := "Non"
			if comp.Degraded {
				degLabel = "OUI"
			}
			sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				html.EscapeString(comp.RoomName),
				html.EscapeString(comp.ElementName),
				html.EscapeString(translateConditionFr(comp.EntryCondition)),
				html.EscapeString(translateConditionFr(comp.ExitCondition)),
				degLabel,
			))
		}
		sb.WriteString("</tbody></table>")
	}

	sb.WriteString("</body></html>")
	return sb.String()
}

func itemCompositeKey(room, element string) string {
	return strings.ToLower(strings.TrimSpace(room)) + "::" + strings.ToLower(strings.TrimSpace(element))
}

func cleanStringSlice(in []string) []string {
	var out []string
	for _, s := range in {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if out == nil {
		return []string{}
	}
	return out
}

func copyStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

func parseDateOrToday(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		now := time.Now().UTC()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Parse("2006-01-02", trimmed)
}

