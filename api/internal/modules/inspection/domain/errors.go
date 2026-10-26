package domain

import "errors"

// Domain errors for the inspection bounded context.
var (
	ErrValidation         = errors.New("inspection: validation error")
	ErrTemplateNotFound   = errors.New("inspection: template not found")
	ErrInspectionNotFound = errors.New("inspection: inspection not found")
	ErrIncidentNotFound   = errors.New("inspection: incident not found")
)
