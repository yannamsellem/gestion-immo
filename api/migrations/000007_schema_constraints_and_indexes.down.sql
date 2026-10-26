CREATE TABLE IF NOT EXISTS pcg_accounts (
    code TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    account_class INT NOT NULL CHECK (account_class BETWEEN 1 AND 8),
    section TEXT NOT NULL DEFAULT '',
    cerfa_code TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DROP INDEX IF EXISTS idx_rent_calls_lease_unpaid_order;

ALTER TABLE lots
    DROP CONSTRAINT IF EXISTS chk_lots_lot_type,
    ALTER COLUMN lot_type SET DEFAULT '';

ALTER TABLE properties
    DROP CONSTRAINT IF EXISTS chk_properties_property_type;

ALTER TABLE inspections
    DROP CONSTRAINT IF EXISTS fk_inspections_lease;
