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
