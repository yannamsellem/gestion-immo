import type { components } from '#lib/core/api/openapi';

export type PaymentMethod = 'BANK_TRANSFER' | 'CHECK' | 'DIRECT_DEBIT' | 'CASH' | string;
export type CashEntryType = 'CASH_RECEIPT' | 'BANK_DEPOSIT' | string;

export type RecordCashPaymentCmd = components['schemas']['RecordCashPaymentCmd'];
export type RecordBankPaymentCmd = components['schemas']['RecordBankPaymentCmd'];
export type RecordCashDepositCmd = components['schemas']['RecordCashDepositCmd'];
export type RentAllocationDTO = components['schemas']['RentAllocation'];
export type PaymentReceiptDTO = components['schemas']['PaymentReceiptDTO'];
export type CashJournalEntryDTO = components['schemas']['CashJournalEntryDTO'];
export type BatchImportResultDTO = components['schemas']['BatchImportResultDTO'];
export type NegativePositionDTO = components['schemas']['NegativePositionDTO'];
