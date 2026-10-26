CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_archived ON tenants(archived);

CREATE TABLE IF NOT EXISTS leases (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    lot_id TEXT NOT NULL REFERENCES lots(id) ON DELETE RESTRICT,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    lease_type TEXT NOT NULL CHECK (lease_type IN ('RESIDENTIAL_UNFURNISHED', 'RESIDENTIAL_FURNISHED', 'COMMERCIAL_3_6_9', 'PROFESSIONAL', 'PARKING', 'SHORT_TERM')),
    rent_net_cents BIGINT NOT NULL DEFAULT 0 CHECK (rent_net_cents >= 0),
    charges_cents BIGINT NOT NULL DEFAULT 0 CHECK (charges_cents >= 0),
    subject_to_vat BOOLEAN NOT NULL DEFAULT FALSE,
    vat_rate_basis_points BIGINT NOT NULL DEFAULT 0 CHECK (vat_rate_basis_points >= 0 AND vat_rate_basis_points <= 10000),
    security_deposit_cents BIGINT NOT NULL DEFAULT 0 CHECK (security_deposit_cents >= 0),
    due_day_of_month INT NOT NULL DEFAULT 1 CHECK (due_day_of_month >= 1 AND due_day_of_month <= 31),
    gli_policy_number TEXT NOT NULL DEFAULT '',
    guarantor_name TEXT NOT NULL DEFAULT '',
    start_date DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date DATE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_leases_legal_entity_id ON leases(legal_entity_id);
CREATE INDEX IF NOT EXISTS idx_leases_lot_id ON leases(lot_id);
CREATE INDEX IF NOT EXISTS idx_leases_tenant_id ON leases(tenant_id);

CREATE TABLE IF NOT EXISTS rent_calls (
    id TEXT PRIMARY KEY,
    lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    period_month TEXT NOT NULL,
    rent_net_cents BIGINT NOT NULL DEFAULT 0,
    vat_cents BIGINT NOT NULL DEFAULT 0,
    charges_cents BIGINT NOT NULL DEFAULT 0,
    total_due_cents BIGINT NOT NULL DEFAULT 0,
    remaining_due_cents BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'PARTIAL', 'PAID', 'UNPAID')),
    due_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (lease_id, period_month)
);

CREATE INDEX IF NOT EXISTS idx_rent_calls_legal_entity_id ON rent_calls(legal_entity_id);
CREATE INDEX IF NOT EXISTS idx_rent_calls_lease_id ON rent_calls(lease_id);
CREATE INDEX IF NOT EXISTS idx_rent_calls_status ON rent_calls(status);
