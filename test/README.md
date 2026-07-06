# Integration Test Harness

End-to-end integration tests for the eegabrechnung platform. Tests run against
the live Docker stack and cover the full application lifecycle from onboarding
through billing, EDA XML exchange, XLSX import, and accounting export.

## Quick Start

```bash
# 1. Start the full test stack (API + Postgres + EDA worker + Mailpit)
cd /path/to/eegabrechnung
docker compose --profile test up -d

# 2. Activate the Python virtual environment
cd test
source .venv/bin/activate   # or: python3 -m venv .venv && pip install -r requirements.txt

# 3. Run all tests
pytest -v

# 4. Run a single test file
pytest test_04_eda_mail.py -v
```

---

## Architecture

```
test/
├── conftest.py            # Session fixtures: API client, test EEG, members, readings, DB conn
├── helpers/
│   ├── api.py             # Typed API client for all Go API endpoints
│   ├── eda_xml.py         # CPDocument XML builders (inbound confirmations)
│   ├── mailpit.py         # Mailpit HTTP API + plain-SMTP helper
│   └── xlsx_builder.py    # In-memory XLSX builders for import tests
├── test_00_health.py      # Health checks + auth
├── test_01_billing.py     # Billing run lifecycle
├── test_02_sepa.py        # SEPA pain.001 / pain.008 XML
├── test_03_eda_file.py    # EDA process API (no worker needed)
├── test_04_eda_mail.py    # EDA end-to-end with FILE transport + Mailpit
├── test_05_onboarding.py  # Member onboarding (public + admin flow)
├── test_06_import.py      # XLSX import: Stammdaten + Energiedaten
├── test_07_accounting.py  # Accounting export: XLSX Buchungsjournal + DATEV
├── pytest.ini
├── requirements.txt
└── run_tests.sh
```

### Docker compose profiles

| Profile | Services added | Purpose |
|---------|---------------|---------|
| _(none)_ | postgres, api, web | Normal development |
| `test` | + mailpit, eda-worker-test | Integration testing |
| `eda` | + eda-worker (MAIL/PONTON) | Production EDA worker |

The `test` profile runs the EDA worker in **FILE transport** mode:
- Outbound XML → `test/eda-outbox/`
- Inbound CPDocuments ← `test/eda-inbox/`
- Mailpit captures SMTP at port 1025, HTTP API at port 8025

---

## Test Data Isolation

**All test data is fully isolated from production data.**

- The session fixture creates a **fresh `[AUTOTEST]` EEG** per test run with a unique 8-character UUID suffix: `[AUTOTEST] Test-EEG <suffix>`.
- The teardown safety guard checks for the `[AUTOTEST]` name prefix before cancelling billing runs, so it cannot affect production EEGs even if pointed at the wrong database.
- The test EEG itself persists after the run (no DELETE /eegs endpoint), but is clearly labelled and has no impact on production workflows.
- Energy readings are inserted for test meter points only, using `ON CONFLICT DO NOTHING`.
- Import tests (Stammdaten, Energiedaten) use unique UUID-based Zählpunkt IDs to avoid collisions.
- EDA inbox/outbox directories are cleared before each EDA test, but these directories are test-specific (mounted via Docker volume for the test worker only).

---

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `API_URL` | `http://localhost:8101` | Go API base URL |
| `WORKER_URL` | `http://localhost:8082` | EDA worker HTTP URL |
| `MAILPIT_URL` | `http://localhost:8025` | Mailpit HTTP API |
| `MAILPIT_SMTP_HOST` | `localhost` | Mailpit SMTP host |
| `MAILPIT_SMTP_PORT` | `1025` | Mailpit SMTP port |
| `ADMIN_EMAIL` | `admin@eeg.at` | Admin login email |
| `ADMIN_PASSWORD` | `admin` | Admin login password |
| `DB_DSN` | `postgresql://eegabrechnung:eegabrechnung@localhost:26433/eegabrechnung` | Direct DB access (readings insertion) |

---

## Test Files

### `test_00_health.py` — Health & Auth (5 tests)

Basic smoke tests that verify the stack is up before any other tests run.

| Test | What it checks |
|------|----------------|
| `test_api_health` | `GET /health` → 200 |
| `test_api_auth_rejects_bad_password` | Login with wrong password → 401 |
| `test_api_login_returns_token` | Login → JWT token returned |
| `test_api_list_eegs` | List EEGs with valid token → 200 |
| `test_worker_health` | EDA worker `GET /health` → 200 (skipped if worker not running) |

---

### `test_01_billing.py` — Billing Run Lifecycle (9 tests)

Full billing cycle including storno and draft deletion. Uses Jan–Mar 2025 readings.

