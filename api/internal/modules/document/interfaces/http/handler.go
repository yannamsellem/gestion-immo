package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/yannamsellem/gestion-immo/internal/modules/document/application"
	"github.com/yannamsellem/gestion-immo/internal/modules/document/domain"
	leasedomain "github.com/yannamsellem/gestion-immo/internal/modules/lease/domain"
	portfoliodomain "github.com/yannamsellem/gestion-immo/internal/modules/portfolio/domain"
	"github.com/yannamsellem/gestion-immo/internal/platform/httpx"
)

// Handler exposes REST HTTP endpoints for the document bounded context.
type Handler struct {
	svc *application.Service
}

// NewHandler constructs a new document HTTP Handler via constructor injection.
func NewHandler(svc *application.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterTemplateBody represents the request body for registering a document template.
type RegisterTemplateBody struct {
	Name         string `json:"name"`
	TemplateType string `json:"template_type"`
	Format       string `json:"format"`
	DocxBase64   string `json:"docx_base64,omitempty"`
	HTMLBody     string `json:"html_body,omitempty"`
}

type registerTemplateInput struct {
	Body RegisterTemplateBody
}

type documentTemplateOutput struct {
	Body application.DocumentTemplateDTO
}

type documentTemplateListOutput struct {
	Body []application.DocumentTemplateDTO
}

type generateFromTemplateInput struct {
	Body application.GenerateDocCmd
}

type storedFileOutput struct {
	Body application.StoredFileDTO
}

type storedFileListOutput struct {
	Body []application.StoredFileDTO
}

// UploadDocumentBody represents the JSON request body for uploading a base64-encoded document to R2.
type UploadDocumentBody struct {
	LegalEntityID string `json:"legal_entity_id,omitempty"`
	LeaseID       string `json:"lease_id,omitempty"`
	Category      string `json:"category"`
	FileName      string `json:"file_name"`
	ContentType   string `json:"content_type"`
	ContentBase64 string `json:"content_base64"`
}

type attachFileInput struct {
	ContentType string `header:"Content-Type"`
	RawBody     []byte
}

// AssembleGLIDossierByLeaseBody represents the optional body for assembling a GLI dossier by lease ID.
type AssembleGLIDossierByLeaseBody struct {
	TemplateID string `json:"template_id,omitempty"`
}

type assembleGLIDossierByLeaseInput struct {
	ID         string `path:"id"`
	TemplateID string `query:"template_id" required:"false"`
	RawBody    []byte
}

// AssembleGLIDossierBody represents the request body for assembling a GLI claim dossier.
type AssembleGLIDossierBody struct {
	LeaseID    string `json:"lease_id"`
	TemplateID string `json:"template_id,omitempty"`
}

type assembleGLIDossierBodyInput struct {
	Body AssembleGLIDossierBody
}

type gliDossierOutput struct {
	Body application.GLIDossierArchiveDTO
}

type leaseIDInput struct {
	ID string `path:"id"`
}

// RegisterRoutes mounts all document and GLI dossier REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.RegisterHumaRoutes(httpx.NewHumaAPI(mux))
}

