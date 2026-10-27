ALTER TABLE inspections
    ADD CONSTRAINT fk_inspections_lease
    FOREIGN KEY (lease_id) REFERENCES leases(id) ON DELETE CASCADE;

ALTER TABLE properties
    ADD CONSTRAINT chk_properties_property_type
    CHECK (property_type IN ('APARTMENT', 'HOUSE', 'COMMERCIAL_PREMISES', 'BARE_LAND', 'PARKING', 'CELLAR', 'BUILDING'));

ALTER TABLE lots
    ALTER COLUMN lot_type SET DEFAULT 'APARTMENT',
    ADD CONSTRAINT chk_lots_lot_type
    CHECK (lot_type IN ('APARTMENT', 'HOUSE', 'COMMERCIAL_PREMISES', 'BARE_LAND', 'PARKING', 'CELLAR', 'BUILDING'));

CREATE INDEX IF NOT EXISTS idx_rent_calls_lease_unpaid_order
    ON rent_calls(lease_id, due_date ASC, period_month ASC)
    WHERE remaining_due_cents > 0 AND status <> 'PAID';

DROP TABLE IF EXISTS pcg_accounts;

---- create above / drop below ----

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
