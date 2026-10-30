CREATE TABLE IF NOT EXISTS document_templates (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL,
    template_type TEXT NOT NULL CHECK (template_type IN ('LEASE', 'RENT_RECEIPT', 'FORMAL_NOTICE', 'GLI_DECLARATION', 'CUSTOM')),
    format TEXT NOT NULL CHECK (format IN ('DOCX', 'HTML')),
    r2_key TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_document_templates_type ON document_templates(template_type);

CREATE TABLE IF NOT EXISTS stored_files (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    legal_entity_id UUID REFERENCES legal_entities(id) ON DELETE SET NULL,
    lease_id UUID REFERENCES leases(id) ON DELETE SET NULL,
    category TEXT NOT NULL DEFAULT '',
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    r2_key TEXT NOT NULL UNIQUE,
    size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stored_files_legal_entity_id ON stored_files(legal_entity_id);
CREATE INDEX IF NOT EXISTS idx_stored_files_lease_id ON stored_files(lease_id);
CREATE INDEX IF NOT EXISTS idx_stored_files_category ON stored_files(category);

CREATE TABLE IF NOT EXISTS gli_dossiers (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    legal_entity_id UUID NOT NULL REFERENCES legal_entities(id) ON DELETE CASCADE,
    template_id UUID REFERENCES document_templates(id) ON DELETE SET NULL,
    gli_policy_number TEXT NOT NULL DEFAULT '',
    total_unpaid_cents BIGINT NOT NULL DEFAULT 0,
    unpaid_months TEXT[] NOT NULL DEFAULT '{}',
    archive_r2_key TEXT NOT NULL,
    included_files TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gli_dossiers_lease_id ON gli_dossiers(lease_id);
CREATE INDEX IF NOT EXISTS idx_gli_dossiers_legal_entity_id ON gli_dossiers(legal_entity_id);
CREATE INDEX IF NOT EXISTS idx_gli_dossiers_template_id ON gli_dossiers(template_id);

CREATE TABLE IF NOT EXISTS inspection_templates (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name TEXT NOT NULL,
    rooms TEXT[] NOT NULL DEFAULT '{}',
    default_elements TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS inspections (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    lease_id UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES inspection_templates(id) ON DELETE RESTRICT,
    inspection_type TEXT NOT NULL CHECK (inspection_type IN ('MOVE_IN', 'MOVE_OUT')),
    entry_inspection_id UUID REFERENCES inspections(id) ON DELETE SET NULL,
    report_pdf_r2_key TEXT NOT NULL DEFAULT '',
    inspection_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inspections_lease_id ON inspections(lease_id);
CREATE INDEX IF NOT EXISTS idx_inspections_template_id ON inspections(template_id);
CREATE INDEX IF NOT EXISTS idx_inspections_entry_inspection_id ON inspections(entry_inspection_id) WHERE entry_inspection_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS inspection_items (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    inspection_id UUID NOT NULL REFERENCES inspections(id) ON DELETE CASCADE,
    room_name TEXT NOT NULL,
    element_name TEXT NOT NULL,
    condition TEXT NOT NULL CHECK (condition IN ('NEW', 'GOOD', 'FAIR_WEAR', 'POOR', 'OUT_OF_SERVICE')),
    observation TEXT NOT NULL DEFAULT '',
    photo_r2_keys TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (inspection_id, room_name, element_name)
);

CREATE INDEX IF NOT EXISTS idx_inspection_items_inspection_id ON inspection_items(inspection_id);

CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    property_id UUID REFERENCES properties(id) ON DELETE SET NULL,
    lot_id UUID REFERENCES lots(id) ON DELETE SET NULL,
    lease_id UUID REFERENCES leases(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'INSURANCE_CLAIM', 'WORK_IN_PROGRESS', 'CLOSED')),
    insurance_claim_number TEXT NOT NULL DEFAULT '',
    estimated_cost_cents BIGINT NOT NULL DEFAULT 0,
    observations TEXT[] NOT NULL DEFAULT '{}',
    photo_r2_keys TEXT[] NOT NULL DEFAULT '{}',
    occurred_at DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incidents_property_id ON incidents(property_id);
CREATE INDEX IF NOT EXISTS idx_incidents_lot_id ON incidents(lot_id);
CREATE INDEX IF NOT EXISTS idx_incidents_lease_id ON incidents(lease_id);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);

---- create above / drop below ----

DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS inspection_items;
DROP TABLE IF EXISTS inspections;
DROP TABLE IF EXISTS inspection_templates;
DROP TABLE IF EXISTS gli_dossiers;
DROP TABLE IF EXISTS stored_files;
DROP TABLE IF EXISTS document_templates;
