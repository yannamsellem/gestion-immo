package application

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	leaseapp "github.com/yannamsellem/gestion-immo/internal/modules/lease/application"
	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	paymentapp "github.com/yannamsellem/gestion-immo/internal/modules/payment/application"
	portfolioapp "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/application"
	"github.com/yannamsellem/gestion-immo/internal/platform/id"
	"github.com/yannamsellem/gestion-immo/internal/platform/money"
	"github.com/yannamsellem/gestion-immo/internal/platform/r2"
	"github.com/yannamsellem/gestion-immo/internal/platform/templating"
)

// PortfolioPort defines the portfolio application operations consumed by the document bounded context.
type PortfolioPort interface {
	GetEntityPortfolioSummary(ctx context.Context, entityID string) (portfolioapp.EntityPortfolioSummaryDTO, error)
}

type legalEntityLookupPort interface {
	GetLegalEntityByID(ctx context.Context, entityID string) (portfolioapp.LegalEntityDTO, error)
}

// LeasePort defines the lease application operations consumed by the document bounded context.
type LeasePort interface {
	GetLeaseByID(ctx context.Context, leaseID string) (leaseapp.LeaseDTO, error)
	GetTenantByID(ctx context.Context, tenantID string) (leaseapp.TenantDTO, error)
	ListUnpaidRentCallsByLease(ctx context.Context, leaseID string) ([]leasedomain.RentCall, error)
	ListRentCallsByLease(ctx context.Context, leaseID string) ([]leaseapp.RentCallDTO, error)
}

// PaymentPort defines the payment application operations consumed by the document bounded context.
type PaymentPort interface {
	ListPaymentsByLease(ctx context.Context, leaseID string) ([]paymentapp.PaymentReceiptDTO, error)
}

// RegisterTemplateCmd carries the input for registering a customizable .docx or HTML template.
type RegisterTemplateCmd struct {
	Name         string `json:"name"`
	TemplateType string `json:"template_type"`
	Format       string `json:"format"`
	DocxBytes    []byte `json:"docx_bytes,omitempty"`
	HTMLBody     string `json:"html_body,omitempty"`
}

