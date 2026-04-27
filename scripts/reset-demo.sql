-- Reset demo EEG data to initial state.
-- Runs nightly via cron container.
-- Only touches data owned by the demo EEG (id: 00000000-0000-0000-0000-000000000010).

BEGIN;

DO $$
DECLARE
  demo_eeg_id UUID := '00000000-0000-0000-0000-000000000010';
  demo_org_id  UUID := '00000000-0000-0000-0000-000000000002';
BEGIN

  -- Delete derived data (order matters for FK constraints)
  DELETE FROM eda_errors         WHERE eeg_id = demo_eeg_id;
  DELETE FROM eda_processes      WHERE eeg_id = demo_eeg_id;
  DELETE FROM eda_messages       WHERE eeg_id = demo_eeg_id;
  DELETE FROM member_portal_sessions WHERE eeg_id = demo_eeg_id;
  DELETE FROM onboarding_requests    WHERE eeg_id = demo_eeg_id;
  DELETE FROM billing_runs       WHERE eeg_id = demo_eeg_id;
  DELETE FROM invoices           WHERE eeg_id = demo_eeg_id;
  DELETE FROM energy_readings    WHERE meter_point_id IN (
    SELECT id FROM meter_points WHERE eeg_id = demo_eeg_id
  );
  DELETE FROM eeg_meter_participations WHERE meter_point_id IN (
    SELECT id FROM meter_points WHERE eeg_id = demo_eeg_id
  );
  DELETE FROM tariff_entries     WHERE schedule_id IN (
    SELECT id FROM tariff_schedules WHERE eeg_id = demo_eeg_id
  );
  DELETE FROM tariff_schedules   WHERE eeg_id = demo_eeg_id;
  DELETE FROM meter_points       WHERE eeg_id = demo_eeg_id;
  DELETE FROM members            WHERE eeg_id = demo_eeg_id;
  DELETE FROM user_eeg_assignments WHERE eeg_id = demo_eeg_id;
  DELETE FROM eegs               WHERE id = demo_eeg_id;
  DELETE FROM users              WHERE organization_id = demo_org_id;
  DELETE FROM organizations      WHERE id = demo_org_id;

  -- Re-insert demo organization
  INSERT INTO organizations (id, name, slug)
  VALUES (demo_org_id, 'Demo Organisation', 'demo@demo.at');

  -- Re-insert demo user (email=demo, password=demo)
  INSERT INTO users (id, organization_id, email, password_hash, name, role)
  VALUES (
    '00000000-0000-0000-0000-000000000099',
    demo_org_id,
    'demo@demo.at',
    '$2b$10$QinVb5LEZ3J/.TylrDhSVepFrIeTW0w3yd/DGo9s2x5BfARN1fS7K',
    'Demo User',
    'admin'
  );

  -- Re-insert demo EEG
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
    demo_eeg_id, demo_org_id,
    'AT_DEMO_EEG_001', 'Netz OÖ GmbH', 'Demo Energiegemeinschaft',
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
  );

  -- Re-grant demo user access to demo EEG
  INSERT INTO user_eeg_assignments (user_id, eeg_id)
  VALUES ('00000000-0000-0000-0000-000000000099', demo_eeg_id);

  -- Re-insert demo members
  INSERT INTO members (id, eeg_id, mitglieds_nr, name1, name2, email, iban, strasse, plz, ort, business_role, status, beitritt_datum)
  VALUES
    ('00000000-0000-0000-0000-000000000011', demo_eeg_id,
     'M001', 'Hans', 'Mustermann', 'hans@demo.at', 'AT61 1904 3002 3457 3201',
     'Hauptstraße 5', '4020', 'Linz', 'privat', 'ACTIVE', '2023-01-01'),
    ('00000000-0000-0000-0000-000000000012', demo_eeg_id,
     'M002', 'Maria', 'Sonnleitner', 'maria@demo.at', 'AT83 1904 3009 9876 0000',
     'Gartenweg 12', '4040', 'Linz', 'privat', 'ACTIVE', '2023-01-01'),
    ('00000000-0000-0000-0000-000000000013', demo_eeg_id,
     'M003', 'Biobauernhof', 'Grünwald GmbH', 'hof@demo.at', 'AT73 3200 0000 1234 5600',
     'Feldweg 3', '4210', 'Gallneukirchen', 'unternehmen', 'ACTIVE', '2023-01-01');

  -- Re-insert demo meter points
  INSERT INTO meter_points (id, eeg_id, member_id, zaehlpunkt, energierichtung, verteilungsmodell, zugeteilte_menge_pct, generation_type, status)
  VALUES
    ('00000000-0000-0000-0000-000000000021', demo_eeg_id,
     '00000000-0000-0000-0000-000000000011',
     'AT_DEMO_00000000000000000000001', 'CONSUMPTION', 'DYNAMIC', 0, NULL, 'ACTIVATED'),
    ('00000000-0000-0000-0000-000000000022', demo_eeg_id,
     '00000000-0000-0000-0000-000000000012',
     'AT_DEMO_00000000000000000000002', 'CONSUMPTION', 'DYNAMIC', 0, NULL, 'ACTIVATED'),
    ('00000000-0000-0000-0000-000000000023', demo_eeg_id,
     '00000000-0000-0000-0000-000000000012',
     'AT_DEMO_00000000000000000000003', 'GENERATION', 'DYNAMIC', 0, 'PV', 'ACTIVATED'),
    ('00000000-0000-0000-0000-000000000024', demo_eeg_id,
     '00000000-0000-0000-0000-000000000013',
     'AT_DEMO_00000000000000000000004', 'GENERATION', 'DYNAMIC', 0, 'PV', 'ACTIVATED');

  -- Re-insert demo tariff schedule
  INSERT INTO tariff_schedules (id, eeg_id, name, granularity, is_active)
  VALUES (
    '00000000-0000-0000-0000-000000000031',
    demo_eeg_id,
    'Standard 2024',
    'annual',
    true
  );

  INSERT INTO tariff_entries (schedule_id, valid_from, valid_until, energy_price, producer_price)
  VALUES (
    '00000000-0000-0000-0000-000000000031',
    '2024-01-01', '2026-12-31',
    8.5, 6.0
  );

END $$;

COMMIT;
