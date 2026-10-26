CREATE TABLE IF NOT EXISTS legal_entities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    entity_type TEXT NOT NULL CHECK (entity_type IN ('SCI_IR', 'SCI_IS', 'SARL_IS', 'FAMILY_SARL', 'SAS', 'SOLE_PROPRIETORSHIP', 'JOINT_OWNERSHIP')),
    siret TEXT NOT NULL DEFAULT '',
    naf_code TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    share_capital_cents BIGINT NOT NULL DEFAULT 0,
    total_shares BIGINT NOT NULL DEFAULT 100 CHECK (total_shares > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_legal_entities_siret ON legal_entities(siret) WHERE siret <> '';

CREATE TABLE IF NOT EXISTS shareholders (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    shares_count BIGINT NOT NULL DEFAULT 0 CHECK (shares_count >= 0),
    detention_basis_points BIGINT NOT NULL DEFAULT 0 CHECK (detention_basis_points >= 0 AND detention_basis_points <= 10000),
    current_account_cents BIGINT NOT NULL DEFAULT 0,
    remuneration_paid_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (legal_entity_id, full_name)
);

CREATE INDEX IF NOT EXISTS idx_shareholders_legal_entity_id ON shareholders(legal_entity_id);

CREATE TABLE IF NOT EXISTS properties (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    property_type TEXT NOT NULL,
    accounting_category TEXT NOT NULL CHECK (accounting_category IN ('FIXED_ASSET', 'MERCHANDISE', 'FINISHED_GOOD')),
    address TEXT NOT NULL DEFAULT '',
    acquisition_cents BIGINT NOT NULL DEFAULT 0,
    work_stock_cents BIGINT NOT NULL DEFAULT 0,
    total_book_value_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_properties_legal_entity_id ON properties(legal_entity_id);

CREATE TABLE IF NOT EXISTS lots (
    id TEXT PRIMARY KEY,
    property_id TEXT NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    lot_type TEXT NOT NULL DEFAULT '',
    surface_m2 BIGINT NOT NULL DEFAULT 0,
    ownership_share BIGINT NOT NULL DEFAULT 0,
    floor TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lots_property_id ON lots(property_id);