// DocumentTemplateDTO is an immutable projection of a domain.DocumentTemplate.
type DocumentTemplateDTO struct {
	ID           string                `json:"id" format:"uuid"`
	Name         string                `json:"name"`
	TemplateType domain.TemplateType   `json:"template_type"`
	Format       domain.TemplateFormat `json:"format"`
	R2Key        string                `json:"r2_key,omitempty"`
	HTMLBody     string                `json:"html_body,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// AttachFileCmd carries the input for uploading and indexing a document/attachment on Cloudflare R2.
type AttachFileCmd struct {
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	LeaseID       string `json:"lease_id,omitempty"`
	Category      string `json:"category"`
	FileName      string `json:"file_name"`
	ContentType   string `json:"content_type"`
	Content       []byte `json:"content"`
}

// GenerateDocCmd carries the input for generating a document (.docx or PDF) from a template.
type GenerateDocCmd struct {
	TemplateID     string                         `json:"template_id" format:"uuid"`
	LegalEntityID  string                         `json:"legal_entity_id,omitempty"`
	LeaseID        string                         `json:"lease_id,omitempty"`
	Variables      map[string]string              `json:"variables,omitempty"`
	Tables         map[string][]map[string]string `json:"tables,omitempty"`
	OutputFileName string                         `json:"output_file_name,omitempty"`
}

// StoredFileDTO is an immutable projection of a domain.StoredFile with a presigned R2 download URL.
type StoredFileDTO struct {
	ID            string    `json:"id" format:"uuid"`
	LegalEntityID string    `json:"legal_entity_id,omitempty"`
	LeaseID       string    `json:"lease_id,omitempty"`
	Category      string    `json:"category"`
	FileName      string    `json:"file_name"`
	ContentType   string    `json:"content_type"`
	R2Key         string    `json:"r2_key"`
	SizeBytes     int64     `json:"size_bytes"`
	DownloadURL   string    `json:"download_url"`
	CreatedAt     time.Time `json:"created_at"`
}

// GLIDossierArchiveDTO is an immutable projection of a generated 1-click GLI claim ZIP dossier.
type GLIDossierArchiveDTO struct {
	ID               string            `json:"id" format:"uuid"`
	LeaseID          string            `json:"lease_id" format:"uuid"`
	LegalEntityID    string            `json:"legal_entity_id" format:"uuid"`
	TenantName       string            `json:"tenant_name"`
	TemplateID       string            `json:"template_id"`
	GLIPolicyNumber  string            `json:"gli_policy_number"`
	TotalUnpaidCents money.AmountCents `json:"total_unpaid_cents"`
	UnpaidMonths     []string          `json:"unpaid_months"`
	ArchiveR2Key     string            `json:"archive_r2_key"`
	DownloadURL      string            `json:"download_url"`
	IncludedFiles    []string          `json:"included_files"`
	ArchiveSizeBytes int64             `json:"archive_size_bytes"`
	CreatedAt        time.Time         `json:"created_at"`
}

// Service orchestrates document templates, R2 file storage, .docx/PDF generation, and 1-click GLI claim dossiers.
type Service struct {
	templates    domain.DocumentTemplateRepository
	storedFiles  domain.StoredFileRepository
	gliDossiers  domain.GLIDossierRepository
	uow          domain.UnitOfWork
	storage      r2.ObjectStorage
	docxEngine   *templating.DocxEngine
	pdfEngine    templating.PdfEngine
	portfolioSvc PortfolioPort
	leaseSvc     LeasePort
	paymentSvc   PaymentPort
}

// NewService constructs a document application Service via constructor injection.
func NewService(
	templates domain.DocumentTemplateRepository,
	storedFiles domain.StoredFileRepository,
	gliDossiers domain.GLIDossierRepository,
	uow domain.UnitOfWork,
	storage r2.ObjectStorage,
	docxEngine *templating.DocxEngine,
	pdfEngine templating.PdfEngine,
	portfolioSvc PortfolioPort,
	leaseSvc LeasePort,
	paymentSvc PaymentPort,
) *Service {
	return &Service{
		templates:    templates,
		storedFiles:  storedFiles,
		gliDossiers:  gliDossiers,
		uow:          uow,
		storage:      storage,
		docxEngine:   docxEngine,
		pdfEngine:    pdfEngine,
		portfolioSvc: portfolioSvc,
		leaseSvc:     leaseSvc,
		paymentSvc:   paymentSvc,
	}
}

// RegisterTemplate validates and stores a customizable .docx or HTML template in R2/PostgreSQL.
func (s *Service) RegisterTemplate(ctx context.Context, cmd RegisterTemplateCmd) (DocumentTemplateDTO, error) {
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return DocumentTemplateDTO{}, fmt.Errorf("%w: template name is required", domain.ErrValidation)
	}

	tplTypeStr := strings.ToUpper(strings.TrimSpace(cmd.TemplateType))
	if tplTypeStr == "" {
		tplTypeStr = string(domain.TemplateTypeCustom)
	}
	tplType := domain.TemplateType(tplTypeStr)
	if !tplType.IsValid() {
		return DocumentTemplateDTO{}, fmt.Errorf("%w: unsupported template_type %q", domain.ErrValidation, cmd.TemplateType)
	}

	formatStr := strings.ToUpper(strings.TrimSpace(cmd.Format))
	if formatStr == "" {
		if len(cmd.DocxBytes) > 0 {
			formatStr = string(domain.FormatDocx)
		} else {
			formatStr = string(domain.FormatHTML)
		}
	}
	format := domain.TemplateFormat(formatStr)
	if !format.IsValid() {
		return DocumentTemplateDTO{}, fmt.Errorf("%w: unsupported format %q", domain.ErrValidation, cmd.Format)
	}

	tplID, err := id.NewV7()
	if err != nil {
		return DocumentTemplateDTO{}, err
	}

	var r2Key string
	htmlBody := strings.TrimSpace(cmd.HTMLBody)
	if htmlBody != "" {
		if err := validateSafeHTMLTemplate(htmlBody); err != nil {
			return DocumentTemplateDTO{}, err
		}
	}

	if format == domain.FormatDocx {
		docxPayload := cmd.DocxBytes
		if len(docxPayload) == 0 {
			bodyText := htmlBody
			if bodyText == "" {
				bodyText = "Document {{societe.nom}} - {{locataire.nom}}"
			}
			docxPayload, err = templating.BuildMinimalDocx(bodyText)
			if err != nil {
				return DocumentTemplateDTO{}, fmt.Errorf("document: build minimal docx: %w", err)
			}
		}
		r2Key = fmt.Sprintf("templates/%s.docx", tplID)
		if err := s.storage.Upload(ctx, r2Key, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", bytes.NewReader(docxPayload)); err != nil {
			return DocumentTemplateDTO{}, err
		}
	} else if htmlBody == "" {
		return DocumentTemplateDTO{}, fmt.Errorf("%w: html_body is required for HTML template", domain.ErrValidation)
	}

	now := time.Now().UTC()
	tpl := domain.DocumentTemplate{
		ID:           tplID,
		Name:         name,
		TemplateType: tplType,
		Format:       format,
		R2Key:        r2Key,
		HTMLBody:     htmlBody,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.templates.SaveTemplate(ctx, tpl); err != nil {
		return DocumentTemplateDTO{}, err
	}

	return toTemplateDTO(tpl), nil
}

// ListTemplates returns all registered document templates.
func (s *Service) ListTemplates(ctx context.Context) ([]DocumentTemplateDTO, error) {
	list, err := s.templates.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DocumentTemplateDTO, 0, len(list))
	for _, tpl := range list {
		out = append(out, toTemplateDTO(tpl))
	}
	return out, nil
}

// AttachFileToR2 uploads a document or attachment (signed lease, EDL, ID) to R2 and records a StoredFile.
func (s *Service) AttachFileToR2(ctx context.Context, cmd AttachFileCmd) (StoredFileDTO, error) {
	fileName := sanitizeFileName(cmd.FileName)
	if fileName == "" {
		return StoredFileDTO{}, fmt.Errorf("%w: file_name is required", domain.ErrValidation)
	}
	if len(cmd.Content) == 0 {
		return StoredFileDTO{}, fmt.Errorf("%w: file content must not be empty", domain.ErrValidation)
	}

	entityID := strings.TrimSpace(cmd.LegalEntityID)
	leaseID := strings.TrimSpace(cmd.LeaseID)
	if leaseID != "" {
		if !id.IsValidUUID(leaseID) {
			return StoredFileDTO{}, fmt.Errorf("%w: invalid lease_id", domain.ErrValidation)
		}
		if s.leaseSvc != nil {
			if _, err := s.leaseSvc.GetLeaseByID(ctx, leaseID); err != nil {
				return StoredFileDTO{}, err
			}
		}
	}
	if entityID != "" {
		if !id.IsValidUUID(entityID) {
			return StoredFileDTO{}, fmt.Errorf("%w: invalid legal_entity_id", domain.ErrValidation)
		}
		if s.portfolioSvc != nil {
			if _, err := s.lookupLegalEntity(ctx, entityID); err != nil {
				return StoredFileDTO{}, err
			}
		}
	}

	fileID, err := id.NewV7()
	if err != nil {
		return StoredFileDTO{}, err
	}

	category := strings.ToUpper(strings.TrimSpace(cmd.Category))
	if category == "" {
		category = "ATTACHMENT"
	}

	contentType := strings.TrimSpace(cmd.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	scope := "general"
	if leaseID != "" {
		scope = "leases/" + leaseID
	} else if entityID != "" {
		scope = "entities/" + entityID
	}
	r2Key := fmt.Sprintf("files/%s/%s_%s", scope, fileID, fileName)

	if err := s.storage.Upload(ctx, r2Key, contentType, bytes.NewReader(cmd.Content)); err != nil {
		return StoredFileDTO{}, err
	}

	now := time.Now().UTC()
	sf := domain.StoredFile{
		ID:            fileID,
		LegalEntityID: entityID,
		LeaseID:       leaseID,
		Category:      category,
		FileName:      fileName,
		ContentType:   contentType,
		R2Key:         r2Key,
		SizeBytes:     int64(len(cmd.Content)),
		CreatedAt:     now,
	}

	if err := s.storedFiles.SaveStoredFile(ctx, sf); err != nil {
		return StoredFileDTO{}, err
	}

	downloadURL, _ := s.storage.PresignGetURL(ctx, sf.R2Key, 1*time.Hour)
	return toStoredFileDTO(sf, downloadURL), nil
}

// GenerateFromTemplate interpolates variables and table rows into a .docx or HTML/PDF template,
// stores the generated document in Cloudflare R2, and returns its StoredFileDTO.
func (s *Service) GenerateFromTemplate(ctx context.Context, cmd GenerateDocCmd) (StoredFileDTO, error) {
	tplID := strings.TrimSpace(cmd.TemplateID)
	if tplID == "" {
		return StoredFileDTO{}, fmt.Errorf("%w: template_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(tplID) {
		return StoredFileDTO{}, fmt.Errorf("%w: template_id must be a valid UUID", domain.ErrValidation)
	}

	entityID := strings.TrimSpace(cmd.LegalEntityID)
	if entityID != "" && !id.IsValidUUID(entityID) {
		return StoredFileDTO{}, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
	}
	leaseID := strings.TrimSpace(cmd.LeaseID)
	if leaseID != "" && !id.IsValidUUID(leaseID) {
		return StoredFileDTO{}, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}

	tpl, err := s.templates.FindTemplateByID(ctx, tplID)
	if err != nil {
		return StoredFileDTO{}, err
	}

	vars := make(map[string]string)

	if leaseID != "" && s.leaseSvc != nil {
		if lease, lErr := s.leaseSvc.GetLeaseByID(ctx, leaseID); lErr == nil {
			if entityID == "" {
				entityID = lease.LegalEntityID
			}
			vars["bail.id"] = lease.ID
			vars["bail.type"] = string(lease.LeaseType)
			vars["bail.loyer_hc"] = lease.RentNetCents.FormatEUR()
			vars["bail.charges"] = lease.ChargesCents.FormatEUR()
			vars["bail.depot_garantie"] = lease.SecurityDepositCents.FormatEUR()
			vars["bail.gli_police"] = lease.GLIPolicyNumber
			vars["bail.garant"] = lease.GuarantorName

			if tenant, tErr := s.leaseSvc.GetTenantByID(ctx, lease.TenantID); tErr == nil {
				vars["locataire.nom"] = tenant.FullName
				vars["locataire.email"] = tenant.Email
				vars["locataire.telephone"] = tenant.Phone
				vars["locataire.adresse"] = tenant.Address
			}
		}
	}

	if entityID != "" && s.portfolioSvc != nil {
		if entity, pErr := s.lookupLegalEntity(ctx, entityID); pErr == nil {
			vars["societe.nom"] = entity.Name
			vars["societe.siret"] = entity.SIRET
			vars["societe.adresse"] = entity.Address
			vars["societe.capital"] = entity.ShareCapitalCents.FormatEUR()
		}
	}

	for k, v := range cmd.Variables {
		vars[k] = v
	}

	var (
		renderedBytes []byte
		contentType   string
		defaultExt    string
	)

	if tpl.Format == domain.FormatDocx {
		var tplBytes []byte
		if tpl.R2Key != "" {
			rc, dErr := s.storage.Download(ctx, tpl.R2Key)
			if dErr == nil {
				tplBytes, _ = io.ReadAll(rc)
				_ = rc.Close()
			}
		}
		if len(tplBytes) == 0 {
			bodyText := tpl.HTMLBody
			if bodyText == "" {
				bodyText = "Bailleur: {{societe.nom}} - Locataire: {{locataire.nom}}"
			}
			tplBytes, err = templating.BuildMinimalDocx(bodyText)
			if err != nil {
				return StoredFileDTO{}, err
			}
		}

		renderedBytes, err = s.docxEngine.RenderDocx(tplBytes, vars, cmd.Tables)
		if err != nil {
			return StoredFileDTO{}, err
		}
		contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		defaultExt = ".docx"
	} else {
		htmlContent := interpolateHTMLTemplate(tpl.HTMLBody, vars)
		renderedBytes, err = s.pdfEngine.RenderHTMLToPDF(ctx, htmlContent)
		if err != nil {
			return StoredFileDTO{}, err
		}
		contentType = "application/pdf"
		defaultExt = ".pdf"
	}

	fileName := sanitizeFileName(cmd.OutputFileName)
	if fileName == "" {
		fileName = strings.ToLower(strings.ReplaceAll(tpl.Name, " ", "_")) + defaultExt
	} else if !strings.HasSuffix(strings.ToLower(fileName), defaultExt) {
		fileName += defaultExt
	}

	return s.AttachFileToR2(ctx, AttachFileCmd{
		LegalEntityID: entityID,
		LeaseID:       leaseID,
		Category:      string(tpl.TemplateType),
		FileName:      fileName,
		ContentType:   contentType,
		Content:       renderedBytes,
	})
}

// AssembleGLIDossier builds a complete 1-click GLI (Garantie Loyers Impayés) claim ZIP archive
// for a lease in negative position, including:
// 1. The populated GLI declaration .docx with entity, property, tenant, and unpaid rent table data,
// 2. The tenant account ledger PDF detailing unpaid rent calls and payment history,
// 3. The formal notice (mise en demeure) PDF,
// 4. All lease/EDL attachments stored on R2 for the lease.
// It persists both the StoredFile metadata for the ZIP archive and the GLIDossier aggregate atomically via UnitOfWork.
func (s *Service) AssembleGLIDossier(ctx context.Context, leaseID string, templateID string) (GLIDossierArchiveDTO, error) {
	trimmedLeaseID := strings.TrimSpace(leaseID)
	if trimmedLeaseID == "" {
		return GLIDossierArchiveDTO{}, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmedLeaseID) {
		return GLIDossierArchiveDTO{}, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}
	if trimmedTpl := strings.TrimSpace(templateID); trimmedTpl != "" && !id.IsValidUUID(trimmedTpl) {
		return GLIDossierArchiveDTO{}, fmt.Errorf("%w: template_id must be a valid UUID", domain.ErrValidation)
	}
	if s.leaseSvc == nil {
		return GLIDossierArchiveDTO{}, fmt.Errorf("%w: lease service port is not configured", domain.ErrValidation)
	}

	lease, err := s.leaseSvc.GetLeaseByID(ctx, trimmedLeaseID)
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	tenant, err := s.leaseSvc.GetTenantByID(ctx, lease.TenantID)
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	unpaidCalls, err := s.leaseSvc.ListUnpaidRentCallsByLease(ctx, lease.ID)
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	sort.SliceStable(unpaidCalls, func(i, j int) bool {
		return unpaidCalls[i].PeriodMonth < unpaidCalls[j].PeriodMonth
	})

	var (
		totalUnpaidCents money.AmountCents
		unpaidMonths     []string
		unpaidTableRows  []map[string]string
	)
	for _, rc := range unpaidCalls {
		if rc.RemainingDueCents <= 0 {
			continue
		}
		totalUnpaidCents = totalUnpaidCents.Add(rc.RemainingDueCents)
		unpaidMonths = append(unpaidMonths, rc.PeriodMonth)
		unpaidTableRows = append(unpaidTableRows, map[string]string{
			"periode":    rc.PeriodMonth,
			"echeance":   rc.DueDate.Format("2006-01-02"),
			"loyer_ht":   rc.RentNetCents.FormatEUR(),
			"charges":    rc.ChargesCents.FormatEUR(),
			"tva":        rc.VATCents.FormatEUR(),
			"total_du":   rc.TotalDueCents.FormatEUR(),
			"montant_du": rc.RemainingDueCents.FormatEUR(),
			"statut":     string(rc.Status),
		})
	}

	if totalUnpaidCents <= 0 || len(unpaidMonths) == 0 {
		return GLIDossierArchiveDTO{}, domain.ErrNoUnpaidRentForGLI
	}

	entityName := lease.LegalEntityID
	entitySIRET := ""
	entityAddress := ""
	propertyAddress := ""
	lotLabel := lease.LotID

	if s.portfolioSvc != nil {
		if summary, pErr := s.portfolioSvc.GetEntityPortfolioSummary(ctx, lease.LegalEntityID); pErr == nil {
			entityName = summary.Entity.Name
			entitySIRET = summary.Entity.SIRET
			entityAddress = summary.Entity.Address
			for _, prop := range summary.Properties {
				for _, lot := range prop.Lots {
					if lot.ID == lease.LotID {
						propertyAddress = prop.Address
						lotLabel = fmt.Sprintf("%s - %s", prop.Label, lot.Label)
						break
					}
				}
			}
		}
	}

	vars := map[string]string{
		"societe.nom":         entityName,
		"societe.siret":       entitySIRET,
		"societe.adresse":     entityAddress,
		"bien.adresse":        propertyAddress,
		"bien.lot":            lotLabel,
		"locataire.nom":       tenant.FullName,
		"locataire.email":     tenant.Email,
		"locataire.telephone": tenant.Phone,
		"bail.id":             lease.ID,
		"bail.type":           string(lease.LeaseType),
		"bail.gli_police":     lease.GLIPolicyNumber,
		"bail.loyer_hc":       lease.RentNetCents.FormatEUR(),
		"bail.charges":        lease.ChargesCents.FormatEUR(),
		"impaye.total":        totalUnpaidCents.FormatEUR(),
		"impaye.mois":         strings.Join(unpaidMonths, ", "),
	}

	docxTemplateBytes, resolvedTemplateID, err := s.resolveGLIDocxTemplate(ctx, templateID)
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	gliDeclarationDocx, err := s.docxEngine.RenderDocx(docxTemplateBytes, vars, map[string][]map[string]string{
		"impayes": unpaidTableRows,
	})
	if err != nil {
		return GLIDossierArchiveDTO{}, fmt.Errorf("document: render GLI declaration docx: %w", err)
	}

	var payments []paymentapp.PaymentReceiptDTO
	if s.paymentSvc != nil {
		payments, _ = s.paymentSvc.ListPaymentsByLease(ctx, lease.ID)
	}

	ledgerHTML := buildTenantLedgerHTML(entityName, tenant.FullName, lease, unpaidCalls, payments, totalUnpaidCents)
	ledgerPDF, err := s.pdfEngine.RenderHTMLToPDF(ctx, ledgerHTML)
	if err != nil {
		return GLIDossierArchiveDTO{}, fmt.Errorf("document: render tenant ledger pdf: %w", err)
	}

	noticeHTML := buildFormalNoticeHTML(entityName, entityAddress, tenant.FullName, lease, unpaidMonths, totalUnpaidCents)
	noticePDF, err := s.pdfEngine.RenderHTMLToPDF(ctx, noticeHTML)
	if err != nil {
		return GLIDossierArchiveDTO{}, fmt.Errorf("document: render formal notice pdf: %w", err)
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	var includedFiles []string

	addZipEntry := func(entryName string, data []byte) error {
		w, zErr := zw.Create(entryName)
		if zErr != nil {
			return zErr
		}
		if _, zErr := w.Write(data); zErr != nil {
			return zErr
		}
		includedFiles = append(includedFiles, entryName)
		return nil
	}

	if err := addZipEntry("01_declaration_gli.docx", gliDeclarationDocx); err != nil {
		_ = zw.Close()
		return GLIDossierArchiveDTO{}, err
	}
	if err := addZipEntry("02_extrait_compte_locataire.pdf", ledgerPDF); err != nil {
		_ = zw.Close()
		return GLIDossierArchiveDTO{}, err
	}
	if err := addZipEntry("03_mise_en_demeure.pdf", noticePDF); err != nil {
		_ = zw.Close()
		return GLIDossierArchiveDTO{}, err
	}

	const maxGLIAttachmentBytes = 64 << 20
	storedAttachments, err := s.storedFiles.ListStoredFilesByLease(ctx, lease.ID)
	if err == nil {
		usedEntryNames := make(map[string]int)
		var totalAttachmentBytes int64
		for _, att := range storedAttachments {
			if strings.HasSuffix(att.R2Key, ".zip") {
				continue
			}
			rc, dErr := s.storage.Download(ctx, att.R2Key)
			if dErr != nil {
				continue
			}
			remaining := maxGLIAttachmentBytes - totalAttachmentBytes
			if remaining <= 0 {
				_ = rc.Close()
				_ = zw.Close()
				return GLIDossierArchiveDTO{}, fmt.Errorf("%w: GLI attachments exceed max cumulative size (%d bytes)", domain.ErrValidation, maxGLIAttachmentBytes)
			}
			attBytes, rErr := io.ReadAll(io.LimitReader(rc, remaining+1))
			_ = rc.Close()
			if rErr != nil || len(attBytes) == 0 {
				continue
			}
			totalAttachmentBytes += int64(len(attBytes))
			if totalAttachmentBytes > maxGLIAttachmentBytes {
				_ = zw.Close()
				return GLIDossierArchiveDTO{}, fmt.Errorf("%w: GLI attachments exceed max cumulative size (%d bytes)", domain.ErrValidation, maxGLIAttachmentBytes)
			}
			entryName := deduplicateAttachmentEntryName(sanitizeFileName(att.FileName), usedEntryNames)
			if err := addZipEntry(entryName, attBytes); err != nil {
				_ = zw.Close()
				return GLIDossierArchiveDTO{}, err
			}
		}
	}

	if err := zw.Close(); err != nil {
		return GLIDossierArchiveDTO{}, fmt.Errorf("document: close GLI zip archive: %w", err)
	}

	dossierID, err := id.NewV7()
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}
	archiveFileID, err := id.NewV7()
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	archiveBytes := zipBuf.Bytes()
	archiveR2Key := fmt.Sprintf("gli_dossiers/%s/%s/%s.zip", lease.LegalEntityID, lease.ID, dossierID)
	if err := s.storage.Upload(ctx, archiveR2Key, "application/zip", bytes.NewReader(archiveBytes)); err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	now := time.Now().UTC()
	archiveStoredFile := domain.StoredFile{
		ID:            archiveFileID,
		LegalEntityID: lease.LegalEntityID,
		LeaseID:       lease.ID,
		Category:      "GLI_DOSSIER",
		FileName:      dossierID + ".zip",
		ContentType:   "application/zip",
		R2Key:         archiveR2Key,
		SizeBytes:     int64(len(archiveBytes)),
		CreatedAt:     now,
	}
	dossier := domain.GLIDossier{
		ID:               dossierID,
		LeaseID:          lease.ID,
		LegalEntityID:    lease.LegalEntityID,
		TemplateID:       resolvedTemplateID,
		GLIPolicyNumber:  lease.GLIPolicyNumber,
		TotalUnpaidCents: totalUnpaidCents,
		UnpaidMonths:     unpaidMonths,
		ArchiveR2Key:     archiveR2Key,
		IncludedFiles:    includedFiles,
		CreatedAt:        now,
	}

	err = s.uow.WithinTransaction(ctx, func(txCtx context.Context, repos domain.Repositories) error {
		if err := repos.StoredFiles.SaveStoredFile(txCtx, archiveStoredFile); err != nil {
			return err
		}
		return repos.GLIDossiers.SaveGLIDossier(txCtx, dossier)
	})
	if err != nil {
		return GLIDossierArchiveDTO{}, err
	}

	downloadURL, _ := s.storage.PresignGetURL(ctx, archiveR2Key, 24*time.Hour)

	return GLIDossierArchiveDTO{
		ID:               dossier.ID,
		LeaseID:          dossier.LeaseID,
		LegalEntityID:    dossier.LegalEntityID,
		TenantName:       tenant.FullName,
		TemplateID:       dossier.TemplateID,
		GLIPolicyNumber:  dossier.GLIPolicyNumber,
		TotalUnpaidCents: dossier.TotalUnpaidCents,
		UnpaidMonths:     dossier.UnpaidMonths,
		ArchiveR2Key:     dossier.ArchiveR2Key,
		DownloadURL:      downloadURL,
		IncludedFiles:    dossier.IncludedFiles,
		ArchiveSizeBytes: int64(len(archiveBytes)),
		CreatedAt:        dossier.CreatedAt,
	}, nil
}

// ListStoredFilesByLease returns all stored files attached to a lease.
func (s *Service) ListStoredFilesByLease(ctx context.Context, leaseID string) ([]StoredFileDTO, error) {
	trimmed := strings.TrimSpace(leaseID)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: lease_id is required", domain.ErrValidation)
	}
	if !id.IsValidUUID(trimmed) {
		return nil, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
	}
	files, err := s.storedFiles.ListStoredFilesByLease(ctx, trimmed)
	if err != nil {
		return nil, err
	}
	out := make([]StoredFileDTO, 0, len(files))
	for _, f := range files {
		url, _ := s.storage.PresignGetURL(ctx, f.R2Key, 1*time.Hour)
		out = append(out, toStoredFileDTO(f, url))
	}
	return out, nil
}

// SearchDocuments searches stored files by entity, lease, or filename/category substring.
func (s *Service) SearchDocuments(ctx context.Context, entityID string, leaseID string, query string) ([]StoredFileDTO, error) {
	var files []domain.StoredFile
	var err error

	trimmedLease := strings.TrimSpace(leaseID)
	trimmedEntity := strings.TrimSpace(entityID)
	if trimmedLease != "" {
		if !id.IsValidUUID(trimmedLease) {
			return nil, fmt.Errorf("%w: lease_id must be a valid UUID", domain.ErrValidation)
		}
		files, err = s.storedFiles.ListStoredFilesByLease(ctx, trimmedLease)
	} else if trimmedEntity != "" {
		if !id.IsValidUUID(trimmedEntity) {
			return nil, fmt.Errorf("%w: legal_entity_id must be a valid UUID", domain.ErrValidation)
		}
		files, err = s.storedFiles.ListStoredFilesByEntity(ctx, trimmedEntity)
	} else {
		return []StoredFileDTO{}, nil
	}
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]StoredFileDTO, 0, len(files))
	for _, f := range files {
		if q != "" && !strings.Contains(strings.ToLower(f.FileName), q) && !strings.Contains(strings.ToLower(f.Category), q) {
			continue
		}
		url, _ := s.storage.PresignGetURL(ctx, f.R2Key, 1*time.Hour)
		out = append(out, toStoredFileDTO(f, url))
	}
	return out, nil
}

func (s *Service) resolveGLIDocxTemplate(ctx context.Context, templateID string) ([]byte, string, error) {
	trimmed := strings.TrimSpace(templateID)
	if trimmed != "" {
		if !id.IsValidUUID(trimmed) {
			return nil, "", fmt.Errorf("%w: template_id must be a valid UUID", domain.ErrValidation)
		}
		tpl, err := s.templates.FindTemplateByID(ctx, trimmed)
		if err != nil {
			return nil, "", err
		}
		if tpl.R2Key != "" {
			rc, dErr := s.storage.Download(ctx, tpl.R2Key)
			if dErr == nil {
				data, rErr := io.ReadAll(rc)
				_ = rc.Close()
				if rErr == nil && len(data) > 0 {
					return data, tpl.ID, nil
				}
			}
		}
		if tpl.HTMLBody != "" {
			data, bErr := templating.BuildMinimalDocx(tpl.HTMLBody)
			if bErr == nil {
				return data, tpl.ID, nil
			}
		}
	}

	defaultText := "DECLARATION DE SINISTRE GLI - Bailleur: {{societe.nom}} (SIRET: {{societe.siret}}) - " +
		"Bien: {{bien.lot}} ({{bien.adresse}}) - Locataire: {{locataire.nom}} - " +
		"Contrat GLI: {{bail.gli_police}} - Mois impayés: {{impaye.mois}} - Total impayé: {{impaye.total}}"
	data, err := templating.BuildMinimalDocx(defaultText)
	if err != nil {
		return nil, "", err
	}
	return data, trimmed, nil
}

func buildTenantLedgerHTML(
	entityName string,
	tenantName string,
	lease leaseapp.LeaseDTO,
	unpaidCalls []leasedomain.RentCall,
	payments []paymentapp.PaymentReceiptDTO,
	totalUnpaid money.AmountCents,
) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html><html lang=\"fr\"><head><meta charset=\"UTF-8\"><title>Extrait de Compte Locataire</title></head><body>")
	sb.WriteString(fmt.Sprintf("<h1>Extrait de Compte Locataire — %s</h1>", html.EscapeString(entityName)))
	sb.WriteString(fmt.Sprintf("<p><strong>Locataire :</strong> %s | <strong>Bail :</strong> %s | <strong>Police GLI :</strong> %s</p>",
		html.EscapeString(tenantName), html.EscapeString(lease.ID), html.EscapeString(lease.GLIPolicyNumber)))

	sb.WriteString("<h2>Échéances Impayées</h2><table><thead><tr><th>Période</th><th>Total Appelé</th><th>Reste Dû</th><th>Statut</th></tr></thead><tbody>")
	for _, rc := range unpaidCalls {
		if rc.RemainingDueCents <= 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			html.EscapeString(rc.PeriodMonth),
			html.EscapeString(rc.TotalDueCents.FormatEUR()),
			html.EscapeString(rc.RemainingDueCents.FormatEUR()),
			html.EscapeString(string(rc.Status)),
		))
	}
	sb.WriteString("</tbody></table>")

	if len(payments) > 0 {
		sb.WriteString("<h2>Historique des Règlements</h2><ul>")
		for _, p := range payments {
			sb.WriteString(fmt.Sprintf("<li>%s — %s (%s) : %s</li>",
				p.PaymentDate.Format("2006-01-02"),
				html.EscapeString(p.Label),
				html.EscapeString(string(p.Method)),
				html.EscapeString(p.AmountCents.FormatEUR()),
			))
		}
		sb.WriteString("</ul>")
	}

	sb.WriteString(fmt.Sprintf("<h3>Solde Débiteur Total : %s</h3>", html.EscapeString(totalUnpaid.FormatEUR())))
	sb.WriteString("</body></html>")
	return sb.String()
}

func buildFormalNoticeHTML(
	entityName string,
	entityAddress string,
	tenantName string,
	lease leaseapp.LeaseDTO,
	unpaidMonths []string,
	totalUnpaid money.AmountCents,
) string {
	return fmt.Sprintf(
		"<!DOCTYPE html><html lang=\"fr\"><head><meta charset=\"UTF-8\"><title>Mise en Demeure</title></head><body>"+
			"<h1>MISE EN DEMEURE DE PAYER LES LOYERS</h1>"+
			"<p>Bailleur : %s (%s)</p>"+
			"<p>Destinataire : %s (Bail %s)</p>"+
			"<p>Périodes impayées : %s</p>"+
			"<p>Montant total exigible : <strong>%s</strong></p>"+
			"</body></html>",
		html.EscapeString(entityName),
		html.EscapeString(entityAddress),
		html.EscapeString(tenantName),
		html.EscapeString(lease.ID),
		html.EscapeString(strings.Join(unpaidMonths, ", ")),
		html.EscapeString(totalUnpaid.FormatEUR()),
	)
}

func validateSafeHTMLTemplate(htmlBody string) error {
	lower := strings.ToLower(htmlBody)
	for _, forbidden := range []string{
		"<script",
		"<iframe",
		"<object",
		"<embed",
		"<link",
		"javascript:",
		"file://",
	} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("%w: html_body contains forbidden active or external tags", domain.ErrValidation)
		}
	}
	return nil
}

func interpolateHTMLTemplate(rawHTML string, vars map[string]string) string {
	out := rawHTML
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = strings.ReplaceAll(out, "{{"+k+"}}", html.EscapeString(vars[k]))
	}
	return out
}

func (s *Service) lookupLegalEntity(ctx context.Context, entityID string) (portfolioapp.LegalEntityDTO, error) {
	if lookup, ok := s.portfolioSvc.(legalEntityLookupPort); ok {
		return lookup.GetLegalEntityByID(ctx, entityID)
	}
	summary, err := s.portfolioSvc.GetEntityPortfolioSummary(ctx, entityID)
	if err != nil {
		return portfolioapp.LegalEntityDTO{}, err
	}
	return summary.Entity, nil
}

func deduplicateAttachmentEntryName(baseName string, usedEntryNames map[string]int) string {
	if baseName == "" {
		baseName = "attachment"
	}
	usedEntryNames[baseName]++
	count := usedEntryNames[baseName]
	if count == 1 {
		return "pieces_jointes/" + baseName
	}
	if dot := strings.LastIndex(baseName, "."); dot > 0 {
		return fmt.Sprintf("pieces_jointes/%s_%d%s", baseName[:dot], count, baseName[dot:])
	}
	return fmt.Sprintf("pieces_jointes/%s_%d", baseName, count)
}

func sanitizeFileName(name string) string {
	trimmed := strings.TrimSpace(name)
	trimmed = strings.ReplaceAll(trimmed, "/", "_")
	trimmed = strings.ReplaceAll(trimmed, "\\", "_")
	trimmed = strings.ReplaceAll(trimmed, "..", "_")
	return trimmed
}

func toTemplateDTO(tpl domain.DocumentTemplate) DocumentTemplateDTO {
	return DocumentTemplateDTO{
		ID:           tpl.ID,
		Name:         tpl.Name,
		TemplateType: tpl.TemplateType,
		Format:       tpl.Format,
		R2Key:        tpl.R2Key,
		HTMLBody:     tpl.HTMLBody,
		CreatedAt:    tpl.CreatedAt,
		UpdatedAt:    tpl.UpdatedAt,
	}
}

func toStoredFileDTO(sf domain.StoredFile, downloadURL string) StoredFileDTO {
	return StoredFileDTO{
		ID:            sf.ID,
		LegalEntityID: sf.LegalEntityID,
		LeaseID:       sf.LeaseID,
		Category:      sf.Category,
		FileName:      sf.FileName,
		ContentType:   sf.ContentType,
		R2Key:         sf.R2Key,
		SizeBytes:     sf.SizeBytes,
		DownloadURL:   downloadURL,
		CreatedAt:     sf.CreatedAt,
	}
}

