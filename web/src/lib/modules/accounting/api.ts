import type { HttpClient } from '#lib/core/api/http-client';
import type {
	AccountBalanceLineDTO,
	AnnualAccountsReportDTO,
	ComputedFinancialStatementsDTO,
	JournalEntryDTO,
	RecordJournalEntryCmd
} from './types';

export class AccountingApiService {
	constructor(private readonly http: HttpClient) {}

	listJournalEntries(entityId: string, fiscalYear: number): Promise<JournalEntryDTO[]> {
		return this.http.requestJson<JournalEntryDTO[]>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/journal-entries?fiscal_year=${fiscalYear}`
		);
	}

	recordJournalEntry(entityId: string, cmd: RecordJournalEntryCmd): Promise<JournalEntryDTO> {
		return this.http.requestJson<JournalEntryDTO>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/journal-entries`,
			{
				method: 'POST',
				body: JSON.stringify({ ...cmd, legal_entity_id: entityId })
			}
		);
	}

	upsertBalanceLines(
		entityId: string,
		fiscalYear: number,
		lines: AccountBalanceLineDTO[]
	): Promise<void> {
		return this.http.requestJson<void>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/statements/lines`,
			{
				method: 'PUT',
				body: JSON.stringify({
					fiscal_year: fiscalYear,
					lines
				})
			}
		);
	}

	previewStatements(
		entityId: string,
		fiscalYear: number,
		disabledLineIds: string[]
	): Promise<ComputedFinancialStatementsDTO> {
		return this.http.requestJson<ComputedFinancialStatementsDTO>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/statements/preview`,
			{
				method: 'POST',
				body: JSON.stringify({
					fiscal_year: fiscalYear,
					disabled_line_ids: disabledLineIds
				})
			}
		);
	}

	generateAnnualAccountsPdf(
		entityId: string,
		fiscalYear: number,
		disabledLineIds: string[]
	): Promise<AnnualAccountsReportDTO> {
		return this.http.requestJson<AnnualAccountsReportDTO>(
			`/api/v1/entities/${encodeURIComponent(entityId)}/statements/pdf`,
			{
				method: 'POST',
				body: JSON.stringify({
					fiscal_year: fiscalYear,
					disabled_line_ids: disabledLineIds
				})
			}
		);
	}

	exportFecTsv(entityId: string, fiscalYear: number): Promise<string> {
		return this.http.requestText(
			`/api/v1/entities/${encodeURIComponent(entityId)}/fec?fiscal_year=${fiscalYear}`
		);
	}
}