// RegisterHumaRoutes registers all document and GLI dossier REST endpoints onto a Huma API.
func (h *Handler) RegisterHumaRoutes(api huma.API) {
	bearerSecurity := []map[string][]string{{"bearerAuth": {}}}

	huma.Register(api, huma.Operation{
		OperationID:   "registerDocumentTemplate",
		Method:        http.MethodPost,
		Path:          "/api/v1/document-templates",
		Summary:       "Register a customizable .docx or HTML document template",
		DefaultStatus: http.StatusCreated,
		MaxBodyBytes:  httpx.MaxUploadBodyBytes,
		Security:      bearerSecurity,
		Tags:          []string{"Document"},
	}, h.handleRegisterTemplate)

	huma.Register(api, huma.Operation{
		OperationID:   "listDocumentTemplates",
		Method:        http.MethodGet,
		Path:          "/api/v1/document-templates",
		Summary:       "List registered document templates",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Document"},
	}, h.handleListTemplates)

	huma.Register(api, huma.Operation{
		OperationID:   "generateDocument",
		Method:        http.MethodPost,
		Path:          "/api/v1/documents/generate",
		Summary:       "Generate a document (.docx or PDF) from a template",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Document"},
	}, h.handleGenerateFromTemplate)

	uploadSchema := huma.SchemaFromType(api.OpenAPI().Components.Schemas, reflect.TypeFor[UploadDocumentBody]())
	huma.Register(api, huma.Operation{
		OperationID:      "uploadDocument",
		Method:           http.MethodPost,
		Path:             "/api/v1/documents/upload",
		Summary:          "Upload and attach a file to Cloudflare R2",
		DefaultStatus:    http.StatusCreated,
		MaxBodyBytes:     httpx.MaxUploadBodyBytes,
		SkipValidateBody: true,
		Security:         bearerSecurity,
		Tags:             []string{"Document"},
	}, h.handleAttachFile)
	if pathItem := api.OpenAPI().Paths["/api/v1/documents/upload"]; pathItem != nil && pathItem.Post != nil {
		pathItem.Post.RequestBody = &huma.RequestBody{
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: uploadSchema},
			},
		}
	}

	gliByLeaseSchema := huma.SchemaFromType(api.OpenAPI().Components.Schemas, reflect.TypeFor[AssembleGLIDossierByLeaseBody]())
	huma.Register(api, huma.Operation{
		OperationID:      "assembleGliDossierByLease",
		Method:           http.MethodPost,
		Path:             "/api/v1/leases/{id}/gli-dossier",
		Summary:          "Assemble a 1-click GLI claim ZIP dossier for a lease",
		DefaultStatus:    http.StatusCreated,
		SkipValidateBody: true,
		Security:         bearerSecurity,
		Tags:             []string{"Document"},
	}, h.handleAssembleGLIDossier)
	if pathItem := api.OpenAPI().Paths["/api/v1/leases/{id}/gli-dossier"]; pathItem != nil && pathItem.Post != nil {
		pathItem.Post.RequestBody = &huma.RequestBody{
			Required: false,
			Content: map[string]*huma.MediaType{
				"application/json": {Schema: gliByLeaseSchema},
			},
		}
	}

	huma.Register(api, huma.Operation{
		OperationID:   "assembleGliDossier",
		Method:        http.MethodPost,
		Path:          "/api/v1/documents/gli-dossier",
		Summary:       "Assemble a 1-click GLI claim ZIP dossier",
		DefaultStatus: http.StatusCreated,
		Security:      bearerSecurity,
		Tags:          []string{"Document"},
	}, h.handleAssembleGLIDossierBody)

	huma.Register(api, huma.Operation{
		OperationID:   "listLeaseDocuments",
		Method:        http.MethodGet,
		Path:          "/api/v1/leases/{id}/documents",
		Summary:       "List stored documents and attachments for a lease",
		DefaultStatus: http.StatusOK,
		Security:      bearerSecurity,
		Tags:          []string{"Document"},
	}, h.handleListLeaseDocuments)
}

func (h *Handler) handleRegisterTemplate(ctx context.Context, in *registerTemplateInput) (*documentTemplateOutput, error) {
	var docxBytes []byte
	if strings.TrimSpace(in.Body.DocxBase64) != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.Body.DocxBase64))
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid docx_base64 payload")
		}
		docxBytes = decoded
	}

	dto, err := h.svc.RegisterTemplate(ctx, application.RegisterTemplateCmd{
		Name:         in.Body.Name,
		TemplateType: in.Body.TemplateType,
		Format:       in.Body.Format,
		DocxBytes:    docxBytes,
		HTMLBody:     in.Body.HTMLBody,
	})
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &documentTemplateOutput{Body: dto}, nil
}

func (h *Handler) handleListTemplates(ctx context.Context, _ *struct{}) (*documentTemplateListOutput, error) {
	list, err := h.svc.ListTemplates(ctx)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if list == nil {
		list = []application.DocumentTemplateDTO{}
	}
	return &documentTemplateListOutput{Body: list}, nil
}

func (h *Handler) handleGenerateFromTemplate(ctx context.Context, in *generateFromTemplateInput) (*storedFileOutput, error) {
	dto, err := h.svc.GenerateFromTemplate(ctx, in.Body)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &storedFileOutput{Body: dto}, nil
}

