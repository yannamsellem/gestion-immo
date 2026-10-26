package http

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"

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

// RegisterRoutes mounts all document and GLI dossier REST endpoints on the Go 1.22+ ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/document-templates", h.handleRegisterTemplate)
	mux.HandleFunc("GET /api/v1/document-templates", h.handleListTemplates)
	mux.HandleFunc("POST /api/v1/documents/generate", h.handleGenerateFromTemplate)
	mux.HandleFunc("POST /api/v1/documents/upload", h.handleAttachFile)
	mux.HandleFunc("POST /api/v1/leases/{id}/gli-dossier", h.handleAssembleGLIDossier)
	mux.HandleFunc("POST /api/v1/documents/gli-dossier", h.handleAssembleGLIDossierBody)
	mux.HandleFunc("GET /api/v1/leases/{id}/documents", h.handleListLeaseDocuments)
}

func (h *Handler) handleRegisterTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name          string `json:"name"`
		TemplateType  string `json:"template_type"`
		Format        string `json:"format"`
		DocxBase64    string `json:"docx_base64,omitempty"`
		HTMLBody      string `json:"html_body,omitempty"`
	}
	if r.Body != nil {
		r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxUploadBodyBytes)
	}
	if err := httpx.DecodeJSONWithLimit(r, &req, httpx.MaxUploadBodyBytes); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	var docxBytes []byte
	if strings.TrimSpace(req.DocxBase64) != "" {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.DocxBase64))
		if err != nil {
			httpx.BadRequest(w, "invalid docx_base64 payload")
			return
		}
		docxBytes = decoded
	}

	dto, err := h.svc.RegisterTemplate(r.Context(), application.RegisterTemplateCmd{
		Name:         req.Name,
		TemplateType: req.TemplateType,
		Format:       req.Format,
		DocxBytes:    docxBytes,
		HTMLBody:     req.HTMLBody,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListTemplates(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if list == nil {
		list = []application.DocumentTemplateDTO{}
	}
	httpx.OK(w, list)
}

func (h *Handler) handleGenerateFromTemplate(w http.ResponseWriter, r *http.Request) {
	var cmd application.GenerateDocCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.GenerateFromTemplate(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleAttachFile(w http.ResponseWriter, r *http.Request) {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	var cmd application.AttachFileCmd

	if strings.Contains(contentType, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxUploadBodyBytes)
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			httpx.BadRequest(w, "invalid multipart form")
			return
		}
		cmd.LegalEntityID = strings.TrimSpace(r.FormValue("legal_entity_id"))
		cmd.LeaseID = strings.TrimSpace(r.FormValue("lease_id"))
		cmd.Category = strings.TrimSpace(r.FormValue("category"))

		file, hdr, err := r.FormFile("file")
		if err != nil {
			httpx.BadRequest(w, "missing file field")
			return
		}
		defer file.Close()
		raw, err := io.ReadAll(file)
		if err != nil {
			httpx.BadRequest(w, "failed to read uploaded file")
			return
		}
		cmd.FileName = hdr.Filename
		cmd.ContentType = hdr.Header.Get("Content-Type")
		cmd.Content = raw
	} else {
		var body struct {
			LegalEntityID string `json:"legal_entity_id"`
			LeaseID       string `json:"lease_id"`
			Category      string `json:"category"`
			FileName      string `json:"file_name"`
			ContentType   string `json:"content_type"`
			ContentBase64 string `json:"content_base64"`
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxUploadBodyBytes)
		}
		if err := httpx.DecodeJSONWithLimit(r, &body, httpx.MaxUploadBodyBytes); err != nil {
			httpx.BadRequest(w, err.Error())
			return
		}
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(body.ContentBase64))
		if err != nil {
			httpx.BadRequest(w, "invalid content_base64")
			return
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

	dto, err := h.svc.AttachFileToR2(r.Context(), cmd)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleAssembleGLIDossier(w http.ResponseWriter, r *http.Request) {
	leaseID := r.PathValue("id")
	templateID := strings.TrimSpace(r.URL.Query().Get("template_id"))
	if r.Body != nil && r.ContentLength != 0 {
		var body struct {
			TemplateID string `json:"template_id"`
		}
		if err := httpx.DecodeJSON(r, &body); err == nil && strings.TrimSpace(body.TemplateID) != "" {
			templateID = strings.TrimSpace(body.TemplateID)
		}
	}

	dto, err := h.svc.AssembleGLIDossier(r.Context(), leaseID, templateID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleAssembleGLIDossierBody(w http.ResponseWriter, r *http.Request) {
	var body struct {
		LeaseID    string `json:"lease_id"`
		TemplateID string `json:"template_id"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.BadRequest(w, err.Error())
		return
	}

	dto, err := h.svc.AssembleGLIDossier(r.Context(), body.LeaseID, body.TemplateID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	httpx.Created(w, dto)
}

func (h *Handler) handleListLeaseDocuments(w http.ResponseWriter, r *http.Request) {
	leaseID := r.PathValue("id")
	files, err := h.svc.ListStoredFilesByLease(r.Context(), leaseID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if files == nil {
		files = []application.StoredFileDTO{}
	}
	httpx.OK(w, files)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation),
		errors.Is(err, domain.ErrNoUnpaidRentForGLI),
		errors.Is(err, leasedomain.ErrValidation),
		errors.Is(err, portfoliodomain.ErrValidation):
		httpx.UnprocessableEntity(w, err.Error())
	case errors.Is(err, domain.ErrTemplateNotFound),
		errors.Is(err, domain.ErrStoredFileNotFound),
		errors.Is(err, domain.ErrGLIDossierNotFound),
		errors.Is(err, leasedomain.ErrLeaseNotFound),
		errors.Is(err, leasedomain.ErrTenantNotFound),
		errors.Is(err, portfoliodomain.ErrLegalEntityNotFound):
		httpx.NotFound(w, err.Error())
	default:
		httpx.LogInternalError(w, "internal document error", err)
	}
}