| Test | What it checks |
|------|----------------|
| `TestBillingCycle::test_create_billing_run` | Create draft run → status=draft |
| `TestBillingCycle::test_billing_run_has_invoices` | Invoices generated with total_amount |
| `TestBillingCycle::test_duplicate_billing_run_rejected` | Overlapping period → 409 |
| `TestBillingCycle::test_finalize_billing_run` | Finalize → status=finalized |
| `TestBillingCycle::test_cannot_delete_finalized_run` | DELETE finalized → 400/409/422 |
| `TestBillingCycle::test_cancel_billing_run_creates_storno` | Cancel → status=cancelled |
| `TestBillingCycle::test_invoices_have_storno_pdf_after_cancel` | storno_pdf_path set |
| `TestDraftDelete::test_create_and_delete_draft` | Delete draft → gone from list |
| `TestInvoicePDF::test_invoice_pdf_is_pdf` | PDF bytes start with `%PDF` |

---

### `test_02_sepa.py` — SEPA Files (3 tests, conditionally skipped)

Skipped when test members have no IBAN. To enable: add IBAN to members in conftest.

| Test | What it checks |
|------|----------------|
| `test_pain001_is_valid_xml` | pain.001 credit transfer → parseable XML |
| `test_pain008_is_valid_xml` | pain.008 direct debit → parseable XML |
| `test_pain001_contains_creditor_id` | SEPA creditor ID present in output |

---

### `test_03_eda_file.py` — EDA Process API (5 tests)

Tests the EDA process creation API endpoints. **No worker required** — only the Go API.

| Test | What it checks |
|------|----------------|
| `test_anmeldung_requires_eda_settings` | POST /eda/anmeldung → 200 when EDA configured |
| `test_anmeldung_process_appears_in_list` | Created process visible in GET /eda/processes |
| `test_anmeldung_without_eda_settings_fails` | EEG without EDA config → 400 |
| `test_abmeldung_creates_process` | POST /eda/abmeldung → process created |
| `test_process_has_deadline` | deadline_at set to ~60 days from now |

---

### `test_04_eda_mail.py` — EDA End-to-End (8 tests)

**Requires the `test` Docker profile** (`docker compose --profile test up -d`).

Tests the full EDA XML pipeline using FILE transport:
- Outbound: API creates process → worker writes XML to `eda-outbox/`
- Inbound: CPDocument XML placed in `eda-inbox/` → worker updates process status

| Test | What it checks |
|------|----------------|
| `TestOutboundXML::test_anmeldung_writes_xml_to_outbox` | Anmeldung job → XML in outbox, process=sent |
| `TestOutboundXML::test_abmeldung_writes_xml_to_outbox` | Abmeldung job → XML in outbox |
| `TestInboundCPDocument::test_erste_anm_sets_first_confirmed` | ERSTE_ANM → first_confirmed |
| `TestInboundCPDocument::test_finale_anm_sets_confirmed` | ERSTE_ANM + FINALE_ANM → confirmed |
| `TestInboundCPDocument::test_abgelehnt_sets_rejected` | ABGELEHNT_ANM → rejected |
| `TestInboundCPDocument::test_unknown_conversation_id_ignored` | Unknown conv ID → no crash |
| `TestInboundCPDocument::test_duplicate_xml_processed_once` | Same XML twice → idempotent |
| `TestSMTPConnectivity::test_smtp_send_xml_arrives_in_mailpit` | SMTP → Mailpit captures message |

**EDA design note:** Mailpit v1.x has SMTP + POP3 only (no IMAP). The test worker uses
FILE transport for CPDocument processing; Mailpit verifies the outbound SMTP path separately.
The `poll-now` endpoint (`POST /eda/poll-now`) triggers an immediate poll cycle and returns
202 immediately while the work runs in a background goroutine.

---

### `test_05_onboarding.py` — Member Onboarding (12 tests)

Tests the full member onboarding lifecycle — from public application to admin approval
and automatic member creation.

| Test | What it checks |
|------|----------------|
| `TestPublicEEGInfo::test_returns_name_and_billing_period` | Public EEG info (no auth) |
| `TestPublicEEGInfo::test_unknown_eeg_returns_404` | Unknown EEG → 404 |
| `TestOnboardingSubmit::test_missing_name_returns_400` | Validation: name1 required |
| `TestOnboardingSubmit::test_missing_email_returns_400` | Validation: email required |
| `TestOnboardingSubmit::test_contract_not_accepted_returns_400` | Validation: contract required |
| `TestOnboardingSubmit::test_valid_submission_returns_201` | Submit → 201 + id |
| `TestOnboardingSubmit::test_status_endpoint_requires_valid_token` | Bad token → 404/410 |
| `TestOnboardingAdminFlow::test_admin_can_list_pending_requests` | Admin list → includes new request |
| `TestOnboardingAdminFlow::test_status_visible_via_token` | Magic token → status page |
| `TestOnboardingAdminFlow::test_reject_sets_status_rejected` | Admin reject → rejected |
| `TestOnboardingAdminFlow::test_approve_creates_member_and_sets_converted` | Approve → member created |
| `TestOnboardingAdminFlow::test_double_approve_returns_409` | Re-approve → 409 |

