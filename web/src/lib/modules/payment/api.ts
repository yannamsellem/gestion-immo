import type { HttpClient } from '#lib/core/api/http-client';
import type {
	BatchImportResultDTO,
	CashJournalEntryDTO,
	NegativePositionDTO,
	PaymentReceiptDTO,
	RecordBankPaymentCmd,
	RecordCashDepositCmd,
	RecordCashPaymentCmd
} from './types';

export class PaymentApiService {
	constructor(private readonly http: HttpClient) {}

	recordCashPayment(cmd: RecordCashPaymentCmd): Promise<PaymentReceiptDTO> {
		return this.http.requestJson<PaymentReceiptDTO>('/api/v1/payments/cash', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	recordBankPayment(cmd: RecordBankPaymentCmd): Promise<PaymentReceiptDTO> {
		return this.http.requestJson<PaymentReceiptDTO>('/api/v1/payments/bank', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	recordCashDeposit(cmd: RecordCashDepositCmd): Promise<CashJournalEntryDTO> {
		return this.http.requestJson<CashJournalEntryDTO>('/api/v1/payments/cash-deposit', {
			method: 'POST',
			body: JSON.stringify(cmd)
		});
	}

	importPaymentsCsv(entityId: string, csvContent: string): Promise<BatchImportResultDTO> {
		return this.http.requestJson<BatchImportResultDTO>('/api/v1/payments/import-csv', {
			method: 'POST',
			body: JSON.stringify({
				entity_id: entityId,
				csv_content: csvContent
			})
		});
	}

	listNegativePositions(entityId: string): Promise<NegativePositionDTO[]> {
		return this.http.requestJson<NegativePositionDTO[]>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/negative-positions`
		);
	}

	listCashJournal(entityId: string): Promise<CashJournalEntryDTO[]> {
		return this.http.requestJson<CashJournalEntryDTO[]>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/cash-journal`
		);
	}
}
