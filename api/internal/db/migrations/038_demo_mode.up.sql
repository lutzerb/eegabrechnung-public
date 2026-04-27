-- Add is_demo flag to eegs table
ALTER TABLE eegs ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;

-- Demo organization
INSERT INTO organizations (id, name, slug)
VALUES ('00000000-0000-0000-0000-000000000002', 'Demo Organisation', 'demo@demo.at')
ON CONFLICT (id) DO NOTHING;

-- Demo user (email=demo, password=demo, role=admin)
INSERT INTO users (id, organization_id, email, password_hash, name, role)
VALUES (
  '00000000-0000-0000-0000-000000000099',
  '00000000-0000-0000-0000-000000000002',
  'demo@demo.at',
  '$2b$10$QinVb5LEZ3J/.TylrDhSVepFrIeTW0w3yd/DGo9s2x5BfARN1fS7K',
  'Demo User',
  'admin'
)
ON CONFLICT (id) DO NOTHING;

-- Demo EEG
INSERT INTO eegs (
  id, organization_id, gemeinschaft_id, netzbetreiber, name,
  energy_price, producer_price, use_vat, vat_pct,
  meter_fee_eur, free_kwh, discount_pct, participation_fee_eur,
  billing_period,
  invoice_number_prefix, invoice_number_digits, invoice_number_start,
  invoice_pre_text, invoice_post_text, invoice_footer_text,
  generate_credit_notes, credit_note_number_prefix, credit_note_number_digits,
  iban, bic, sepa_creditor_id,
  strasse, plz, ort,
  is_demo
) VALUES (
  '00000000-0000-0000-0000-000000000010',
  '00000000-0000-0000-0000-000000000002',
  'AT_DEMO_EEG_001',
  'Netz OÖ GmbH',
  'Demo Energiegemeinschaft',
  8.5, 6.0, true, 20,
  2.0, 0, 0, 5.0,
  'monthly',
  'RE', 5, 1,
  'Vielen Dank für Ihre Mitgliedschaft bei der Demo Energiegemeinschaft.',
  'Bei Fragen wenden Sie sich an info@demo-eeg.at.',
  'Demo Energiegemeinschaft | Musterstraße 1 | 4020 Linz',
  false, 'GS', 5,
  'AT61 1904 3002 3457 3201', 'BKAUATWW', 'AT98ZZZ01234567890',
  'Musterstraße 1', '4020', 'Linz',
  true
)
ON CONFLICT (id) DO NOTHING;

-- Grant demo user access to demo EEG
INSERT INTO user_eeg_assignments (user_id, eeg_id)
VALUES ('00000000-0000-0000-0000-000000000099', '00000000-0000-0000-0000-000000000010')
ON CONFLICT DO NOTHING;

-- Demo members
INSERT INTO members (id, eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort, business_role, status, beitritt_datum)
VALUES
  ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010',
   'M001', 'Hans', 'Mustermann', 'hans@demo.at', 'AT61 1904 3002 3457 3201',
   'Hauptstraße 5', '4020', 'Linz', 'privat', 'ACTIVE', '2023-01-01'),
  ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000010',
   'M002', 'Maria', 'Sonnleitner', 'maria@demo.at', 'AT83 1904 3009 9876 0000',
   'Gartenweg 12', '4040', 'Linz', 'privat', 'ACTIVE', '2023-01-01'),
  ('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000010',
   'M003', 'Biobauernhof', 'Grünwald GmbH', 'hof@demo.at', 'AT73 3200 0000 1234 5600',
   'Feldweg 3', '4210', 'Gallneukirchen', 'unternehmen', 'ACTIVE', '2023-01-01')
ON CONFLICT (id) DO NOTHING;

-- Demo meter points (DELETE first to handle re-runs safely around the zaehlpunkt unique constraint)
DELETE FROM meter_points WHERE id IN (
  '00000000-0000-0000-0000-000000000021',
  '00000000-0000-0000-0000-000000000022',
  '00000000-0000-0000-0000-000000000023',
  '00000000-0000-0000-0000-000000000024'
);
INSERT INTO meter_points (id, eeg_id, member_id, zaehlpunkt, energierichtung, verteilungsmodell, zugeteilte_menge_pct, generation_type, status)
VALUES
  ('00000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000010',
   '00000000-0000-0000-0000-000000000011',
   'AT_DEMO_00000000000000000000001', 'CONSUMPTION', 'DYNAMIC', 0, NULL, 'ACTIVATED'),
  ('00000000-0000-0000-0000-000000000022', '00000000-0000-0000-0000-000000000010',
   '00000000-0000-0000-0000-000000000012',
   'AT_DEMO_00000000000000000000002', 'CONSUMPTION', 'DYNAMIC', 0, NULL, 'ACTIVATED'),
  ('00000000-0000-0000-0000-000000000023', '00000000-0000-0000-0000-000000000010',
   '00000000-0000-0000-0000-000000000012',
   'AT_DEMO_00000000000000000000003', 'GENERATION', 'DYNAMIC', 0, 'PV', 'ACTIVATED'),
  ('00000000-0000-0000-0000-000000000024', '00000000-0000-0000-0000-000000000010',
   '00000000-0000-0000-0000-000000000013',
   'AT_DEMO_00000000000000000000004', 'GENERATION', 'DYNAMIC', 0, 'PV', 'ACTIVATED');

-- Demo tariff schedule (active)
INSERT INTO tariff_schedules (id, eeg_id, name, granularity, is_active)
VALUES (
  '00000000-0000-0000-0000-000000000031',
  '00000000-0000-0000-0000-000000000010',
  'Standard 2024',
  'annual',
  true
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tariff_entries (schedule_id, valid_from, valid_until, energy_price, producer_price)
VALUES (
  '00000000-0000-0000-0000-000000000031',
  '2024-01-01', '2026-12-31',
  8.5, 6.0
)
ON CONFLICT (schedule_id, valid_from) DO NOTHING;