**Note:** The magic token is `json:"-"` in the API response. Tests that need the token
fetch it directly from the DB via psycopg2.

---

### `test_06_import.py` — XLSX Import (11 tests)

Tests Stammdaten (member master data) and Energiedaten (readings) XLSX import.
XLSX files are built in-memory by `helpers/xlsx_builder.py`.

| Test | What it checks |
|------|----------------|
| `TestStammdatenImport::test_import_creates_member_and_meter_point` | Upload → member + meter point created |
| `TestStammdatenImport::test_import_only_activated_rows` | INACTIVE rows skipped |
| `TestStammdatenImport::test_import_idempotent_on_reupload` | Re-upload → upsert, no duplicates |
| `TestEnergieDatenImport::test_preview_returns_new_row_count` | Preview → total/new/conflict counts |
| `TestEnergieDatenImport::test_import_inserts_rows` | Import → rows_inserted reported |
| `TestEnergieDatenImport::test_import_skip_mode_does_not_overwrite` | mode=skip → existing unchanged |
| `TestEnergieDatenImport::test_unknown_zaehlpunkt_reported_as_skipped` | Unknown meter → skipped_meters |
| `TestCoverage::test_coverage_returns_year_and_days` | Coverage → year + days array |
| `TestCoverage::test_coverage_days_include_covered_months` | Jan–May 2025 has ≥140 covered days |
| `TestCoverage::test_coverage_empty_year_returns_empty_list` | Year with no data → empty |

---

### `test_07_accounting.py` — Accounting Export (8 tests)

Tests the XLSX Buchungsjournal and DATEV EXTF CSV export endpoints.

**Important:** The accounting export filters invoices by `created_at` (invoice creation date),
not by billing period. Tests that verify data rows use the current-month range (`EXPORT_FROM`/
`EXPORT_TO`), while format-only tests can use any range.

| Test | What it checks |
|------|----------------|
| `test_xlsx_export_is_valid_xlsx` | Response is valid XLSX (PK header + worksheets) |
| `test_xlsx_export_contains_buchungsjournal_sheet` | Sheet "Buchungsjournal" with expected headers |
| `test_datev_export_starts_with_extf` | DATEV starts with `"EXTF"` |
| `test_datev_export_has_header_and_column_row` | Row 1 = EXTF header, row 2 = column headers |
| `test_datev_export_has_data_rows_when_invoices_exist` | Invoices → ≥1 data row in DATEV |
| `test_missing_from_date_returns_400` | Missing `from` param → 400 |
| `test_missing_to_date_returns_400` | Missing `to` param → 400 |
| `test_export_empty_period_returns_valid_files` | No invoices → valid but empty XLSX/DATEV |

---

## Fixtures Reference

All fixtures are defined in `conftest.py`. Session-scoped fixtures run once per `pytest` invocation.

| Fixture | Scope | Description |
|---------|-------|-------------|
| `api` | session | Authenticated `APIClient` (admin login) |
| `test_eeg` | session | Creates `[AUTOTEST] Test-EEG <id>` with EDA settings; teardown cancels billing runs |
| `test_members` | session | 3 members: consumer, prosumer, producer |
| `test_meter_points` | session | 1 meter point per member (unique Zählpunkt IDs with run suffix) |
| `test_readings` | session | Inserts hourly readings Jan–May 2025 directly via psycopg2 |
| `db_conn` | session | Raw psycopg2 connection (used to fetch magic tokens not exposed by API) |

---

## Adding New Tests

1. Create `test_NN_name.py` in `test/`.
2. Use session fixtures (`api`, `test_eeg`, `test_members`, `test_meter_points`, `test_readings`) — these are shared across the entire run.
3. Use **unique identifiers** (e.g., `uuid4().hex[:8]`) for any new EDA Zählpunkte, member names, or inbox filenames to avoid cross-run database collisions.
4. For billing runs, use a month **not used by other tests** (Jan–May 2025 is available; Apr 2025 used by test_07, Jun+ has no readings by default).
5. Add API methods to `helpers/api.py` if new endpoints are needed.

---

## Known Limitations

- **SEPA tests skip** when test members have no IBAN. The conftest could be extended to add IBANs.
- **EDA worker tests require** `docker compose --profile test up -d`. Without the worker, `test_04_eda_mail.py` is auto-skipped.
- **Test EEGs accumulate** in the DB (no DELETE /eegs endpoint). Each run adds one `[AUTOTEST]` EEG. These are harmless but can be cleaned up manually with:
  ```sql
  DELETE FROM eegs WHERE name LIKE '[AUTOTEST]%';
  ```
- The **magic token** for onboarding status is not exposed via the API (it's `json:"-"`). `test_05_onboarding.py` fetches it directly from the DB. This is acceptable for integration tests.
