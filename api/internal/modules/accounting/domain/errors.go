package domain

import "errors"

var (
	// ErrValidation indicates an invalid domain command or accounting invariant violation.
	ErrValidation = errors.New("accounting: validation error")
	// ErrUnbalancedJournalEntry indicates that total debits do not equal total credits.
	ErrUnbalancedJournalEntry = errors.New("accounting: journal entry is unbalanced (total debit != total credit)")
	// ErrJournalEntryNotFound indicates a requested journal entry does not exist.
	ErrJournalEntryNotFound = errors.New("accounting: journal entry not found")
	// ErrReportNotFound indicates a requested annual accounts report does not exist.
	ErrReportNotFound = errors.New("accounting: annual accounts report not found")
)
