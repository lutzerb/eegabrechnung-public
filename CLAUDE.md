# eegabrechnung — Claude Instructions

## Project Overview
Austrian EEG (Energiegemeinschaft) billing platform.

- **api/** — Go REST API (chi/v5, pgx/v5, golang-migrate, golang-jwt/v5)
- **web/** — Next.js 16 App Router frontend (Tailwind CSS, next-auth v5 beta)
- **docker-compose.yaml** — full local stack (3 containers: postgres, api, web)

## Running the Stack
```bash
cd /mnt/HC_Volume_103451728/eegabrechnung
docker compose up -d
```
- Web: http://localhost:3001
- API: http://localhost:8101
- Postgres: localhost:26433

**Only port 3001 needs to be forwarded** — the browser never talks to the API directly. All client-side API calls go through Next.js proxy routes under `web/app/api/`.

## Login
- URL: http://localhost:3001
- Email: `admin@eegwn.at`
- Password: siehe `secrets/admin-credentials.txt` (nicht im Repo)

**No Keycloak.** Auth is handled by a Credentials provider in next-auth v5 that calls the Go API's `/api/v1/auth/login` endpoint. Passwords are bcrypt-hashed and stored in Postgres.

## Auth Architecture
- **next-auth v5 beta** with `CredentialsProvider` — email/password login form in the app itself
- `trustHost: true` required for SSH tunnel / non-localhost deployments
- Go API signs its own **HS256 JWTs** with `JWT_SECRET` (shared env var)
- `api/internal/auth/jwt.go` — `SignToken` / `ParseToken`
- `api/internal/auth/middleware.go` — validates Bearer token, stores `*Claims` in context
- `auth.ClaimsFromContext(ctx)` — used in handlers to get `OrganizationID` for multi-tenancy
- Token lifetime: 8 hours (no refresh needed — Credentials flow returns fresh token on re-login)

## Multi-tenancy
Every user belongs to an `organization`. All EEG queries are scoped by `organization_id` from the JWT claims. Members/meter-points are implicitly scoped because they belong to EEGs.

Default org ID: `00000000-0000-0000-0000-000000000001` (created by migration 005).

## Docker Networking
Server-side Next.js calls the API at `http://eegabrechnung-api:8080` (internal Docker network) via `API_INTERNAL_URL`. **All browser→API calls go through Next.js proxy routes** (`web/app/api/...`) — never directly to port 8101.

**Critical:** containers cannot reach `localhost` — always use the internal service name for server-to-server calls.

## Proxy Route Pattern
All client-side API calls use Next.js route handlers under `web/app/api/`. Pattern:
```ts
// web/app/api/eegs/[eegId]/something/route.ts
import { auth } from "@/lib/auth";
const API = process.env.API_INTERNAL_URL || "http://localhost:8080";
export async function GET(request: Request, { params }) {
  const session = await auth();
  if (!session?.accessToken) return Response.json({ error: "Unauthorized" }, { status: 401 });
  const res = await fetch(`${API}/api/v1/eegs/${params.eegId}/something`, {
    headers: { Authorization: `Bearer ${session.accessToken}` },
  });
  return Response.json(await res.json(), { status: res.status });
}
```

## Database Migrations
Embedded in `api/internal/db/migrations/`, applied automatically at startup via golang-migrate.

| Migration | What it adds |
|-----------|-------------|
| 001_init | eegs, members, meter_points, energy_readings, invoices |
| 002_eda | eda_messages |
| 003_pricing | producer_price, use_vat, vat_pct, meter_fee_eur, free_kwh, discount_pct, participation_fee_eur, billing_period |
| 004_features | invoice_number_prefix/digits, invoice_pre/post/footer_text, invoice status |
| 005_auth | organizations, users, organization_id on eegs; default org + admin user |
| 006_member_vat | per-member use_vat / vat_pct overrides |
| 007_member_uid | uid_nummer (VAT ID) on members |
| 008_member_address | strasse, plz, ort on members |
| 009_invoice_breakdown | consumption_kwh, generation_kwh columns on invoices |
| 010_sepa_fields | iban, bic, sepa_creditor_id on eegs |
| 011_billing_runs | billing_runs table; invoice → billing_run_id FK |
| 012_user_assignments | user_eeg_assignments (per-user EEG access control) |
| 013_tariff_schedules | tariff_schedules + tariff_entries; partial unique index for one-active-per-EEG |
| 014_eda_schema | message_type/subject/body/processed_at on eda_messages; eda_marktpartner_id/eda_netzbetreiber_id/eda_transition_date on eegs |
| 015_eda_processes | eda_processes table (process_type, status lifecycle, conversation_id, participation_factor, deadline_at) |
| 016_eda_gaps | source column on energy_readings (xlsx\|eda); message_id dedup on eda_messages; eda_errors dead-letter table; eda_worker_status singleton |
| 017_member_status_invstart | status column on members (ACTIVE/INACTIVE); invoice_number_start on eegs |
| 018_energy_quality | quality column on energy_readings (L0/L1/L2/L3); L3 excluded from billing |
| 019_credit_notes | generate_credit_notes, credit_note_number_prefix/digits on eegs; document_type on invoices (invoice\|credit_note) |
| 020_logo | logo_path on eegs |
| 021_mehrfachteilnahme | eeg_meter_participations table (factor, share_type, valid_from/until) for multi-EEG membership |
| 022_onboarding | onboarding_requests table (magic_token flow, status: pending→approved→converted/rejected) |
| 023_member_dates | beitritt_datum, austritt_datum on members |
| 024_onboarding_beitritt | beitritts_datum on onboarding_requests |
| 025_accounting | net_amount/vat_amount/vat_pct_applied on invoices; DATEV fields on eegs (revenue/expense/debitor accounts, consultant/client nr) |
| 026_eeg_address_billing_workflow | strasse/plz/ort/uid_nummer on eegs; billing_run status 'completed'→'finalized'; storno_pdf_path on invoices |
| 027_job_retry | retry_count on jobs table |
| 028_eeg_gruendungsdatum | gruendungsdatum (founding date) on eegs |
| 029_meter_point_generation_type | generation_type on meter_points (PV/Windkraft/Wasserkraft etc.) |
| 030_member_portal | member_portal_sessions table (magic-link auth for member self-service) |
| 031_onboarding_contract | onboarding_contract_text on eegs |
| 032_member_email_campaigns | member_email_campaigns table + attachments |
| 033_eeg_documents | eeg_documents table (uploadable PDFs for onboarding page) |
| 034_onboarding_email_verify | onboarding_email_verifications table (magic-token email verify) |
| 035_invoice_split_vat | Split VAT on invoices (separate generation/consumption rows) |
| 036_invoice_number_uniqueness | Unique index on invoice number per EEG |
| 037_document_show_in_onboarding | show_in_onboarding flag on eeg_documents |
| 038_demo_mode | is_demo flag on eegs (blocks email sending in demo EEGs) |
| 039_eda_message_status | status column on eda_messages (pending/processed/error) |
| 040_eda_message_addresses | from_address/to_address on eda_messages |
| 041_eda_process_ecmplist_fields | Additional fields on eda_processes for ECMPList |
| 042_eda_messages_process_id | eda_process_id FK on eda_messages |
| 043_per_eeg_credentials | Per-EEG credentials on eegs: eda_imap_*, eda_smtp_*, smtp_* (AES-256-GCM encrypted) |
| 044_eda_error_subject | subject column on eda_errors (stores MailSubject of the referenced outbound message) |
| 045_onboarding_business_fields | business_role, uid_nummer, use_vat on onboarding_requests; carried through to member on Convert |
| 046_onboarding_reminder | reminder_sent_at on onboarding_requests; used by hourly background check to send 72h follow-up |
| 047_meter_point_abgemeldet_am | abgemeldet_am (date) on meter_points; set by worker when CM_REV_SP confirmed (CM_REV_CUS/CM_REV_IMP) via ECMPList |
| 048_eda_process_error_notification | error_notification_sent_at on eda_processes; used by worker to send email on error/rejected status |
| 049_auto_billing | auto_billing_enabled/day_of_month/period/last_run_at on eegs; daily scheduler creates draft billing runs |
| 050_sepa_return | sepa_return_at/reason/note on invoices; manual return tracking + CAMT.054 auto-import |
| 051_gap_alert | gap_alert_enabled/threshold_days on eegs; gap_alert_sent_at on meter_points; hourly GapChecker sends email when readings missing |
| 052/053_meter_point_notes | notes (text, NOT NULL DEFAULT '') on meter_points |
| 054_rename_ec_einzel_anm | Renames EC_EINZEL_ANM → EC_REQ_ONL in eda_processes and pending jobs (canonical ebutilities.at process name) |
| 055_meter_point_consent_id | consent_id (text, DEFAULT '') on meter_points; stores NB-assigned ConsentId from ZUSTIMMUNG_ECON; required for CM_REV_SP |
| 056_ea_buchhaltung | ea_konten, ea_buchungen, ea_belege, ea_uva_perioden tables; ea_steuernummer/ea_finanzamt/ea_uva_periodentyp on eegs |
| 057_ea_banktransaktionen | ea_banktransaktionen table (bank statement import MT940/CAMT053; match_status: offen/auto/bestaetigt/ignoriert) |
| 058_uva_kennzahlen | kz_044 (10% USt) + kz_057 (RC domestic §19) on ea_uva_perioden |
| 059/060_konto_k1kz | k1_kz on ea_konten; maps each account to FinanzOnline K1 Kennzahl (059 initial, 060 corrected per BMF K1 2025) |
| 061_sepa_mandate_prenotification | sepa_mandate_signed_at/ip/text on members (captured during onboarding); sepa_pre_notification_days on eegs (default 14) |
| 062_portal_show_full_energy | portal_show_full_energy (bool, default true) on eegs; controls whether member portal shows full EEG energy or only member share |
| 063_invoice_split_amounts | consumption_net_amount + generation_net_amount (DECIMAL 12,4) on invoices; backfilled for pure consumers/producers; prosumers approximate via flat EEG prices |
| 064_invoice_split_amounts_tariff_backfill | Re-backfill prosumer split amounts for EEGs using tariff schedules (where flat energy_price=0 left them as zeros); uses time-weighted average tariff price over billing period |
| 065_invoice_split_amounts_vat_recovery | Exact recovery of prosumer split amounts for non-KU EEGs (use_vat=true, vat_pct > 0) using consumptionNet = vat_amount × 100 / vat_pct; remaining KU EEG rows fixed at startup by billing.BackfillSplitAmounts() |
| 066_ea_buchungen_audit | Soft-delete on ea_buchungen (`deleted_at`, `deleted_by`); `ea_buchungen_changelog` table (BAO §131 audit trail — records create/update/delete with old/new JSON snapshots, `changed_by`, `reason`); all read queries filter `deleted_at IS NULL` by default; `incl_deleted=true` param on list to include soft-deleted entries |

## Energy Unit Convention
**All energy values throughout the codebase are stored and transmitted in kWh**, despite column/field names using the `wh_` prefix (e.g. `wh_total`, `wh_self`, `wh_community`). This naming is a historical artifact — do NOT divide these values by 1000 when displaying as kWh. The `fmtKwh()` helper in `web/components/energy-charts.tsx` is the reference implementation: it displays values directly as kWh and only converts to MWh when the value exceeds 100 000.

## Key Features
- **Billing runs**: group invoices per billing operation; overlap detection (409 on double-billing); member/type filter; force override; draft→finalized workflow; storno PDF on cancellation
- **Auto-billing scheduler**: daily check at 06:00 Vienna time; checks `auto_billing_enabled` EEGs; data completeness check via `MissingReadingDays()`; creates draft run + sends email; skips on overlap or recent run (< 20 days); `billing/scheduler.go`
- **SEPA Rücklastschriften**: `PATCH .../invoices/{id}/sepa-return` sets/clears manual return (reason, note, date); `GET .../invoices?sepa_returned=true` filters; `POST .../sepa/camt054` imports CAMT.054 XML (matches by EndToEndId=invoice UUID); `sepa/camt054.go` namespace-agnostic parser; dashboard alert; red badge in billing page
- **Tariff schedules**: time-series pricing (annual/monthly/daily/15-min granularity); one active schedule per EEG; weighted fallback to flat EEG prices for uncovered periods
- **Credit notes**: VAT-liable producers receive Gutschriften instead of negative invoices; document_type on invoices distinguishes invoice vs credit_note
- **SEPA files**: pain.001 (credit transfers) and pain.008 (direct debits) generation
- **Import overlap handling**: preview endpoint compares XLSX rows against DB; overwrite/skip/cancel choice; source tracking (xlsx|eda); quality filtering (L0-L2 billed, L3 excluded)
- **Data coverage timeline**: per-day reading coverage chart on import page; auto-refreshes after import
- **Energy reports**: per-member and EEG-wide analytics; granularity: year/month/day/15-min; CSV/XLSX export; kWh display (switches to MWh above 100 000 kWh)
- **Accounting export**: DATEV Buchungsstapel CSV + XLSX; configurable GL accounts; VAT breakdown stored at billing time
- **User administration**: role-based access (admin/user); per-user EEG assignments
- **Member lifecycle**: status (ACTIVE/INACTIVE); beitritt_datum/austritt_datum; configurable invoice number start
- **Austritt-Workflow**: `POST .../members/{id}/austritt` sets INACTIVE + austritt_datum, enqueues CM_REV_SP (Widerruf) for all active meter points with a stored consent_id; idempotent — skips ZPs with pending/sent CM_REV_SP; EDA only triggered when EEG has credentials + not demo
- **Meter point delete**: button on member detail page; only allowed when no active EDA processes pending; sends CM_REV_SP (Widerruf) if meter point has consent_id
- **SEPA Mandat**: `GET .../members/{memberID}/sepa-mandat` generates SEPA direct debit mandate PDF; sepa_mandate_signed_at/ip/text captured during onboarding; sepa_pre_notification_days (default 14) configures collection date offset per SEPA Rulebook
- **Member email campaigns**: `POST .../communications` sends HTML email to selected members (all/consumer/producer/prosumer/individual selection); placeholder substitution ({{name}}, {{eeg_name}}, etc.); attachments; `GET .../communications` lists sent campaigns with full history; `GET .../communications/{id}` returns campaign detail
- **Meter point notes**: free-text notes field on meter points; shown on member detail page
- **Onboarding portal**: public self-service registration form with magic-token email verification; admin approval queue; auto-creates member + meter points + EDA Anmeldung on convert
- **Member portal**: magic-link self-service dashboard for members; monthly energy breakdown; invoice list + PDF download (no password required)
- **Mehrfachteilnahme**: meter point participates in multiple EEGs simultaneously (Austrian EAG April 2024); factor + share type (GC/RC_R/RC_L/CC) + date range; source of truth for EDA
- **OeMAG market prices**: scraped from oem-ag.at; sync to producer/energy price
- **EDA process management**: Anmeldung (EC_REQ_ONL), Teilnahmefaktor (EC_PRTFACT_CHG), Widerruf (CM_REV_SP), Datenanforderung (EC_REQ_PT), Zählpunktliste (EC_PODLIST) via MaKo XML; process lifecycle tracking (pending→sent→first_confirmed→confirmed/completed/rejected/error); deadline tracking; duplicate-change prevention; eda_errors dead-letter table; eda_worker_status singleton
- **EEG settings**: address (strasse/plz/ort/uid_nummer for §11 UStG invoice block), logo, founding date, generation type on meter points (PV/Wind/Wasser)
- **Backup/Restore**: full EEG snapshot export (JSON) and restore via transaction

## Additional API Endpoints

```
# Member lifecycle
POST /api/v1/eegs/{eegID}/members/{memberID}/austritt  — deregister member (sets INACTIVE, triggers CM_REV_SP for all active meter points with consent_id)
                                                         body: { "austritt_datum": "YYYY-MM-DD" }; idempotent (skips ZPs with pending CM_REV_SP)
GET  /api/v1/eegs/{eegID}/members/{memberID}/sepa-mandat — download SEPA direct debit mandate PDF

# Bulk email campaigns
GET  /api/v1/eegs/{eegID}/communications               — list sent campaigns
GET  /api/v1/eegs/{eegID}/communications/{id}          — get campaign detail
POST /api/v1/eegs/{eegID}/communications               — send campaign; multipart: subject, body (HTML), member_ids[] or member_type, attachments

# Onboarding
POST /api/v1/public/eegs/{eegID}/onboarding           — submit membership application (public, no auth)
GET  /api/v1/eegs/{eegID}/onboarding                  — list onboarding requests (admin)
POST /api/v1/eegs/{eegID}/onboarding/{id}/convert      — approve & convert to member
DELETE /api/v1/eegs/{eegID}/onboarding/{id}            — reject/delete request

# Member portal (magic-link, no Bearer auth)
POST /api/v1/public/portal/request-link               — send magic link to member email
GET  /api/v1/public/portal/{token}/activate            — activate session from link
GET  /api/v1/portal/me                                 — member dashboard data (energy + invoices)

# Mehrfachteilnahme
GET  /api/v1/eegs/{eegID}/participations               — list participations
POST /api/v1/eegs/{eegID}/participations               — create participation
PUT  /api/v1/eegs/{eegID}/participations/{id}          — update participation
DELETE /api/v1/eegs/{eegID}/participations/{id}        — delete participation

# Accounting
GET  /api/v1/eegs/{eegID}/accounting/export?from=&to=&format=datev|xlsx

# Backup / Restore
GET  /api/v1/eegs/{eegID}/backup                       — download JSON snapshot
POST /api/v1/eegs/{eegID}/restore                      — restore from JSON file

# Search
GET  /api/v1/eegs/{eegID}/search?q=...                 — search members/meter-points/invoices

# Logo
GET  /api/v1/eegs/{eegID}/logo                         — serve EEG logo image
POST /api/v1/eegs/{eegID}/logo                         — upload logo

# E/A-Buchhaltung
GET  /api/v1/eegs/{eegID}/ea/settings                  — get EA settings (Steuernummer, UVA-Periodentyp)
PUT  /api/v1/eegs/{eegID}/ea/settings                  — update EA settings
GET  /api/v1/eegs/{eegID}/ea/konten                    — list Kontenplan
POST /api/v1/eegs/{eegID}/ea/konten                    — create Konto
PUT  /api/v1/eegs/{eegID}/ea/konten/{kontoID}          — update Konto
DELETE /api/v1/eegs/{eegID}/ea/konten/{kontoID}        — delete Konto
GET  /api/v1/eegs/{eegID}/ea/buchungen?jahr=&konto_id=&richtung=&bezahlt=&incl_deleted=  — list Buchungen; incl_deleted=true shows soft-deleted entries
POST /api/v1/eegs/{eegID}/ea/buchungen                 — create Buchung; body: {beleg_datum,zahlung_datum,konto_id,beschreibung,betrag_brutto,ust_code,richtung,gegenseite,notizen}
GET  /api/v1/eegs/{eegID}/ea/buchungen/{buchungID}     — get Buchung detail (incl. Belege, deleted_at/deleted_by)
PUT  /api/v1/eegs/{eegID}/ea/buchungen/{buchungID}     — update Buchung; optional body field `reason` recorded in changelog
DELETE /api/v1/eegs/{eegID}/ea/buchungen/{buchungID}   — soft-delete Buchung (manual only); sets deleted_at/deleted_by; optional body: {"reason":"..."}; changelog entry written
GET  /api/v1/eegs/{eegID}/ea/buchungen/{buchungID}/changelog — audit trail for one booking (BAO §131); returns []EABuchungChangelog ordered by changed_at ASC
GET  /api/v1/eegs/{eegID}/ea/changelog?von=&bis=&user=&operation=&limit=&offset=  — EEG-wide changelog; filters: date range, user UUID, operation (create|update|delete); default limit 200, max 500
GET  /api/v1/eegs/{eegID}/ea/buchungen/export?jahr=&konto_id=  — XLSX export
POST /api/v1/eegs/{eegID}/ea/belege                    — upload Beleg (multipart: datei + buchung_id)
GET  /api/v1/eegs/{eegID}/ea/belege/{belegID}          — download Beleg
DELETE /api/v1/eegs/{eegID}/ea/belege/{belegID}        — delete Beleg
GET  /api/v1/eegs/{eegID}/ea/saldenliste?jahr=         — balance list; returns []EASaldenlisteEintrag (flat array)
GET  /api/v1/eegs/{eegID}/ea/kontenblatt/{kontoID}?von=YYYY-MM-DD&bis=YYYY-MM-DD  — account sheet; returns {konto,eintraege,summe}
GET  /api/v1/eegs/{eegID}/ea/jahresabschluss?jahr=&format=xlsx  — annual statement; returns {jahr,total_einnahmen,total_ausgaben,ueberschuss,einnahmen[],ausgaben[]}
GET  /api/v1/eegs/{eegID}/ea/uva                       — list UVA periods
POST /api/v1/eegs/{eegID}/ea/uva                       — create UVA period
GET  /api/v1/eegs/{eegID}/ea/uva/{uvaID}/kennzahlen    — compute UVA Kennzahlen
PATCH /api/v1/eegs/{eegID}/ea/uva/{uvaID}/eingereicht  — mark UVA as submitted
GET  /api/v1/eegs/{eegID}/ea/uva/{uvaID}/export        — FinanzOnline XML export
GET  /api/v1/eegs/{eegID}/ea/erklaerungen/u1?jahr=     — U1 annual VAT summary
GET  /api/v1/eegs/{eegID}/ea/erklaerungen/k1?jahr=     — K1 corporate tax basis
GET  /api/v1/eegs/{eegID}/ea/import/preview?jahr=      — preview EEG invoices not yet imported
POST /api/v1/eegs/{eegID}/ea/import/rechnungen         — import selected EEG invoices as Buchungen; body: {invoice_ids:[]}
POST /api/v1/eegs/{eegID}/ea/bank/import               — import bank statement (multipart: datei, format=mt940|camt053)
GET  /api/v1/eegs/{eegID}/ea/bank/transaktionen?status=offen|ignoriert  — list bank transactions
POST /api/v1/eegs/{eegID}/ea/bank/match                — match transaction to Buchung; body: {transaktion_id,buchung_id}
DELETE /api/v1/eegs/{eegID}/ea/bank/transaktionen/{transaktionID}  — ignore bank transaction
```

## Web Pages
| Path | Purpose |
|------|---------|
| `/eegs/[eegId]/reports` | Energy analytics (year/month/day/15-min, per-member breakdown, CSV/XLSX export) |
| `/eegs/[eegId]/accounting` | Accounting export (DATEV CSV + XLSX) |
| `/eegs/[eegId]/onboarding` | Admin approval queue for new member applications |
| `/eegs/[eegId]/participations` | Mehrfachteilnahme CRUD |
| `/eegs/[eegId]/billing` | Billing runs management |
| `/eegs/[eegId]/eda` | EDA process list + Anmeldung/Abmeldung/Teilnahmefaktor actions |
| `/eegs/[eegId]/import` | Energy data import (XLSX) with coverage chart |
| `/eegs/[eegId]/tariffs` | Tariff schedule management |
| `/eegs/[eegId]/communications` | Bulk email campaigns (compose, member selection, history) |
| `/eegs/[eegId]/settings` | EEG configuration (address, logo, SEPA, billing, DATEV, EDA, auto-billing) |
| `/eegs/[eegId]/ea` | E/A-Buchhaltung dashboard (KPI cards, open UVA alerts, nav grid) |
| `/eegs/[eegId]/ea/buchungen` | Journal (all Buchungen, year/konto/richtung filter, XLSX export) |
| `/eegs/[eegId]/ea/buchungen/neu` | New manual Buchung form |
| `/eegs/[eegId]/ea/buchungen/[buchungId]` | Buchung detail + Beleg upload/delete + per-booking changelog (BAO §131); delete requires reason |
| `/eegs/[eegId]/ea/changelog` | EEG-wide audit log — all Buchung mutations (create/update/delete) with old/new values |
| `/eegs/[eegId]/ea/konten` | Kontenplan CRUD |
| `/eegs/[eegId]/ea/saldenliste` | Balance list by account, year picker, XLSX export |
| `/eegs/[eegId]/ea/kontenblatt/[id]` | Account sheet (all Buchungen on one Konto, running balance) |
| `/eegs/[eegId]/ea/jahresabschluss` | Annual income/expense statement, XLSX export |
| `/eegs/[eegId]/ea/uva` | UVA periods + Kennzahlen + FinanzOnline XML export |
| `/eegs/[eegId]/ea/erklaerungen` | U1 / K1 annual tax declaration data |
| `/eegs/[eegId]/ea/import` | Import EEG invoices as Buchungen (preview + confirm) |
| `/eegs/[eegId]/ea/bank` | Bank statement import (MT940/CAMT.053) + transaction matching |
| `/eegs/[eegId]/ea/settings` | E/A settings (Steuernummer, UVA-Periodentyp) |
| `/onboarding/[eegId]` | Public member self-registration form |
| `/portal/[token]` | Magic-link entry point for member portal |
| `/portal/dashboard` | Member self-service dashboard (energy + invoices) |
| `/admin/users` | User administration (admin only) |

## EDA Worker
Separate binary (`api/cmd/worker/`) activated via `docker compose --profile eda up`.

### Transport modes
| Mode | How to activate | Use case |
|------|----------------|---------|
| `MAIL` | `EDA_TRANSPORT=MAIL` (default) | Production — IMAP polling + SMTP send (per-EEG credentials from DB) |
| `FILE` | `EDA_TRANSPORT=FILE` | Local testing — reads/writes XML files |
| `PONTON` | `EDA_TRANSPORT=PONTON` | **NOT production-ready** — skeleton exists in `transport/ponton.go` but the Send/Poll logic does not match the real Ponton XP API and has never been tested against a live Ponton instance |

### Ponton X/P Migration Plan

**Trigger:** E-Mail-Gateway erlaubt max. 2.500 Nachrichten/Monat (ein- + ausgehend). Ab ~40 Mitgliedern mit je 2 Zählpunkten wird diese Grenze erreicht. Danach ist KEP-Betreiber-Anbindung via Ponton X/P Pflicht (lt. EDA GmbH Regelwerk).

**Was Ponton X/P ist:** Keine REST-API, sondern eine eigenständige Java-Anwendung (Messenger) die lokal läuft und über AS4 (HTTPS) mit dem EDA-Netzwerk kommuniziert. Die Integration erfolgt über den **BWA (Binary Webservice Adapter)**:

```
Unser Worker → [HTTP POST an BWA] → Ponton Messenger → [AS4/HTTPS via SIA] → edanet
edanet       → SIA               → Ponton Messenger → [HTTP POST Callback] → Unser API /eda/inbound
```

**Kein Polling mehr nötig** — Ponton pushed Nachrichten aktiv an einen Callback-Endpunkt.

#### Phase 1 — Voraussetzungen (manuell, kein Code)

| Schritt | Was | Bei wem |
|---------|-----|---------|
| 1 | EDA-Plattformvertrag unterschreiben | EDA GmbH |
| 2 | Service-Desk bei Ponton registrieren | ponton.de/support |
| 3 | AT-Code(s) der EEG(s) bestätigen | EDA GmbH |
| 4 | Partner-Zertifikat beantragen (AT-Code als CN) | Ponton Service Desk |
| 5 | Öffentliche HTTPS-URL für eingehende Nachrichten bereitstellen (Listener) | Eigene Infra |
| 6 | Firewall: ausgehend HTTPS zu SIA `217.x.x.x`, eingehend HTTPS Listener-Port | Server |

Eingehende URL-Format: `https://<domain>:9002/datenplattform/<FirmenName>` — ein Name deckt alle AT-Codes ab.

#### Phase 2 — Ponton Messenger als Docker-Service

Neuer Service in `docker-compose.yaml` (Ponton-Image wird nach Vertragsabschluss geliefert, kein öffentliches Hub-Image):

```yaml
ponton-messenger:
  image: ponton/xp-messenger:latest
  ports:
    - "8181:8080"   # BWA / interne Adapter (nur intern erreichbar)
    - "8443:8443"   # Listener (öffentlich, via Reverse-Proxy)
  volumes:
    - ponton-data:/opt/ponton/data   # DB, Zertifikate, Konfiguration
  environment:
    - PONTON_DB_URL=jdbc:postgresql://eegabrechnung-postgres:5432/ponton
```

Ponton braucht eine **echte DB** (kein HSQLDB im Prod-Betrieb) — unsere bestehende Postgres-Instanz reicht, einfach eine zweite DB `ponton` anlegen. Erstkonfiguration erfolgt über die Ponton Web-GUI.

Ponton GUI-Konfiguration (einmalig):
- Lokales Partner-Profil pro AT-Code anlegen (Backend Partner ID = AT-Code)
- Remote-Profil "Wechselplattform / EnergyLink" aus Partner-Registrierung laden
- Partner-Vereinbarung zwischen lokalem + remote Profil erstellen
- EDA-Schemaset und EDA-Adapter installieren (aus Ponton Release Notes)
- BWA-Backend-URL auf unseren `/api/internal/eda/inbound` Endpunkt zeigen lassen
- Profil in Partner-Registrierung hochladen (damit SIA uns kennt)

#### Phase 3 — ponton.go neu schreiben

Das bestehende Skeleton in `api/internal/eda/transport/ponton.go` ist falsch (falsche Header, falsche Endpunkte). Es muss das BWA 3.0 Protokoll implementieren:

**Outbound (Send):**
```go
POST http://ponton-messenger:8080/xp/bwa/eda
Content-Type: application/xml
X-ebms-from: <unser AT-Code>      // z.B. RC105970
X-ebms-to:   <Ziel AT-Code>       // Netzbetreiber
X-ebms-service: ...               // EDA-Prozess-Typ
X-ebms-action:  ...
Body: raw XML (CMRequest/ECMPList/CPRequest/etc.)
```

Genaue Header-Namen stehen in der Ponton End-User-Dokumentation (verfügbar nach Ponton-Login).

**Inbound (empfangen):** Ponton POSTs an unseren neuen Endpunkt:
```go
// Neuer Endpunkt in api/internal/server/server.go registrieren:
POST /api/internal/eda/inbound
// Auth: shared secret (nur von Ponton-Messenger erreichbar, nicht öffentlich)
// Body: raw XML → direkt in denselben Worker-Verarbeitungspfad
```

ACKs: Ponton X/P erledigt AS4-Acknowledgements intern — `SendAck` ist weiterhin No-Op.

#### Phase 4 — Worker-Änderungen

Minimal (Worker ist schon transport-agnostisch):
- Env-Var `EDA_PONTON_BWA_URL` hinzufügen (z.B. `http://ponton-messenger:8080/xp/bwa/eda`)
- `pollLoop()` im Worker bei PONTON-Modus deaktivieren (kein Polling mehr nötig)
- Inbound-Endpoint in `server.go` registrieren, Messages direkt in `jobs`-Tabelle schreiben

#### Phase 5 — Multi-Tenant

Ein Ponton-Messenger für alle EEGs. Ponton unterstützt mehrere lokale Partner-Profile (ein Profil pro AT-Code). Im BWA-Aufruf gibt `X-ebms-from` den richtigen AT-Code an.

Inbound-Routing: Ponton liefert im Callback den Empfänger-AT-Code → Worker mapt AT-Code auf EEG. Dafür braucht `eegs`-Tabelle ein Feld `eda_marktpartner_id` (bereits vorhanden als `eda_netzbetreiber_id` / `eda_marktpartner_id`).

#### Phase 6 — Migration

| Zustand | Transport |
|---------|-----------|
| EEG < ~2.000 Nachrichten/Monat | MAIL (wie heute, keine Änderung) |
| EEG mit KEP-Vertrag / über Limit | PONTON |

Umschalten per `EDA_TRANSPORT`-Env-Var auf Worker-Ebene (heute schon möglich). Für echtes per-EEG-Routing wäre eine `eda_transport_mode`-Spalte in `eegs` der saubere nächste Schritt.

#### Zeitplan / Abhängigkeiten

```
Phase 1 (Verträge, 2-4 Wochen)
  → Phase 2 (Ponton Docker + Testsystem-Konfiguration, 1-2 Tage)
  → Phase 3+4 (Code: ponton.go + Inbound-Endpoint, ~1 Tag)
  → Integrationstest auf Testsystem (1-2 Wochen mit Ponton-Support)
  → Phase 5 (Multi-Tenant-Routing, ½ Tag)
  → Prod-Rollout
```


### FILE transport (local testing)
```bash
# Start worker with FILE transport
docker compose --profile eda run --rm -e EDA_TRANSPORT=FILE eda-worker

# Place inbound XML (e.g. CPDocument confirmation) here:
#   test/eda-inbox/<file>.xml
# Processed files moved to:
#   test/eda-inbox/processed/
# Outbound XML written to:
#   test/eda-outbox/<timestamp>_<process>.xml
```

### EDA API endpoints (all require Bearer token)
```
GET  /api/v1/eegs/{eegID}/eda/processes                  — list all processes
POST /api/v1/eegs/{eegID}/eda/anmeldung                  — register meter point (EC_REQ_ONL)
POST /api/v1/eegs/{eegID}/eda/widerruf                   — revoke consent (CM_REV_SP); requires stored consent_id on meter point
POST /api/v1/eegs/{eegID}/eda/teilnahmefaktor            — change participation factor (EC_PRTFACT_CHG)
POST /api/v1/eegs/{eegID}/eda/zaehlerstandsgang          — request historical meter data (EC_REQ_PT)
GET  /api/v1/eegs/{eegID}/eda/messages                   — list EDA messages (scoped by eeg_id)
GET  /api/v1/eegs/{eegID}/eda/messages/{id}/xml          — download raw XML for a message
GET  /api/v1/eegs/{eegID}/eda/errors                     — list dead-letter errors
```

Zaehlerstandsgang request body:
```json
{ "zaehlpunkt": "AT...", "date_from": "2026-03-01", "date_to": "2026-03-31" }
```

### Key EDA files
- `api/internal/eda/transport/file.go` — FILE transport (inbox/outbox directories)
- `api/internal/eda/transport/mail.go` — MAIL transport (IMAP polling + SMTP send); edanetProzessID map
- `api/internal/eda/xml/cprequest_builder.go` — builds outbound CPRequest XML (EC_REQ_PT, EC_PODLIST)
- `api/internal/eda/xml/cmrequest_builder.go` — builds outbound CMRequest XML (EC_REQ_ONL)
- `api/internal/eda/xml/ecmplist_builder.go` — builds outbound ECMPList XML (EC_PRTFACT_CHG)
- `api/internal/eda/xml/cmrevoke_builder.go` — builds outbound CMRevoke XML (CM_REV_SP)
- `api/internal/eda/xml/cpdocument_parser.go` — parses inbound CPDocument confirmations
- `api/internal/handler/eda.go` — HTTP handlers for Anmeldung/Abmeldung/Teilnahmefaktor
- `api/internal/repository/eda_process.go` — EDA process CRUD + lifecycle queries
- `api/internal/repository/job.go` — `EnqueueEDA` inserts job into `jobs` table
- `api/internal/eda/worker.go` — poll/send loop; processes outbound jobs + inbound messages

### EDA XML Namespace Rules
Critical: namespace prefixes in XML struct tags must exactly match the schema.

| XML document | Applies to | Rule |
|---|---|---|
| ECMPList 01.10 | ALL elements in MarketParticipantDirectory AND ProcessDirectory | `cp:` prefix — including `MessageId`, `ConversationId`, `MeteringPoint` |
| CMRequest 01.30 | document elements `cp:`, RoutingHeader sub-elements `ct:` | See cmrequest_builder.go |
| CPRequest 01.12 | document elements `cp:`, RoutingHeader sub-elements `ct:` | See cprequest_builder.go |

**ECMPList gotcha**: `MessageId`, `ConversationId`, `MeteringPoint` in ProcessDirectory use `cp:` (NOT `ct:`). Using `ct:` causes edanet rejection: "Invalid content was found starting with element '{common/types:MessageId}'. One of '{ecmplist/01p10:MessageId}' is expected."

### CMRequest Schema Versions
- **01.30** active on edanet from **12.04.2026** (`EC_REQ_ONL_02.30`, `EC_REQ_OFF_02.20`)
- Before that date edanet returns "No activated XML Schema for MessageType:null Version:null" — not a bug, just timing
- Code in `cmrequest_builder.go` is already on 01.30

### edanetProzessID map (email Subject line)
Process type → edanet Subject field (in `api/internal/eda/transport/mail.go`):
```
EC_PRTFACT_CHG → EC_PRTFACT_CHANGE_01.00
EC_REQ_ONL     → EC_REQ_ONL_02.30
EC_REQ_PT      → CR_REQ_PT_04.10
EC_PODLIST     → EC_PODLIST_01.00
CM_REV_SP      → CM_REV_SP_01.00
```

### IMAP / Worker Operational Notes

**BODY.PEEK[] is mandatory**: The worker fetches with `{Peek: true}` in `FetchItemBodySection`. Without it, IMAP auto-marks messages as `\Seen` on fetch. If the worker then crashes or times out before explicitly marking `\Seen`, those messages are silently lost — next poll finds 0 unseen messages and processes stay stuck at "sent" indefinitely.

**Test worker job stealing**: `eegabrechnung-eda-worker-test` (FILE transport) shares the same PostgreSQL database as the production MAIL worker. `FOR UPDATE SKIP LOCKED` means whichever worker runs first claims the job. Symptom: processes marked "sent" with empty Subject, no SMTP delivery, no edanet response. **Always stop the test worker before using the MAIL worker:**
```bash
docker stop eegabrechnung-eegabrechnung-eda-worker-test-1
```

**Worker health monitoring**: The `eda_worker_status` table (single row) is the ground truth for worker health:
- `last_poll_at` — when the last poll cycle completed
- `last_error` — last error string (or null if healthy)

**Widerruf (CM_REV_SP) request JSON field**: The direct Widerruf HTTP handler uses `valid_from` (not `date_from`) in the request body:
```json
{ "meter_point_id": "...", "valid_from": "2026-05-01" }
```

**Anmeldung valid_from rules**: EC_REQ_ONL `valid_from` must be **at least tomorrow and at most 30 days in the future** (validated in `handler/eda.go`). Onboarding convert auto-clamps stored `beitritts_datum` to this range. Same rule shown in UI with `min`/`max` on date pickers.

**Austritt/Widerruf valid_from rules**: CM_REV_SP `valid_from` must be **at least tomorrow and at most 30 Austrian working days in the future** (same validation in `handler/member.go` austritt handler).

**Post-confirmation actions (ABSCHLUSS_ECON)**:
- EC_REQ_ONL confirmed → sets `meter_point.registriert_seit` from ECMPList DateFrom; stores `consent_id` from ECMPList; sends confirmation email to member (via EEG SMTP); activates onboarding + member
- CM_REV_SP confirmed (CM_REV_CUS / CM_REV_IMP) → sets `meter_point.abgemeldet_am` from ECMPList ConsentEnd date

**EDA status on meter points** (`meterPointShort` in API response): `registriert_seit`, `abgemeldet_am`, `anmeldung_status`, `abmeldung_status` — derived from `eda_processes`; shown as status badge on member detail page.

## Pricing Logic
Prices are set **only** in Tarifpläne (tariff schedules). 
 During billing:
1. Load active tariff schedule entries overlapping the billing period
2. Blend entries weighted by overlap duration
3. Uncovered fractions fall back to `eeg.energy_price` / `eeg.producer_price` (DB fields, not exposed in UI)

## API Testing
```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8101/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@eegwn.at","password":"<siehe secrets/admin-credentials.txt>"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

# Use token
curl -H "Authorization: Bearer $TOKEN" http://localhost:8101/api/v1/eegs
```

## Rebuild After Changes
```bash
# API only
docker compose build eegabrechnung-api && docker compose up -d eegabrechnung-api

# Web only
docker compose build eegabrechnung-web && docker compose up -d eegabrechnung-web

# EDA worker only (SEPARATE Dockerfile — rebuilding api does NOT rebuild worker!)
docker compose build eegabrechnung-eda-worker && docker compose up -d --force-recreate eegabrechnung-eda-worker

# Both
docker compose build && docker compose up -d
```

## Playwright Testing (on server)
```bash
cd /tmp/pw_test   # playwright is installed here
node my_test.mjs
```

## Test Data
- EEG: "Sonnenstrom Mustertal" (ID: 5d0151e8-8714-4605-9f20-70ec5d5d5b46)
- 3 members: Hans Mustermann (CONSUMER), Maria Sonnleitner (PROSUMER), Biobauernhof Grünwald GmbH (PRODUCER)
- Each member has meter points assigned

## Key env vars
| Var | Used by | Purpose |
|-----|---------|---------|
| `JWT_SECRET` | api + web | Shared secret for HS256 JWT signing/verification |
| `CREDENTIAL_ENCRYPTION_KEY` | api + eda-worker | AES-256-GCM key for per-EEG credentials (base64, 32 bytes). **Required.** Generate: `openssl rand -base64 32` |
| `API_INTERNAL_URL` | web | Internal Docker URL for server-side API calls (`http://eegabrechnung-api:8080`) |
| `NEXTAUTH_SECRET` / `AUTH_SECRET` | web | next-auth session encryption key |
| `EDA_TRANSPORT` | eda-worker | Transport mode: `MAIL` (default), `PONTON`, or `FILE` |
| `EDA_INBOX_DIR` | eda-worker | Directory to read inbound XML from (FILE mode, default `./test/eda-inbox`) |
| `EDA_OUTBOX_DIR` | eda-worker | Directory to write outbound XML to (FILE mode, default `./test/eda-outbox`) |

## Per-EEG Credential Architecture
All EDA and invoice SMTP credentials are stored **per EEG** in the `eegs` table, encrypted with AES-256-GCM. There is **no global env-var fallback** — EEGs without credentials silently skip sending.

### DB columns (migration 043)
| Column | Purpose |
|--------|---------|
| `eda_imap_host`, `eda_imap_user`, `eda_imap_password_enc` | EDA IMAP mailbox (edanet.at IMAP) |
| `eda_smtp_host`, `eda_smtp_user`, `eda_smtp_password_enc`, `eda_smtp_from` | EDA SMTP send (edanet.at SMTP) |
| `smtp_host`, `smtp_user`, `smtp_password_enc`, `smtp_from` | Invoice / campaign email (resend.com etc.) |

### Code locations
- `api/internal/crypto/credentials.go` — `Encrypt(key, plaintext)` / `Decrypt(key, ciphertext)` (base64(nonce || ct))
- `api/internal/repository/eeg.go` — `NewEEGRepository(db, encKey)`, decrypts on read, encrypts on write, `ListEEGsWithIMAPCredentials()`
- `api/internal/eda/worker.go` — `receiveInboundPerEEG()` iterates EEGs with IMAP creds; `sendJob()` creates per-EEG MailTransport
- All handlers build `invoice.SMTPConfig` inline from `eeg.SMTPHost/User/Password/From` at request time

### Set credentials via API (PUT /api/v1/eegs/{eegID})
Include the credential fields in the PUT body. Passwords are only written when non-empty (empty = keep existing encrypted value):
```json
{
  "eda_imap_host": "mail.edanet.at:993",
  "eda_imap_user": "rc105970",
  "eda_imap_password": "...",
  "eda_smtp_host": "mail.edanet.at:465",
  "eda_smtp_user": "rc105970",
  "eda_smtp_password": "...",
  "eda_smtp_from": "rc105970@edanet.at",
  "smtp_host": "smtp.resend.com:587",
  "smtp_user": "resend",
  "smtp_password": "re_...",
  "smtp_from": "kontakt@eegwn.at"
}
```

## Local Backup
Automated daily backup with BorgBackup (7-day retention). Set up 2026-04-16.

- **Script**: `scripts/backup-eeg.sh` — pg_dump (custom format) + Docker volumes (invoices, documents) → Borg archive
- **Borg repo**: `/mnt/HC_Volume_103451728/backups/borg-eeg/` (no encryption, local only)
- **Cron**: daily 02:00 UTC (`crontab -l` as current user)
- **Log**: `/mnt/HC_Volume_103451728/backups/backup.log`
- **Retention**: `--keep-daily=7`; dedup means ~60–80 MB total for 7 days

```bash
# List archives
borg list /mnt/HC_Volume_103451728/backups/borg-eeg

# Restore DB from a specific archive
borg extract /mnt/HC_Volume_103451728/backups/borg-eeg::eeg-2026-04-16T0200 --strip-components 3 db.dump
docker exec -i eegabrechnung-eegabrechnung-postgres-1 \
  pg_restore -U eegabrechnung -d eegabrechnung --clean < db.dump
```
