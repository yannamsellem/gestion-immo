package domain

import (
	"context"
	"time"
)

// DefaultRoomElements provides the standard element checklist per room when a template does not specify custom elements.
var DefaultRoomElements = []string{
	"Sols",
	"Murs",
	"Plafond",
	"Menuiseries / Fenêtres",
	"Électricité / Éclairage",
	"Équipements",
}

// InspectionTemplate represents the Aggregate Root for a reusable room/element grid layout
// for a property type (e.g., T2, Studio, Local Commercial).
type InspectionTemplate struct {
	ID              string
	Name            string
	Rooms           []string
	DefaultElements []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// InspectionTemplateRepository defines the persistence contract for the InspectionTemplate aggregate root.
type InspectionTemplateRepository interface {
	SaveTemplate(ctx context.Context, tpl InspectionTemplate) error
	FindTemplateByID(ctx context.Context, id string) (*InspectionTemplate, error)
	ListTemplates(ctx context.Context) ([]InspectionTemplate, error)
}
