import assert from 'node:assert/strict';
import { access, readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(__dirname, '..');

// Dynamically import statement-state.ts to verify module presence and runtime behavior
const statementStateMod = await import('../src/lib/api/statement-state.ts');
assert.equal(typeof statementStateMod.toggleStatementLine, 'function', 'toggleStatementLine must be exported');
assert.equal(typeof statementStateMod.toDisabledArray, 'function', 'toDisabledArray must be exported');

const requiredFiles = [
	'package.json',
	'svelte.config.js',
	'vite.config.ts',
	'tsconfig.json',
	'components.json',
	'src/app.d.ts',
	'src/app.css',
	'src/app.html',
	'src/lib/utils.ts',
	'src/lib/api/client.ts',
	'src/lib/api/statement-state.ts',
	'src/lib/api/statement-state.test.ts',
	'src/lib/components/ui/button/index.ts',
	'src/lib/components/ui/button/button.svelte',
	'src/lib/components/ui/card/index.ts',
	'src/lib/components/ui/card/card.svelte',
	'src/lib/components/ui/badge/index.ts',
	'src/lib/components/ui/badge/badge.svelte',
	'src/lib/components/ui/table/index.ts',
	'src/lib/components/ui/table/table.svelte',
	'src/lib/components/ui/switch/index.ts',
	'src/lib/components/ui/switch/switch.svelte',
	'src/lib/components/ui/tabs/index.ts',
	'src/lib/components/ui/tabs/tabs.svelte',
	'src/lib/components/ui/dialog/index.ts',
	'src/lib/components/ui/dialog/dialog.svelte',
	'src/lib/components/ui/input/index.ts',
	'src/lib/components/ui/input/input.svelte',
	'src/routes/+layout.server.ts',
	'src/routes/+layout.svelte',
	'src/routes/+page.svelte',
	'src/routes/login/+page.svelte',
	'src/routes/signup/+page.svelte',
	'src/routes/admin/users/+page.svelte',
	'src/routes/entities/[id]/+page.svelte',
	'src/routes/payments/negative-positions/+page.svelte',
	'src/routes/inspections/[id]/+page.svelte',
	'src/routes/accounting/[entityId]/statements/+page.svelte'
];

for (const relPath of requiredFiles) {
	const fullPath = path.join(webRoot, relPath);
	await access(fullPath);
	const content = await readFile(fullPath, 'utf8');
	assert.ok(content.trim().length > 0, `File ${relPath} must not be empty`);
}

// Verify API client exports and methods
const clientMod = await import('../src/lib/api/client.ts');
assert.equal(typeof clientMod.createApiClient, 'function', 'createApiClient must be exported');
const api = clientMod.createApiClient({ baseUrl: 'http://localhost:8080' });
const expectedClientMethods = [
	'signup',
	'login',
	'listPendingUsers',
	'approveUser',
	'rejectUser',
	'listEntities',
	'getEntitySummary',
	'createEntity',
	'createProperty',
	'listLeases',
	'generateRentCalls',
	'recordCashPayment',
	'importPaymentsCSV',
	'listNegativePositions',
	'assembleGLIDossier',
	'getInspection',
	'upsertInspectionItems',
	'generateInspectionPDF',
	'previewFinancialStatements',
	'generateAnnualAccountsPDF',
	'exportFEC'
];

for (const method of expectedClientMethods) {
	assert.equal(typeof api[method], 'function', `API client must implement ${method}()`);
}

// Verify key domain requirements across Svelte routes
const loginPage = await readFile(path.join(webRoot, 'src/routes/login/+page.svelte'), 'utf8');
assert.ok(
	loginPage.includes('Votre inscription est en attente de validation par le Super Administrateur'),
	'Login page must display French pending approval message on 403 ACCOUNT_NOT_APPROVED'
);

const statementsPage = await readFile(
	path.join(webRoot, 'src/routes/accounting/[entityId]/statements/+page.svelte'),
	'utf8'
);
assert.ok(statementsPage.includes('toggleStatementLine'), 'Statements page must use toggleStatementLine');
assert.ok(statementsPage.includes('Switch'), 'Statements page must render shadcn-svelte Switch');

const negPosPage = await readFile(
	path.join(webRoot, 'src/routes/payments/negative-positions/+page.svelte'),
	'utf8'
);
assert.ok(negPosPage.includes('Générer Dossier GLI'), 'Negative positions page must include 1-click GLI dossier action');

console.log(`✔ Verified ${requiredFiles.length} SvelteKit SSR & shadcn-svelte files and API client contracts.`);
