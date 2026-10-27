CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE RESTRICT,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    unallocated_cents BIGINT NOT NULL DEFAULT 0 CHECK (unallocated_cents >= 0),
    method TEXT NOT NULL CHECK (method IN ('CASH', 'BANK_TRANSFER', 'CHECK', 'DIRECT_DEBIT')),
    label TEXT NOT NULL DEFAULT '',
    external_ref TEXT NOT NULL DEFAULT '',
    receipt_number TEXT NOT NULL DEFAULT '',
    payment_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_legal_entity_id ON payments(legal_entity_id);
CREATE INDEX IF NOT EXISTS idx_payments_lease_id ON payments(lease_id);
CREATE INDEX IF NOT EXISTS idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_payment_date ON payments(payment_date);

CREATE TABLE IF NOT EXISTS rent_allocations (
    id TEXT PRIMARY KEY,
    payment_id TEXT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    rent_call_id TEXT NOT NULL REFERENCES rent_calls(id) ON DELETE RESTRICT,
    period_month TEXT NOT NULL,
    applied_cents BIGINT NOT NULL CHECK (applied_cents > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rent_allocations_payment_id ON rent_allocations(payment_id);
CREATE INDEX IF NOT EXISTS idx_rent_allocations_rent_call_id ON rent_allocations(rent_call_id);

CREATE TABLE IF NOT EXISTS cash_journal_entries (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    payment_id TEXT REFERENCES payments(id) ON DELETE SET NULL,
    account_code TEXT NOT NULL DEFAULT '530',
    counterpart_account_code TEXT NOT NULL DEFAULT '411',
    entry_type TEXT NOT NULL CHECK (entry_type IN ('CASH_RECEIPT', 'BANK_DEPOSIT', 'CASH_EXPENSE')),
    label TEXT NOT NULL,
    debit_cents BIGINT NOT NULL DEFAULT 0 CHECK (debit_cents >= 0),
    credit_cents BIGINT NOT NULL DEFAULT 0 CHECK (credit_cents >= 0),
    receipt_number TEXT NOT NULL DEFAULT '',
    entry_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cash_journal_legal_entity_id ON cash_journal_entries(legal_entity_id);
CREATE INDEX IF NOT EXISTS idx_cash_journal_entry_date ON cash_journal_entries(entry_date);

---- create above / drop below ----

DROP TABLE IF EXISTS cash_journal_entries;
DROP TABLE IF EXISTS rent_allocations;
DROP TABLE IF EXISTS payments;
