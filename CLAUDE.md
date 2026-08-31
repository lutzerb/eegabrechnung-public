# eegabrechnung — Claude Instructions

## Project Overview
Austrian EEG (Energiegemeinschaft) billing platform.

- **api/** — Go REST API (chi/v5, pgx/v5, golang-migrate, golang-jwt/v5)
- **web/** — Next.js 16 App Router frontend (Tailwind CSS, next-auth v5 beta)
- **docker-compose.yaml** — full local stack (3 containers: postgres, api, web); optional forecast service via `--profile forecast`

## Running the Stack
```bash
cd /mnt/HC_Volume_103451728/eegabrechnung
docker compose up -d
```
- Web: http://localhost:3001
- API: http://localhost:8101
- Postgres: localhost:26433

**Optional: ML Forecast Service** (benötigt ≥200 Stundenwerte historische Messdaten pro EEG):
```bash
docker compose --profile forecast up -d
```
- Forecast Service: intern auf Port 8200 (nicht nach außen exponiert)
- Aktiviert den "7-Tage-Prognose"-Toggle auf der Reports-Seite
- Ohne dieses Profil zeigt der Toggle "Prognose-Service nicht erreichbar"

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
| 067_email_log | `email_log` table (eeg_id, email_type, to_address, subject, status sent\|failed, error_msg, member_id, invoice_id) — every outbound email (invoices, campaigns, onboarding, portal, EDA notices) is logged via `invoice.SendLogged()` |
| 068_email_verify_reminder | reminder_sent_at on onboarding_email_verifications; used by hourly background check to send 72h follow-up to people who entered their email but never completed the form |
| 069_gemeinschaft_typ | `gemeinschaft_typ` on eegs (EEG\|GEA\|BEG, default 'EEG') — selects Marktpartner-ID prefix (RC/GC/CC) and whether Netzbetreiber is required at creation (not required for BEG) |
| 070_zaehlpunkt_recycling | Drops global `UNIQUE(zaehlpunkt)` on meter_points; replaces with partial unique index `(eeg_id, zaehlpunkt) WHERE abgemeldet_am IS NULL` — allows same Zählpunkt to be reused by a new member after the previous one deregistered (tenant change / Mieterwechsel); historical readings stay on the old UUID row |
| 071_eda_process_response_info | `response_codes` (text[]), `meter_owner_name`, `portal_approval_url`, `customer_notified_at` on eda_processes — captures ANTWORT_ECON response codes; drives the §16e ElWOG Smartmeter-Benachrichtigung email (code 182 = not remotely readable, NB must install within 2 months) |
| 072_bank_transaktion_dedup | Partial unique index `(eeg_id, buchungsdatum, betrag, referenz) WHERE referenz <> ''` on ea_banktransaktionen — prevents duplicate rows when re-importing overlapping MT940/CAMT.053 statement periods |
| 073_bank_auftraggeber_text | Widens `ea_banktransaktionen.auftraggeber_empfaenger` to unbounded TEXT (some bank exports exceed the previous varchar limit) |
| 074_bank_dedup_no_ref | Secondary dedup index `(eeg_id, buchungsdatum, betrag, verwendungszweck) WHERE referenz = ''` on ea_banktransaktionen — covers bank entries without a reference (e.g. Raiffeisen NOTPROVIDED charges) |
| 075_invoice_paid_at | `paid_at` (date) on invoices — set by the CAMT.053 payment-matching import (`POST .../billing/camt053`) or manually during EA-Rechnungsimport |
| 076_member_tariffs | `member_id` (nullable) + 5 `*_override` columns on tariff_schedules ("Individualtarif" — per-member pricing overrides, independent one-active-per-scope indexes for EEG-default vs. per-member); `zaehlpunkts_gebuehr_eur` on eegs (fee × active meter points per member, unlike the flat `meter_fee_eur`) |
| 077_meter_point_registration_periods | `meter_point_registration_periods` table — full Anmeldung/Abmeldung history per Zählpunkt, keyed by `(eeg_id, zaehlpunkt)` (not `meter_point_id`) so history survives Zählpunkt-recycling; partial unique index enforces max one open period per Zählpunkt; backfilled from existing `meter_points.registriert_seit`/`abgemeldet_am` |
| 078_eda_message_zaehlpunkt | `zaehlpunkt` (text, DEFAULT '') + partial index on eda_messages — makes the EDA message log searchable by Zählpunkt; backfilled by extracting all `<MeteringPoint>` values from the stored XML payloads (list responses get all values space-joined) |
| 079_meter_point_direction_mismatch | `direction_mismatch_notified_at` on meter_points — dedup flag for the automatic operator email when an incoming CR_MSG OBIS schema doesn't match the stored Energierichtung (e.g. swapped consumption/generation Zählpunkte of a prosumer) |
| 080_invoice_payment_notice_mode | `invoice_payment_notice_mode` (text, default `sepa_lastschrift`) on eegs — controls the "Zahlungshinweis" paragraph on consumer invoices/emails: `sepa_lastschrift` (default, unchanged text) \| `ueberweisung` (shows EEG's own IBAN/BIC for manual bank transfer) \| `none` (section omitted entirely) |
| 081_eda_dis_model | `eda_dis_model` (text, default `D`) on eegs — the ECDisModel (statisch `S` \| dynamisch `D`) declared to the Netzbetreiber, decided once for the whole community instead of per EDA action; set in EEG-Einstellungen (EDA-Tab), used as the fixed value for Online-Anmeldung (EC_REQ_ONL) and Teilnahmefaktor-Änderung (EC_PRTFACT_CHG) — no more manual per-action dropdown |
| 082_member_sepa_mandate_history | `member_sepa_mandate_history` table (member_id, eeg_id, iban, signed_at, signed_ip, signed_text, archived_at, reason) — archives a member's SEPA mandate snapshot whenever their IBAN changes, so past mandates stay retrievable for audit purposes. Written by both the admin IBAN edit (reason `iban_change_admin`, signature cleared afterwards) and the member-portal self-service IBAN change (reason `iban_change_portal`, freshly signed) |
| 083_fee_billing_mode | `fee_billing_mode` (text, default `per_month`) on eegs — Fixgebühren (meter_fee, participation_fee, Zählpunktsgebühr) pro angefangenem Kalendermonat des Abrechnungszeitraums (`per_month`) oder einmal pro Abrechnungslauf (`per_invoice`); Radio-Group im Rechnungen-Tab der EEG-Einstellungen |
| 084_billing_run_scope | `billing_type` (default 'all') + `member_ids` (uuid[], NULL = alle) on billing_runs — Overlap-Prüfung ist scope-bewusst: consumption_only/production_only bzw. disjunkte Mitglieds-Teilmengen über denselben Zeitraum kollidieren nicht mehr |
| 085_member_email_change_verifications | `member_email_change_verifications` table (new_email, token, expires_at, verified_at; partial unique index = max one pending request per member) — member-portal self-service email change; confirmation link is sent to the NEW address and must be clicked before the member row is updated |
| 086_energy_imbalance_threshold | `energy_imbalance_threshold_promille` (numeric, default 1) on eegs — tolerance (‰ of the larger total) for the non-blocking Bezug/Einspeisung imbalance warning computed on full billing runs |
| 087_rename_ec_req_pt | Renames EC_REQ_PT → CR_REQ_PT (canonical ebutilities.at process name) in eda_processes.process_type, eda_messages.message_type and pending jobs (type + payload `process` field); mirrors migration 054 |
| 088_rename_kz057_to_kz032 | Renames `ea_uva_perioden.kz_057` → `kz_032` — Reverse-Charge-Steuerschuld (§19 Abs. 1 zweiter Satz UStG) gehört im FinanzOnline U30-Formular in Kennzahl 032, nicht 057; XML-Export-Feldreihenfolge in `VERSTEUERT` korrigiert (KZ032 steht laut echter BMF-XSD hinter KZ044, nicht neben KZ056/KZ057) |
| 089_uva_kz016 | `kz_016` on ea_uva_perioden — Kleinunternehmer-Umsätze (§ 6 Abs. 1 Z 27 UStG) müssen in KZ 016 (Teilmenge von KZ 000) gemeldet werden, auch in einer UVA, die nur wegen einer Reverse-Charge-Steuerschuld (KZ 032) abgegeben wird; zuvor wurden KU-Umsätze in einer RC-only-UVA gar nicht gemeldet. XML-Export: neues `STEUERFREI`-Element (nur `KZ016`) zwischen `KZ000` und `VERSTEUERT` in `LIEFERUNGEN_LEISTUNGEN_EIGENVERBRAUCH`, Reihenfolge gegen echte BMF-XSD (`u30.xsd`/`jahreserklaerungen.xsd`) validiert |
| 095_eeg_referral_options | `referral_options` (TEXT[], default `{Empfehlung von einem Mitglied,Internet,Zeitung}`) on eegs — per-EEG configurable list of "Wie sind Sie auf uns aufmerksam geworden?" options offered on the public onboarding form; admin-editable in EEG-Einstellungen (Onboarding-Tab); "Sonstiges" with free text is always additionally offered by the frontend, not stored here |
| 096_onboarding_referral_source | `referral_source` + `referral_source_note` (text, default '') on onboarding_requests — optional applicant answer to the referral-source question; onboarding-only metadata, not carried over to the `members` row on conversion (same as `phone`) |
| 099_eeg_display_name | `display_name` (text, default '') on eegs — optional alias shown everywhere except legally binding documents (invoice Rechnungssteller block, SEPA XML/mandate, onboarding declaration), which always use `name`; empty = fall back to `name` (`domain.EEG.DisplayNameOrName()`) |
| 112_service_fee_per_kwh | `servicegebuehr_bezug_ct_kwh` + `servicegebuehr_einspeisung_ct_kwh` (NUMERIC, default 0) on eegs — optional per-kWh service fee shown as its own invoice line, separate from `energy_price`/`producer_price`; `servicegebuehr_bezug_ct_kwh_override` + `servicegebuehr_einspeisung_ct_kwh_override` (nullable) on tariff_schedules for per-member Individualtarif overrides (nil = EEG default, 0 = exempt) |
| 113_invoice_show_zero_fees_per_type | Splits the single `invoice_show_zero_fees` flag into four independently selectable flags — `invoice_show_zero_fee_fixgebuehr`, `invoice_show_zero_fee_zaehlpunktsgebuehr`, `invoice_show_zero_fee_servicegebuehr_bezug`, `invoice_show_zero_fee_servicegebuehr_einspeisung` — one per fee-type PDF line item; the old flag's value is carried over via column rename (Fixgebühr) plus backfill (the other three), so an EEG that had it enabled keeps showing all four at 0,00 € until narrowed down |

## Energy Unit Convention
**All energy values throughout the codebase are stored and transmitted in kWh**, despite column/field names using the `wh_` prefix (e.g. `wh_total`, `wh_self`, `wh_community`). This naming is a historical artifact — do NOT divide these values by 1000 when displaying as kWh. The `fmtKwh()` helper in `web/components/energy-charts.tsx` is the reference implementation: it displays values directly as kWh and only converts to MWh when the value exceeds 100 000.

## Key Features
- **Billing runs**: group invoices per billing operation; overlap detection (409 on double-billing); member/type filter; force override; draft→finalized workflow; storno PDF on cancellation
- **Auto-billing scheduler**: daily check at 06:00 Vienna time; checks `auto_billing_enabled` EEGs; data completeness check via `MissingIntervals()`; creates draft run + sends email; skips on overlap or recent run (< 20 days); `billing/scheduler.go`. Meter points with a pending/sent/first_confirmed EC_REQ_ONL process are excluded from the completeness check — their `registriert_seit` is set at onboarding-convert time but data won't arrive until the Netzbetreiber confirms.
- **SEPA Rücklastschriften**: `PATCH .../invoices/{id}/sepa-return` sets/clears manual return (reason, note, date); `GET .../invoices?sepa_returned=true` filters; `POST .../sepa/camt054` imports CAMT.054 XML (matches by EndToEndId=invoice UUID); `sepa/camt054.go` namespace-agnostic parser; dashboard alert; red badge in billing page
- **Tariff schedules**: time-series pricing (annual/monthly/daily/15-min granularity); one active schedule per EEG; weighted fallback to flat EEG prices for uncovered periods
- **Credit notes**: VAT-liable producers receive Gutschriften instead of negative invoices; document_type on invoices distinguishes invoice vs credit_note
- **SEPA files**: pain.001 (credit transfers) and pain.008 (direct debits) generation
- **Import overlap handling**: preview endpoint compares XLSX rows against DB; overwrite/skip/cancel choice; source tracking (xlsx|eda); quality filtering (L0-L2 billed, L3 excluded)
- **BEG-Mehrfach-Netzbetreiber Energiedaten-Import**: EDA-Portal-Exports für eine BEG, die mehrere Netzbetreiber umfasst, enthalten neben dem kombinierten `Energiedaten`-Sheet zusätzliche `Energiedaten_<NB-Kennung>`-Sheets (z.B. `Energiedaten_AT008100`, `Energiedaten_AT008000`) — `importer.ParseEnergieDaten` liest alle davon und merged sie dedupliziert nach (Zählpunkt, Zeitstempel); ein normaler Einzel-NB-Export mit nur einem `Energiedaten`-Sheet verhält sich unverändert
- **Data coverage timeline**: per-day reading coverage chart on import page; auto-refreshes after import
- **Energy reports**: per-member and EEG-wide analytics; granularity: year/month/day/15-min; CSV/XLSX export; kWh display (switches to MWh above 100 000 kWh)
- **Accounting export**: DATEV Buchungsstapel CSV + XLSX; configurable GL accounts; VAT breakdown stored at billing time
- **User administration**: role-based access (admin/user); per-user EEG assignments
- **Member lifecycle**: status (ACTIVE/INACTIVE); beitritt_datum/austritt_datum; configurable invoice number start
- **Austritt-Workflow**: `POST .../members/{id}/austritt` sets INACTIVE + austritt_datum, enqueues CM_REV_SP (Widerruf) for all active meter points with a stored consent_id; idempotent — skips ZPs with pending/sent CM_REV_SP; EDA only triggered when EEG has credentials + not demo
- **Meter point delete**: button on member detail page; only allowed when no active EDA processes pending; sends CM_REV_SP (Widerruf) if meter point has consent_id
- **Zählpunkt recycling (Mieterwechsel)**: same Zählpunkt string can be reused after a member leaves — set `abgemeldet_am` on the old row (manually via API/UI for non-EDA EEGs, or automatically by EDA worker after CM_REV_SP confirmation), then create the Zählpunkt for the new member. The partial unique index `(eeg_id, zaehlpunkt) WHERE abgemeldet_am IS NULL` enforces that only one active row per Zählpunkt exists per EEG. Historical energy readings remain on the old UUID row and are unaffected.
- **Registrierungshistorie (Zählpunkt-Anmeldung/Abmeldung)**: `meter_point_registration_periods` (migration 077) records every Anmeldung/Abmeldung period per Zählpunkt, keyed by `(eeg_id, zaehlpunkt)` so history survives both Zählpunkt-recycling and re-registration of the same row; `GET .../meter-points/{meterPointID}/history` returns the ordered period list; shown as a vertical timeline on the meter point edit page. Opened/closed automatically from `UpdateRegistriertSeit`/`UpdateAbgemeldetAm` (EDA confirmations) and from manual `abgemeldet_am` edits (source `eda` vs `manual` vs `backfill`). **Unsolicited Netzbetreiber-side revokes** (e.g. a Zählpunkt-Lieferantenwechsel resetting EEG-participation consent, `MessageCode=AUFHEBUNG_CCMI`) are now processed automatically too: `processCMRevokeUnsolicited` (`worker.go`) falls back to matching the meter point directly by Zählpunkt + ConsentId (verified) when no outbound `CM_REV_SP` process exists to match against, closes the period, sets `abgemeldet_am`, tags the stored `eda_messages` row with the correct `eeg_id`, and — for the involuntary variants (`AUFHEBUNG_CCMS_IMP` / `AUFHEBUNG_CCMI`) — emails the operator, same as a matched CM_REV_SP confirmation. Previously such messages were received and parsed but silently dropped (logged a warning only) because the matching logic only handled revokes we ourselves had requested.
- **SEPA Mandat**: `GET .../members/{memberID}/sepa-mandat` generates SEPA direct debit mandate PDF; sepa_mandate_signed_at/ip/text captured during onboarding; sepa_pre_notification_days (default 14) configures collection date offset per SEPA Rulebook
- **SEPA Mandats-Historie (IBAN-Änderung)**: whenever a member's IBAN changes, the prior mandate snapshot (IBAN, signed_at/ip/text) is archived into `member_sepa_mandate_history` (migration 082) before being overwritten — old mandates stay retrievable for audit purposes. Two paths: (1) **Admin edits IBAN directly** (`PUT .../members/{memberID}`) — old mandate archived (`reason=iban_change_admin`), but the signature fields are then cleared on the member row so the mandate PDF shows "not yet confirmed" until the member re-confirms; (2) **Member self-service via portal** (`POST /api/v1/public/portal/sepa-mandate`, body `{iban, confirm:true}`) — old mandate archived (`reason=iban_change_portal`), new IBAN + freshly signed mandate (signed_at/signed_ip/signed_text) stored atomically, confirmation email sent to the member **and** a separate info-only notification to the EEG admin (`eeg.SMTPFrom`, email_type `sepa_mandate_changed_admin` — same "admin gets a copy" convention as EDA error/mismatch emails), so an IBAN change doesn't go unnoticed. `GET .../members/{memberID}/sepa-mandat/history` lists the archive (admin, newest first); shown as a timeline card ("Mandats-Historie") on the member detail page. Portal UI: "Profil" tab on `/portal/dashboard` shows the masked current IBAN + an "IBAN ändern" form with a mandatory consent checkbox.
- **Member email campaigns**: `POST .../communications` sends HTML email to selected members; field names: `subject`, `html_body`, `member_ids` (JSON array string — NOT `member_ids[]`); placeholder substitution ({{name}}, {{eeg_name}}, etc.); attachments; omitting `member_ids` sends to ALL active members of the EEG — **always set this field when targeting individuals**. curl example: `-F "member_ids=[\"uuid1\",\"uuid2\"]"`. `recipient_count` in the response is always 0 (updated async after send). `GET .../communications` lists history; `GET .../communications/{id}` campaign detail
- **Meter point notes**: free-text notes field on meter points; shown on member detail page
- **Onboarding portal**: public self-service registration form with magic-token email verification; admin approval queue; auto-creates member + meter points + EDA Anmeldung on convert. Hourly background job sends two kinds of 72h reminders (admin CC'd on both): (1) `eda_sent` requests where the member hasn't confirmed NB data access yet; (2) abandoned email verifications where someone entered their email but never submitted the form (`onboarding_email_verifications.reminder_sent_at` tracks dedup)
- **Member portal**: magic-link self-service dashboard for members; monthly energy breakdown; invoice list + PDF download (no password required)
- **Mehrfachteilnahme**: meter point participates in multiple EEGs simultaneously (Austrian EAG April 2024); factor + share type (GC/RC_R/RC_L/CC) + date range; source of truth for EDA
- **OeMAG market prices**: scraped from oem-ag.at; sync to producer/energy price
- **EDA process management**: Anmeldung (EC_REQ_ONL), Teilnahmefaktor (intern `EC_PRTFACT_CHG`, offiziell **EC_PRTFACT_CHANGE**), Widerruf (CM_REV_SP), Datenanforderung (CR_REQ_PT), Zählpunktliste (EC_PODLIST) via MaKo XML; process lifecycle tracking (pending→sent→first_confirmed→confirmed/completed/rejected/error); deadline tracking; duplicate-change prevention; eda_errors dead-letter table; eda_worker_status singleton
- **EEG settings**: address (strasse/plz/ort/uid_nummer for §11 UStG invoice block), logo, founding date, generation type on meter points (PV/Wind/Wasser)
- **Anzeigename (Alias)**: `display_name` (migration 099, optional, default '') lets an EEG present a different public-facing name than its legal `name` — e.g. when several BEG hang off the same GmbH and each wants its own brand name while the invoice issuer stays the shared legal entity. `domain.EEG.DisplayNameOrName()` resolves it (falls back to `name`) and is used everywhere the EEG name is just displayed (nav switcher, breadcrumbs, dashboard, portal, invoice/onboarding/campaign/auto-billing/gap-alert emails). **Always stays the legal `name`** — never the alias — on anything legally/financially binding: the Rechnungssteller block on Rechnung/Gutschrift/Storno PDFs (`invoice/pdf.go`, `invoice/pdf_theme.go`), SEPA XML creditor/debtor name (pain.001/pain.008/Finanzamtszahlung), the SEPA-Lastschriftmandat PDF's "Gläubiger-Bezeichnung", and the onboarding join declaration + SEPA mandate consent texts (`OnboardingForm.tsx`'s `legalName` prop, `PortalDashboardClient.tsx`). Editable in EEG-Einstellungen (Allgemein-Tab) under "Anzeigename (optional)"; the EEG list page (`/eegs`) additionally shows the legal name as a small second line under the alias, since several EEGs can share one legal entity while each alias stays unique.
- **Backup/Restore**: full or partial EEG snapshot export (JSON) and restore via transaction. `?sections=members,meter_points,tariff_schedules,billing_runs,invoices,participations,readings` on the export selects which of the 7 sections to include (omitted = all, unchanged default behavior); the file records the selection in `included`. Restore only deletes/replaces sections present in `included` (legacy files without `included` = full restore, as before) — for `members`/`meter_points`, a partial restore upserts (`ON CONFLICT (id) DO UPDATE`) instead of delete+insert, because their FK cascades (`members`→meter_points/invoices/tariff_schedules, `meter_points`→energy_readings/participations) would otherwise wipe non-included sections; other 5 sections are cascade-leaves and stay delete+insert, just conditional. UI: `web/components/backup-restore-section.tsx` has per-section checkboxes + a "Nur Stammdaten" preset (members/meter_points/tariff_schedules/participations)
- **Gemeinschaftstypen**: `gemeinschaft_typ` = EEG (§79 EAG, Marktpartner-ID `RC######`) | GEA (§16c ElWOG, `GC######`) | BEG (§16a ElWOG, `CC######`); selected as a 3-column grid at EEG creation (`web/app/eegs/new/page.tsx`); Netzbetreiber is optional for BEG (community can span multiple grid operators), required for EEG/GEA. **Multi-Netzbetreiber EDA routing for BEG is implemented**: per-ZP actions (Anmeldung, Teilnahmefaktor, CR_REQ_PT, Widerruf) derive the target NB from the Zählpunkt prefix (`zaehlpunkt[:8]`) instead of the EEG-wide `eda_netzbetreiber_id`, and `EC_PODLIST` loops over all distinct NBs of the active meter points (`handler/eda.go`). BEG is less field-proven than EEG/GEA operation
- **Individualtarife**: member-specific tariff-plan overrides (migration 076) — a member can get their own time-varying Arbeitspreis (own `tariff_entries`) plus nullable overrides for `free_kwh`/`discount_pct`/`meter_fee_eur`/`participation_fee_eur`/`zaehlpunkts_gebuehr_eur`/`servicegebuehr_bezug_ct_kwh`/`servicegebuehr_einspeisung_ct_kwh` (the last two added in migration 112); nil = fall back to the EEG default, 0 = member exempt from that fee. Independent "one active schedule" scope per member vs. the EEG-wide default (two partial unique indexes). Routes mirror the EEG-wide tariff endpoints under `/eegs/{eegID}/members/{memberID}/tariffs/...`; UI at `/eegs/[eegId]/members/[memberId]/tariff`, with a diff card on the member detail page listing active overrides vs. EEG defaults. **Never call this feature "Sozialtarif"** — that term is legally reserved in Austria for ElWOG basic-supply protections and doesn't apply here.
- **Zählpunktsgebühr**: EEG-wide fee × number of a member's *active* meter points (`abgemeldet_am IS NULL`), added on top of the flat `meter_fee_eur`/`participation_fee_eur`. Always rendered as its own PDF line item ("Zählpunktsgebühr (N × X €)"), unlike the flat fees which only get a separate line on multi-month invoices.
- **Servicegebühr pro kWh** (migration 112): optional `servicegebuehr_bezug_ct_kwh`/`servicegebuehr_einspeisung_ct_kwh` (ct/kWh, default 0) charged on top of the energy price, each its own PDF line ("Servicegebühr Bezug"/"Servicegebühr Einspeisung") with its own kWh × ct/kWh breakdown. Both fold into the consumption (Bezug) net amount — including the Einspeisung variant, which is charged on generation kWh but booked against Bezug so it carries the EEG-level VAT instead of the member-specific generation VAT (net cash effect is the same as deducting it from the feed-in credit). Configurable EEG-wide in EEG-Einstellungen (Rechnungen-Tab) or per member via Individualtarif override.
- **Gebührenzeilen bei 0,00 € anzeigen**: four independently selectable flags (migration 113, one per fee-type PDF line: Fixgebühr/Mitgliedsbeitrag, Zählpunktsgebühr, Servicegebühr Bezug, Servicegebühr Einspeisung) — default off (a 0,00 € line is omitted), each toggle sits directly under its fee's input field in EEG-Einstellungen (Rechnungen-Tab).
- **Email log**: every outbound email (invoices, campaigns, onboarding, portal, EDA notices) is recorded via `invoice.SendLogged()` into `email_log` (status sent/failed, error message); `GET /eegs/{eegID}/email-log`; viewable at `/eegs/[eegId]/settings/email-log`
- **CAMT.053 Zahlungsabgleich**: `POST /eegs/{eegID}/billing/camt053` reads a bank statement, matches `TxDtls.EndToEndId` (invoice UUID, no dashes) against invoices, and marks matches `status=paid` with `paid_at` = booking date; multi-file upload supported from the billing page's global toolbar
- **Bank-Import Duplikatschutz**: two partial unique indexes on `ea_banktransaktionen` prevent double-import on overlapping MT940/CAMT.053 statement periods — by `(eeg_id, buchungsdatum, betrag, referenz)` when a bank reference exists, else by `(eeg_id, buchungsdatum, betrag, verwendungszweck)`
- **EDA response codes & §16e Smartmeter-Benachrichtigung**: worker captures `ResponseCode`/`MeterOwnerName`/`PortalApprovalURL` from `ANTWORT_ECON`; response code `182` (meter not remotely readable) triggers an automatic member email noting the Netzbetreiber's 2-month smart-meter installation obligation under §16e ElWOG (`worker.sendSmartmeterInfoEmail`, idempotent via `customer_notified_at`)
- **Bulk-Mail Performance**: `invoice.BulkSender` opens one SMTP connection per send run and reuses it for all invoices, instead of a fresh TCP+TLS+login per mail (fixed a Cloudflare-tunnel timeout at ~50+ invoices)
- **Energierichtungs-Mismatch-Erkennung**: worker's `detectSchemaDirection` compares the OBIS schema of incoming CR_MSG data against the meter point's stored Energierichtung; on mismatch (e.g. a prosumer's two Zählpunkte swapped at onboarding) it sends a one-time operator email (email_type `eda_direction_mismatch`), idempotent via `direction_mismatch_notified_at` (migration 079)
- **Bezug/Einspeisung-Ungleichgewichts-Warnung**: full unfiltered billing runs compare EEG-wide consumption (wh_self) vs generation (wh_community) totals — the same physically shared energy pool reported twice by the NB; if the relative diff exceeds `energy_imbalance_threshold_promille` (‰, default 1, configurable in EEG settings, migration 086), the run response carries a non-blocking `imbalance_warning` shown in the billing form. Member-filtered and consumption/production-only runs skip the check (false positives)
- **G.01T vor G.01 (Mehrfachteilnahme)**: on Mehrfachteilnahme meter points the NB sends BOTH G.01 (100% of the plant) and G.01T (× Teilnahmefaktor) in arbitrary document order — `buildReadingsFromCRMsg` lets G.01T set `wh_total` bindingly; plain G.01 is only a fallback when no G.01T exists (previously the last code in the document won, corrupting scaled values)
- **Portal-Selbstbedienung E-Mail-Änderung**: member requests a new email in the portal Profil tab (`POST /api/v1/public/portal/email-change`); a confirmation link goes to the NEW address and only `POST .../email-change/confirm/{token}` (page `/portal/email-change/confirm`) applies the change; max one pending request per member (migration 085)
- **Portal-Hinweis in E-Mails**: invoice emails and the onboarding welcome email include a pointer to the member portal (link built from `WEB_BASE_URL`)

## Additional API Endpoints

```
# Member lifecycle
POST /api/v1/eegs/{eegID}/members/{memberID}/austritt  — deregister member (sets INACTIVE, triggers CM_REV_SP for all active meter points with consent_id)
                                                         body: { "austritt_datum": "YYYY-MM-DD" }; idempotent (skips ZPs with pending CM_REV_SP)
GET  /api/v1/eegs/{eegID}/members/{memberID}/sepa-mandat — download SEPA direct debit mandate PDF
GET  /api/v1/eegs/{eegID}/members/{memberID}/sepa-mandat/history — list archived SEPA mandate snapshots (past IBANs), newest first

# Bulk email campaigns
GET  /api/v1/eegs/{eegID}/communications               — list sent campaigns
GET  /api/v1/eegs/{eegID}/communications/{id}          — get campaign detail
POST /api/v1/eegs/{eegID}/communications               — send campaign; multipart: subject, html_body, member_ids (JSON array string), attachments

# Onboarding
POST /api/v1/public/eegs/{eegID}/onboarding           — submit membership application (public, no auth)
GET  /api/v1/eegs/{eegID}/onboarding                  — list onboarding requests (admin)
POST /api/v1/eegs/{eegID}/onboarding/manual            — admin creates a manual onboarding link (no self-service submission needed)
POST /api/v1/eegs/{eegID}/onboarding/{id}/convert      — approve & convert to member
DELETE /api/v1/eegs/{eegID}/onboarding/{id}            — reject/delete request

# Individualtarife (member-specific tariff overrides — same handlers as /eegs/{eegID}/tariffs, memberID scopes them)
GET    /api/v1/eegs/{eegID}/members/{memberID}/tariffs
POST   /api/v1/eegs/{eegID}/members/{memberID}/tariffs
GET    /api/v1/eegs/{eegID}/members/{memberID}/tariffs/{scheduleID}
PUT    /api/v1/eegs/{eegID}/members/{memberID}/tariffs/{scheduleID}
DELETE /api/v1/eegs/{eegID}/members/{memberID}/tariffs/{scheduleID}
PUT    /api/v1/eegs/{eegID}/members/{memberID}/tariffs/{scheduleID}/entries
POST   /api/v1/eegs/{eegID}/members/{memberID}/tariffs/{scheduleID}/activate
DELETE /api/v1/eegs/{eegID}/members/{memberID}/tariffs/{scheduleID}/activate

# Email log
GET  /api/v1/eegs/{eegID}/email-log                    — list every outbound email attempt (status sent|failed) for the EEG

# Meter point registration history
GET  /api/v1/eegs/{eegID}/meter-points/{meterPointID}/history — full Anmeldung/Abmeldung period history for the meter point's Zählpunkt, oldest first

# Payment matching
POST /api/v1/eegs/{eegID}/billing/camt053              — import CAMT.053 bank statement; matches invoice UUID via EndToEndId, marks status=paid

# Member portal (magic-link, no Bearer auth — session token from /exchange)
POST /api/v1/public/portal/request-link               — send magic link to member email (rate-limited)
POST /api/v1/public/portal/exchange                    — exchange magic-link token for a session token
GET  /api/v1/public/portal/me                          — member dashboard data
GET  /api/v1/public/portal/energy                      — monthly energy breakdown
GET  /api/v1/public/portal/invoices                    — invoice list
GET  /api/v1/public/portal/invoices/{invoiceID}/pdf    — invoice PDF download
GET  /api/v1/public/portal/documents                   — EEG documents list (+ /{docID} download)
GET  /api/v1/public/portal/meter-points                — member's meter points
POST /api/v1/public/portal/change-factor               — request Teilnahmefaktor change
POST /api/v1/public/portal/sepa-mandate                — member self-service IBAN change; body: {iban, confirm:true}; archives old mandate, stores freshly signed one
POST /api/v1/public/portal/email-change                — request email change; sends confirmation link to the NEW address (rate-limited)
POST /api/v1/public/portal/email-change/confirm/{token} — confirm & apply the email change

# Mehrfachteilnahme
GET  /api/v1/eegs/{eegID}/participations               — list participations
POST /api/v1/eegs/{eegID}/participations               — create participation
PUT  /api/v1/eegs/{eegID}/participations/{id}          — update participation
DELETE /api/v1/eegs/{eegID}/participations/{id}        — delete participation

# Accounting
GET  /api/v1/eegs/{eegID}/accounting/export?from=&to=&format=datev|xlsx

# Backup / Restore
GET  /api/v1/eegs/{eegID}/backup?sections=...           — download JSON snapshot; sections= comma-list restricts to a subset (omitted = all)
POST /api/v1/eegs/{eegID}/restore                      — restore from JSON file; only touches sections present in the file's "included" list

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
| `/eegs/new` | Create a new EEG; Gemeinschaftstyp selector (EEG/GEA/BEG) as a 3-column grid, Netzbetreiber hidden/optional when BEG is selected |
| `/eegs/[eegId]/reports` | Energy analytics (year/month/day/15-min, per-member breakdown, CSV/XLSX export); shows an L3 quality warning banner when the Netzbetreiber sent unbilled faulty readings |
| `/eegs/[eegId]/accounting` | Accounting export (DATEV CSV + XLSX) |
| `/eegs/[eegId]/onboarding` | Admin approval queue for new member applications |
| `/eegs/[eegId]/participations` | Mehrfachteilnahme CRUD |
| `/eegs/[eegId]/billing` | Billing runs management |
| `/eegs/[eegId]/eda` | EDA process list + Anmeldung/Abmeldung/Teilnahmefaktor actions (also reachable per-ZP from meter point edit page); "Zählpunktdaten anfordern" (CR_REQ_PT) supports Mehrfachabfrage with member/ZP checkbox selection; message log searchable by Zählpunkt (migration 078) |
| `/eegs/[eegId]/import` | Energy data import (XLSX) with coverage chart |
| `/eegs/[eegId]/tariffs` | Tariff schedule management |
| `/eegs/[eegId]/communications` | Bulk email campaigns (compose, member selection, history) |
| `/eegs/[eegId]/settings` | EEG configuration (address, logo, SEPA, billing, DATEV, EDA, auto-billing); Rechnungen tab has "Fixgebühr" (flat, was mislabeled "Zählpunktgebühr") and "Zählpunktsgebühr" (true per-active-ZP fee) as separate fields; Rechnungen tab also has a "Zahlungshinweis" radio group (SEPA-Lastschrift / Überweisungshinweis / kein Hinweis, migration 080) and a "Fixgebühren-Abrechnung" radio group (pro Monat / pro Abrechnungslauf, migration 083); EDA tab has a "Verteilungsmodell" select (statisch/dynamisch, migration 081) that fixes the ECDisModel for the whole community; Abrechnung section has the "Ungleichgewichts-Toleranz" (‰) field for the imbalance warning (migration 086) |
| `/eegs/[eegId]/settings/email-log` | List of every outbound email attempt for the EEG (status, recipient, subject, error) |
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
| `/portal/dashboard` | Member self-service dashboard (energy + invoices); "Profil" tab shows masked current IBAN + self-service "IBAN ändern" form (re-signs SEPA mandate, archives the old one) and an "E-Mail-Adresse ändern" form (confirmation link to the new address, migration 085) |
| `/portal/email-change/confirm` | Confirmation landing page for the member email change (reads token from query, calls the confirm endpoint) |
| `/admin/users` | User administration (admin only) |
| `/eegs/[eegId]/members/[memberId]/meter-points/[id]/edit` | Meter point edit: attributes, abgemeldet_am (manual), Teilnahmefaktor, "Zählpunkt abmelden" EDA section (shown only when EDA configured + consent_id set + not yet deregistered), Registrierungshistorie timeline card |
| `/eegs/[eegId]/members/[memberId]/tariff` | Individualtarif management for one member — own tariff-entry editor + overrides form (free_kwh/discount_pct/meter_fee_eur/participation_fee_eur/zaehlpunkts_gebuehr_eur, each nullable back to EEG default); member detail page shows a diff card of active overrides vs. EEG defaults |

## EDA Worker
Separate binary (`api/cmd/worker/`) activated via `docker compose --profile eda up`.

### Transport modes
| Mode | How to activate | Use case |
|------|----------------|---------|
| `MAIL` | `EDA_TRANSPORT=MAIL` (default) | Production — IMAP polling + SMTP send (per-EEG credentials from DB) |
| `FILE` | `EDA_TRANSPORT=FILE` | Local testing — reads/writes XML files |
| `PONTON` | `EDA_TRANSPORT=PONTON` | **NOT production-ready** — skeleton exists in `transport/ponton.go` but the Send/Poll logic does not match the real Ponton XP API and has never been tested against a live Ponton instance |

### Ponton X/P Migration Plan

**Trigger:** E-Mail-Gateway erlaubt max. 2.500 Nachrichten/Monat. Ab ~40 Mitgliedern mit je 2 Zählpunkten → Ponton X/P Pflicht (EDA GmbH Regelwerk).

**Architektur:** Ponton Messenger (Java-App) läuft als Docker-Service; kommuniziert über AS4/HTTPS mit edanet. Unser Worker sendet via BWA HTTP POST, empfängt über Callback-Endpunkt (kein Polling mehr nötig):
```
Worker → POST http://ponton-messenger:8080/xp/bwa/eda → edanet
edanet → Ponton Callback → POST /api/internal/eda/inbound → Worker
```

**Implementierung (wenn nötig):**
- Phase 1: EDA-Plattformvertrag + Ponton Service-Desk + Partner-Zertifikat + öffentliche HTTPS-URL + Firewall (SIA `217.x.x.x`)
- Phase 2: `ponton-messenger` Docker-Service; GUI-Konfiguration (Partner-Profile, EDA-Schema, BWA-Backend-URL)
- Phase 3: `transport/ponton.go` neu schreiben (BWA 3.0; Skeleton ist falsch); neuer Inbound-Endpunkt `POST /api/internal/eda/inbound` in `server.go`
- Phase 4: Env-Var `EDA_PONTON_BWA_URL`; `pollLoop()` bei PONTON deaktivieren
- Phase 5: Multi-Tenant via `X-ebms-from` AT-Code; Inbound-Routing über `eda_marktpartner_id` (bereits vorhanden)
- Phase 6: Umschalten per `EDA_TRANSPORT`; unter Limit → MAIL, über Limit → PONTON


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
POST /api/v1/eegs/{eegID}/eda/zaehlerstandsgang          — request historical meter data (CR_REQ_PT)
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
- `api/internal/eda/xml/cprequest_builder.go` — builds outbound CPRequest XML (CR_REQ_PT, EC_PODLIST)
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
CR_REQ_PT      → CR_REQ_PT_04.10
EC_PODLIST     → EC_PODLIST_01.00
CM_REV_SP      → CM_REV_SP_01.00
```

**Internal vs. official process names**: the left-hand values are internal DB/code identifiers (`eda_processes.process_type`, handlers, jobs). Only `EC_PRTFACT_CHG` still differs from the official ebutilities.at name (**EC_PRTFACT_CHANGE**) — kept deliberately as internal short form. `EC_REQ_PT` was renamed to the canonical **CR_REQ_PT** in code + data (migration 087, analogous to 054 EC_EINZEL_ANM → EC_REQ_ONL); the wire format (Subject Prozess-Id, XML MessageCodes) was always correct.

### IMAP / Worker Operational Notes

**BODY.PEEK[] is mandatory**: The worker fetches with `{Peek: true}` in `FetchItemBodySection`. Without it, IMAP auto-marks messages as `\Seen` on fetch. If the worker then crashes or times out before explicitly marking `\Seen`, those messages are silently lost — next poll finds 0 unseen messages and processes stay stuck at "sent" indefinitely.

**Test worker job stealing**: `eegabrechnung-eda-worker-test` (FILE transport) shares the same PostgreSQL database as the production MAIL worker. `FOR UPDATE SKIP LOCKED` means whichever worker runs first claims the job. Symptom: processes marked "sent" with empty Subject, no SMTP delivery, no edanet response. **Always stop the test worker before using the MAIL worker:**
```bash
docker stop eegabrechnung-eegabrechnung-eda-worker-test-1
```

**Worker health monitoring**: The `eda_worker_status` table (single row) is the ground truth for worker health:
- `last_poll_at` — when the last poll cycle completed
- `last_error` — last error string (or null if healthy)

**Widerruf (CM_REV_SP) request JSON field**: The direct Widerruf HTTP handler uses `consent_end` (not `valid_from`) in the request body:
```json
{ "zaehlpunkt": "AT...", "consent_end": "2026-05-01" }
```

**Anmeldung valid_from rules**: EC_REQ_ONL `valid_from` must be **at least tomorrow and at most 30 days in the future** (validated in `handler/eda.go`). Onboarding convert auto-clamps stored `beitritts_datum` to this range. Same rule shown in UI with `min`/`max` on date pickers.

**Austritt/Widerruf consent_end rules**: CM_REV_SP `consent_end` must be **at least today and at most 30 Austrian working days in the future** (validated in `handler/eda.go` WiderrufEEG and `handler/member.go` austritt handler).

**Post-confirmation actions (ABSCHLUSS_ECON)**:
- EC_REQ_ONL confirmed → sets `meter_point.registriert_seit` from ECMPList DateFrom; stores `consent_id` from ECMPList; sends confirmation email to member (via EEG SMTP); activates onboarding + member
- CM_REV_SP confirmed (CM_REV_CUS / CM_REV_IMP) → sets `meter_point.abgemeldet_am` from ECMPList ConsentEnd date

**EDA status on meter points** (`meterPointShort` in API response): `registriert_seit`, `abgemeldet_am`, `anmeldung_status`, `abmeldung_status` — derived from `eda_processes`; shown as status badge on member detail page.

## Pricing Logic
Prices are set **only** in Tarifpläne (tariff schedules). 
 During billing (true time-of-use since 2026-07-07):
1. Load active tariff schedule entries overlapping the billing period
2. **Time-of-use**: every 15-min reading is priced with the tariff entry covering its own timestamp (`ReadingRepository.SumTOUForMember` — one query per member joining readings × tariff_entries)
3. kWh outside every entry fall back to `eeg.energy_price` / `eeg.producer_price` (DB fields, not exposed in UI)
4. Free kWh / discount scale the consumption amount proportionally across all entries
5. The invoice shows the derived consumption-weighted average ct/kWh (per month on multi-month invoices)
6. `tariffWeightedPrice` (time-weighted blend, ignores load profile) remains only as an emergency fallback if the TOU query fails

**Fixed fees** (`meter_fee_eur`, `participation_fee_eur`, `zaehlpunkts_gebuehr_eur`) are multiplied by the number of started Vienna calendar months of the (member-effective) billing period when `fee_billing_mode = 'per_month'` (default), or charged once per run with `'per_invoice'`.

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

# Web only — pass current git commit so version is visible in the nav
GIT_COMMIT=$(git rev-parse --short HEAD) docker compose build eegabrechnung-web && \
GIT_COMMIT=$(git rev-parse --short HEAD) docker compose up -d eegabrechnung-web

# EDA worker only (SEPARATE Dockerfile — rebuilding api does NOT rebuild worker!)
docker compose build eegabrechnung-eda-worker && docker compose up -d --force-recreate eegabrechnung-eda-worker

# Both
GIT_COMMIT=$(git rev-parse --short HEAD) docker compose build && \
GIT_COMMIT=$(git rev-parse --short HEAD) docker compose up -d
```

The `GIT_COMMIT` env var is injected as `NEXT_PUBLIC_GIT_COMMIT` build arg into the web image and shown as a short hash in the nav sidebar below the sign-out button. This lets users verify which version is running after an update.

## Playwright Testing (on server)
```bash
cd /tmp/pw_test   # playwright is installed here
node my_test.mjs
```

## Test Data
- EEG: "Sonnenstrom Mustertal" (ID: 5d0151e8-8714-4605-9f20-70ec5d5d5b46)
- 3 members: Hans Mustermann (CONSUMER), Maria Sonnleitner (PROSUMER), Biobauernhof Grünwald GmbH (PRODUCER)
- Each member has meter points assigned

**Never run manual/API testing against "Energiegemeinschaft Wiener Neustadt West"** (ID: `35ac1dc0-e252-43a2-8f63-ec8b7663c1d2`, aka "eegwn") — this is the real production EEG with real member/billing data. Use one of the dedicated test EEGs instead: "BEG Test" (`aa9d20c1-6520-4ad8-8b2e-58bb1e12caab`, `gemeinschaft_typ: BEG`), "GEA Test" (`143f6f15-2c05-415c-a128-fada1768fc98`), "Test EEG" (`5a71eacc-635b-4cac-a65a-e3e70084ded2`), "VAT Matrix Test — Kleinunternehmer"/"— Regelbesteuerung 20%", or "Demo Energiegemeinschaft" (`00000000-0000-0000-0000-000000000010`, `is_demo: true`, blocks email sending). Create additional members/meter points/simulated energy readings there as needed.

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
