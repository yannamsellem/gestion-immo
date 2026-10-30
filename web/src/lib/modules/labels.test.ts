import { describe, expect, it } from 'vitest';
import {
	ACCOUNTING_CATEGORY_OPTIONS,
	ENTITY_TYPE_OPTIONS,
	formatEntityTypeLabel,
	formatLotTypeLabel,
	formatPropertyTypeLabel,
	LOT_TYPE_OPTIONS,
	PROPERTY_TYPE_OPTIONS
} from './portfolio/labels';
import {
	formatIndexationTypeLabel,
	formatLeaseTypeLabel,
	INDEXATION_TYPE_OPTIONS,
	LEASE_TYPE_OPTIONS
} from './lease/labels';
import {
	formatTemplateTypeLabel,
	TEMPLATE_FORMAT_OPTIONS,
	TEMPLATE_TYPE_OPTIONS
} from './document/labels';
import { formatPaymentMethodLabel, PAYMENT_METHOD_OPTIONS } from './payment/labels';
import {
	CONDITION_STATE_OPTIONS,
	formatConditionStateLabel,
	formatIncidentStatusLabel,
	formatIncidentStatusVariant,
	formatInspectionTypeLabel,
	INCIDENT_STATUS_OPTIONS,
	INSPECTION_TYPE_OPTIONS
} from './inspection/labels';

describe('domain enum French label dictionaries', () => {
	it('maps every OpenAPI EntityType to a distinct French label', () => {
		const entityTypes = [
			'SCI_IS',
			'SCI_IR',
			'SARL_IS',
			'FAMILY_SARL',
			'SAS',
			'SOLE_PROPRIETORSHIP',
			'JOINT_OWNERSHIP'
		] as const;

		const labels = entityTypes.map((code) => formatEntityTypeLabel(code));
		for (let i = 0; i < entityTypes.length; i += 1) {
			expect(labels[i]).not.toBe(entityTypes[i]);
			expect(labels[i].length).toBeGreaterThan(2);
		}
		expect(new Set(labels).size).toBe(entityTypes.length);
		expect(ENTITY_TYPE_OPTIONS.map((opt) => opt.value)).toEqual([...entityTypes]);
	});

	it('maps property, lot, and accounting category options to French labels', () => {
		expect(PROPERTY_TYPE_OPTIONS.length).toBeGreaterThanOrEqual(4);
		expect(LOT_TYPE_OPTIONS.length).toBeGreaterThanOrEqual(5);
		expect(ACCOUNTING_CATEGORY_OPTIONS.map((opt) => opt.value)).toContain('FIXED_ASSET');
		expect(ACCOUNTING_CATEGORY_OPTIONS.map((opt) => opt.value)).toContain('MERCHANDISE');

		for (const opt of PROPERTY_TYPE_OPTIONS) {
			expect(formatPropertyTypeLabel(opt.value)).toBe(opt.label);
		}
		for (const opt of LOT_TYPE_OPTIONS) {
			expect(formatLotTypeLabel(opt.value)).toBe(opt.label);
		}
	});

	it('maps every LeaseType and IndexationType to its French label', () => {
		const leaseTypes = [
			'RESIDENTIAL_UNFURNISHED',
			'RESIDENTIAL_FURNISHED',
			'COMMERCIAL_3_6_9',
			'PROFESSIONAL',
			'SHORT_TERM',
			'PARKING'
		] as const;

		for (const code of leaseTypes) {
			const label = formatLeaseTypeLabel(code);
			expect(label).not.toBe(code);
		}
		expect(LEASE_TYPE_OPTIONS.map((opt) => opt.value)).toEqual([...leaseTypes]);
		expect(INDEXATION_TYPE_OPTIONS.length).toBeGreaterThanOrEqual(3);
		for (const opt of INDEXATION_TYPE_OPTIONS) {
			expect(formatIndexationTypeLabel(opt.value)).toBe(opt.label);
		}
	});

	it('maps every TemplateType and TemplateFormat to its French label', () => {
		const templateTypes = [
			'LEASE',
			'RENT_RECEIPT',
			'FORMAL_NOTICE',
			'GLI_DECLARATION',
			'CUSTOM'
		] as const;

		for (const code of templateTypes) {
			expect(formatTemplateTypeLabel(code)).not.toBe(code);
		}
		expect(TEMPLATE_TYPE_OPTIONS.map((opt) => opt.value)).toEqual([...templateTypes]);
		expect(TEMPLATE_FORMAT_OPTIONS.map((opt) => opt.value)).toEqual(['HTML', 'DOCX']);
	});

	it('maps every PaymentMethod to its French label', () => {
		const methods = ['BANK_TRANSFER', 'DIRECT_DEBIT', 'CHECK', 'CASH'] as const;
		for (const method of methods) {
			expect(formatPaymentMethodLabel(method)).not.toBe(method);
		}
		expect(PAYMENT_METHOD_OPTIONS.map((opt) => opt.value)).toEqual([...methods]);
	});

	it('maps IncidentStatus, InspectionType, and ConditionState to French labels and badge variants', () => {
		const statuses = ['OPEN', 'INSURANCE_CLAIM', 'WORK_IN_PROGRESS', 'CLOSED'] as const;
		for (const status of statuses) {
			expect(formatIncidentStatusLabel(status)).not.toBe(status);
		}
		expect(INCIDENT_STATUS_OPTIONS.map((opt) => opt.value)).toEqual([...statuses]);
		expect(formatIncidentStatusVariant('OPEN')).toBe('destructive');
		expect(formatIncidentStatusVariant('CLOSED')).toBe('outline');

		for (const inspType of ['MOVE_IN', 'MOVE_OUT'] as const) {
			expect(formatInspectionTypeLabel(inspType)).not.toBe(inspType);
		}
		expect(INSPECTION_TYPE_OPTIONS.map((opt) => opt.value)).toEqual(['MOVE_IN', 'MOVE_OUT']);

		const conditions = ['NEW', 'GOOD', 'FAIR_WEAR', 'POOR', 'OUT_OF_SERVICE'] as const;
		for (const condition of conditions) {
			expect(formatConditionStateLabel(condition)).not.toBe(condition);
		}
		expect(CONDITION_STATE_OPTIONS.map((opt) => opt.value)).toEqual([...conditions]);
	});
});
