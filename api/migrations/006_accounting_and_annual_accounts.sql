CREATE TABLE IF NOT EXISTS pcg_accounts (
    code TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    account_class INT NOT NULL CHECK (account_class BETWEEN 1 AND 8),
    section TEXT NOT NULL DEFAULT '',
    cerfa_code TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS journal_entries (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    fiscal_year INT NOT NULL CHECK (fiscal_year >= 1900),
    journal_code TEXT NOT NULL DEFAULT 'OD',
    journal_label TEXT NOT NULL DEFAULT 'General Journal',
    entry_number TEXT NOT NULL,
    entry_date DATE NOT NULL,
    document_ref TEXT NOT NULL DEFAULT '',
    document_date DATE NOT NULL,
    label TEXT NOT NULL,
    validation_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_journal_entries_entity_year ON journal_entries(legal_entity_id, fiscal_year);
CREATE INDEX IF NOT EXISTS idx_journal_entries_entry_date ON journal_entries(entry_date);

CREATE TABLE IF NOT EXISTS journal_lines (
    id TEXT PRIMARY KEY,
    journal_entry_id TEXT NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    fiscal_year INT NOT NULL,
    line_id TEXT NOT NULL DEFAULT '',
    account_code TEXT NOT NULL,
    account_label TEXT NOT NULL DEFAULT '',
    sub_account_code TEXT NOT NULL DEFAULT '',
    sub_account_label TEXT NOT NULL DEFAULT '',
    section TEXT NOT NULL DEFAULT '',
    label TEXT NOT NULL DEFAULT '',
    debit_cents BIGINT NOT NULL DEFAULT 0 CHECK (debit_cents >= 0),
    credit_cents BIGINT NOT NULL DEFAULT 0 CHECK (credit_cents >= 0),
    net_n_minus_1_cents BIGINT NOT NULL DEFAULT 0,
    reconciliation_code TEXT NOT NULL DEFAULT '',
    reconciliation_date TEXT NOT NULL DEFAULT '',
    foreign_currency_amount_cents BIGINT NOT NULL DEFAULT 0,
    currency_code TEXT NOT NULL DEFAULT 'EUR',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_journal_lines_entry_id ON journal_lines(journal_entry_id);
CREATE INDEX IF NOT EXISTS idx_journal_lines_entity_year ON journal_lines(legal_entity_id, fiscal_year);
CREATE INDEX IF NOT EXISTS idx_journal_lines_account_code ON journal_lines(account_code);

CREATE TABLE IF NOT EXISTS account_balance_lines (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    fiscal_year INT NOT NULL CHECK (fiscal_year >= 1900),
    line_id TEXT NOT NULL,
    account_code TEXT NOT NULL,
    label TEXT NOT NULL,
    section TEXT NOT NULL,
    cerfa_code TEXT NOT NULL DEFAULT '',
    gross_n_cents BIGINT NOT NULL DEFAULT 0,
    depreciation_n_cents BIGINT NOT NULL DEFAULT 0,
    net_n_cents BIGINT NOT NULL DEFAULT 0,
    net_n_minus_1_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (legal_entity_id, fiscal_year, line_id)
);

CREATE INDEX IF NOT EXISTS idx_account_balance_lines_entity_year ON account_balance_lines(legal_entity_id, fiscal_year);

CREATE TABLE IF NOT EXISTS annual_account_reports (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    fiscal_year INT NOT NULL CHECK (fiscal_year >= 1900),
    disabled_line_ids TEXT[] NOT NULL DEFAULT '{}',
    net_income_n_cents BIGINT NOT NULL DEFAULT 0,
    pdf_r2_key TEXT NOT NULL,
    file_name TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_annual_account_reports_entity_year ON annual_account_reports(legal_entity_id, fiscal_year);

---- create above / drop below ----

DROP TABLE IF EXISTS annual_account_reports;
DROP TABLE IF EXISTS account_balance_lines;
DROP TABLE IF EXISTS journal_lines;
DROP TABLE IF EXISTS journal_entries;
DROP TABLE IF EXISTS pcg_accounts;
