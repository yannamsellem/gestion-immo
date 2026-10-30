import type { PaymentMethod } from './types';

export const PAYMENT_METHOD_OPTIONS: ReadonlyArray<{ value: PaymentMethod; label: string }> = [
	{ value: 'BANK_TRANSFER', label: 'Virement SEPA (512)' },
	{ value: 'DIRECT_DEBIT', label: 'Prélèvement automatique (512)' },
	{ value: 'CHECK', label: 'Chèque bancaire (512)' },
	{ value: 'CASH', label: 'Espèces — Livre de Caisse (530)' }
];

const PAYMENT_METHOD_LABELS = new Map(PAYMENT_METHOD_OPTIONS.map((opt) => [opt.value, opt.label]));

export function formatPaymentMethodLabel(value: string): string {
	return PAYMENT_METHOD_LABELS.get(value) ?? value;
}
