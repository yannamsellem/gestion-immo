export type PaymentMethod = 'BANK_TRANSFER' | 'CHECK' | 'DIRECT_DEBIT' | 'CASH' | string;
export type CashEntryType = 'CASH_RECEIPT' | 'BANK_DEPOSIT' | string;

export interface RecordCashPaymentCmd {
	lease_id: string;
	amount_cents: number;
	label: string;
	receipt_number?: string;
	payment_date?: string;
}

export interface RecordBankPaymentCmd {
	lease_id: string;
	amount_cents: number;
	method?: PaymentMethod;
	label: string;
	external_ref?: string;
	payment_date?: string;
}

export interface RecordCashDepositCmd {
	legal_entity_id: string;
	amount_cents: number;
	label: string;
	entry_date?: string;
}

export interface RentAllocationDTO {
	rent_call_id: string;
	period_month: string;
	allocated_cents: number;
	remaining_due_cents: number;
}

export interface PaymentReceiptDTO {
	id: string;
	legal_entity_id: string;
	lease_id: string;
	tenant_id: string;
	amount_cents: number;
	unallocated_cents: number;
	method: PaymentMethod;
	label: string;
	external_ref?: string;
	receipt_number?: string;
	payment_date: string;
	allocations: RentAllocationDTO[];
	cash_journal_entry_id?: string;
	created_at: string;
}

export interface CashJournalEntryDTO {
	id: string;
	legal_entity_id: string;
	payment_id?: string;
	account_code: string;
	counterpart_account_code: string;
	entry_type: CashEntryType;
	label: string;
	debit_cents: number;
	credit_cents: number;
	receipt_number?: string;
	entry_date: string;
	created_at: string;
}

export interface BatchImportResultDTO {
	entity_id: string;
	imported_count: number;
	total_imported_cents: number;
	payments: PaymentReceiptDTO[];
}

export interface NegativePositionDTO {
	lease_id: string;
	legal_entity_id: string;
	lot_id: string;
	tenant_id: string;
	tenant_name: string;
	total_unpaid_cents: number;
	unpaid_months: string[];
	oldest_due_date: string;
	gli_policy_number: string;
	gli_eligible: boolean;
}
