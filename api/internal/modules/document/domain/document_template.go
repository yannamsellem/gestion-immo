package domain

import (
	"context"
	"time"
)

// TemplateType classifies a customizable document template.
type TemplateType string

const (
	TemplateTypeLease          TemplateType = "LEASE"
	TemplateTypeRentReceipt    TemplateType = "RENT_RECEIPT"
	TemplateTypeFormalNotice   TemplateType = "FORMAL_NOTICE"
	TemplateTypeGLIDeclaration TemplateType = "GLI_DECLARATION"
	TemplateTypeCustom         TemplateType = "CUSTOM"
)

// IsValid reports whether the TemplateType is supported.
func (t TemplateType) IsValid() bool {
	switch t {
	case TemplateTypeLease, TemplateTypeRentReceipt, TemplateTypeFormalNotice, TemplateTypeGLIDeclaration, TemplateTypeCustom:
		return true
	default:
		return false
	}
}

// TemplateFormat identifies the underlying document rendering format (.docx vs HTML/PDF).
type TemplateFormat string

const (
	FormatDocx TemplateFormat = "DOCX"
	FormatHTML TemplateFormat = "HTML"
)

// IsValid reports whether the TemplateFormat is supported.
func (f TemplateFormat) IsValid() bool {
	switch f {
	case FormatDocx, FormatHTML:
		return true
	default:
		return false
	}
}

// DocumentTemplate represents the Aggregate Root for a reusable .docx or HTML/PDF template stored in PostgreSQL/R2.
type DocumentTemplate struct {
	ID           string
	Name         string
	TemplateType TemplateType
	Format       TemplateFormat
	R2Key        string
	HTMLBody     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DocumentTemplateRepository defines the persistence contract for the DocumentTemplate aggregate root.
type DocumentTemplateRepository interface {
	SaveTemplate(ctx context.Context, tpl DocumentTemplate) error
	FindTemplateByID(ctx context.Context, id string) (*DocumentTemplate, error)
	ListTemplates(ctx context.Context) ([]DocumentTemplate, error)
}