func (h *Handler) handleAttachFile(ctx context.Context, in *attachFileInput) (*storedFileOutput, error) {
	contentType := strings.ToLower(in.ContentType)
	var cmd application.AttachFileCmd

	if strings.Contains(contentType, "multipart/form-data") {
		_, params, err := mime.ParseMediaType(in.ContentType)
		if err != nil || params["boundary"] == "" {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid multipart form")
		}
		mr := multipart.NewReader(bytes.NewReader(in.RawBody), params["boundary"])
		form, err := mr.ReadForm(20 << 20)
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid multipart form")
		}
		defer func() { _ = form.RemoveAll() }()
		if len(form.Value["legal_entity_id"]) > 0 {
			cmd.LegalEntityID = strings.TrimSpace(form.Value["legal_entity_id"][0])
		}
		if len(form.Value["lease_id"]) > 0 {
			cmd.LeaseID = strings.TrimSpace(form.Value["lease_id"][0])
		}
		if len(form.Value["category"]) > 0 {
			cmd.Category = strings.TrimSpace(form.Value["category"][0])
		}
		files := form.File["file"]
		if len(files) == 0 {
			return nil, httpx.StatusError(http.StatusBadRequest, "missing file field")
		}
		file, err := files[0].Open()
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "failed to read uploaded file")
		}
		defer file.Close()
		raw, err := io.ReadAll(file)
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "failed to read uploaded file")
		}
		cmd.FileName = files[0].Filename
		cmd.ContentType = files[0].Header.Get("Content-Type")
		cmd.Content = raw
	} else {
		var body UploadDocumentBody
		if err := json.Unmarshal(in.RawBody, &body); err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, err.Error())
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(body.ContentBase64))
		if err != nil {
			return nil, httpx.StatusError(http.StatusBadRequest, "invalid content_base64")
		}
		cmd = application.AttachFileCmd{
			LegalEntityID: body.LegalEntityID,
			LeaseID:       body.LeaseID,
			Category:      body.Category,
			FileName:      body.FileName,
			ContentType:   body.ContentType,
			Content:       raw,
		}
	}

	dto, err := h.svc.AttachFileToR2(ctx, cmd)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &storedFileOutput{Body: dto}, nil
}

func (h *Handler) handleAssembleGLIDossier(ctx context.Context, in *assembleGLIDossierByLeaseInput) (*gliDossierOutput, error) {
	templateID := strings.TrimSpace(in.TemplateID)
	if trimmed := bytes.TrimSpace(in.RawBody); len(trimmed) > 0 {
		var body AssembleGLIDossierByLeaseBody
		if err := json.Unmarshal(trimmed, &body); err == nil && strings.TrimSpace(body.TemplateID) != "" {
			templateID = strings.TrimSpace(body.TemplateID)
		}
	}

	dto, err := h.svc.AssembleGLIDossier(ctx, in.ID, templateID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &gliDossierOutput{Body: dto}, nil
}

func (h *Handler) handleAssembleGLIDossierBody(ctx context.Context, in *assembleGLIDossierBodyInput) (*gliDossierOutput, error) {
	dto, err := h.svc.AssembleGLIDossier(ctx, in.Body.LeaseID, in.Body.TemplateID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	return &gliDossierOutput{Body: dto}, nil
}

func (h *Handler) handleListLeaseDocuments(ctx context.Context, in *leaseIDInput) (*storedFileListOutput, error) {
	files, err := h.svc.ListStoredFilesByLease(ctx, in.ID)
	if err != nil {
		return nil, mapDomainError(err)
	}
	if files == nil {
		files = []application.StoredFileDTO{}
	}
	return &storedFileListOutput{Body: files}, nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrNoUnpaidRentForGLI),
		errors.Is(err, leasedomain.ErrValidation),
		errors.Is(err, portfoliodomain.ErrValidation):
		return httpx.StatusError(http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, domain.ErrTemplateNotFound),
		errors.Is(err, domain.ErrStoredFileNotFound),
		errors.Is(err, domain.ErrGLIDossierNotFound),
		errors.Is(err, leasedomain.ErrLeaseNotFound),
		errors.Is(err, leasedomain.ErrTenantNotFound),
		errors.Is(err, portfoliodomain.ErrLegalEntityNotFound):
		return httpx.StatusError(http.StatusNotFound, err.Error())
	default:
		return httpx.StatusError(http.StatusInternalServerError, "internal document error")
	}
}
