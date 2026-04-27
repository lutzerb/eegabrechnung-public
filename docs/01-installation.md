# Kapitel 1: Installation & Betrieb

## 1.1 Systemvoraussetzungen

| Komponente | Mindestanforderung |
|------------|-------------------|
| Docker Engine | 24.x oder neuer |
| Docker Compose | Plugin v2.x (Compose-Datei-Format 3) |
| RAM | 2 GB (empfohlen: 4 GB) |
| Festplatte | 10 GB freier Speicher |

Die folgenden Ports müssen auf dem Host verfügbar sein:

| Port | Dienst | Zweck |
|------|--------|-------|
| 3001 | Web (Next.js) | Browser-Zugriff, einziger nach außen erforderlicher Port |
| 8101 | API (Go) | Interne Nutzung; nur lokal oder per SSH-Tunnel erreichbar |
| 26433 | PostgreSQL | Direktzugriff für Datenbankadministration |

<div class="tip">

Für Produktivbetrieb hinter einem Reverse-Proxy (z. B. Caddy, nginx) muss lediglich Port 3001 exponiert werden. Browser kommunizieren ausschließlich über Next.js-Proxy-Routen mit der API — kein direkter Zugriff auf Port 8101 erforderlich.

</div>

---

## 1.2 Stack starten

```bash
cd /mnt/HC_Volume_103451728/eegabrechnung
docker compose up -d
```

Der Befehl startet drei Container:

| Container | Image | Aufgabe |
|-----------|-------|---------|
| `eegabrechnung-postgres` | postgres:16 | Primäre Datenbank; persistenter Volume-Mount |
| `eegabrechnung-api` | (lokal gebaut) | Go-REST-API; führt Datenbankmigrationen beim Start aus |
| `eegabrechnung-web` | (lokal gebaut) | Next.js-Frontend; leitet API-Aufrufe intern weiter |

Nach dem Start ist die Anwendung unter `http://localhost:3001` erreichbar.

---

## 1.3 Wichtige Umgebungsvariablen

Die Variablen werden in `docker-compose.yaml` gesetzt. Für Produktivumgebungen empfiehlt sich eine `.env`-Datei im Projektverzeichnis.

| Variable | Dienst | Zweck |
|----------|--------|-------|
| `JWT_SECRET` | api, web | Gemeinsames Geheimnis für HS256-JWT-Signierung und -Verifikation |
| `API_INTERNAL_URL` | web | Interne Docker-URL für serverseitige API-Aufrufe (`http://eegabrechnung-api:8080`) |
| `NEXTAUTH_SECRET` / `AUTH_SECRET` | web | Verschlüsselungsschlüssel für next-auth-Sessions |
| `EDA_TRANSPORT` | eda-worker | Transportmodus: `MAIL` (Standard), `PONTON` oder `FILE` |
| `EDA_INBOX_DIR` | eda-worker | Verzeichnis für eingehende XML-Dateien (FILE-Modus; Standard: `./test/eda-inbox`) |
| `EDA_OUTBOX_DIR` | eda-worker | Verzeichnis für ausgehende XML-Dateien (FILE-Modus; Standard: `./test/eda-outbox`) |

<div class="warning">

`JWT_SECRET` muss in API und Web identisch sein. Ein Mismatch führt dazu, dass alle API-Anfragen mit 401 Unauthorized abgewiesen werden.

</div>

---

## 1.4 Datenbankmigrationen

Die Migrationen sind im API-Binary eingebettet (`api/internal/db/migrations/`) und werden beim Start automatisch via **golang-migrate** ausgeführt. Es ist kein manueller Eingriff erforderlich.

| Migration | Inhalt |
|-----------|--------|
| 001\_init | `eegs`, `members`, `meter_points`, `energy_readings`, `invoices` |
| 002\_eda | `eda_messages` |
| 003\_pricing | `producer_price`, `use_vat`, `vat_pct`, `meter_fee_eur`, `free_kwh`, `discount_pct`, `participation_fee_eur`, `billing_period` |
| 004\_features | `invoice_number_prefix`/`digits`, `invoice_pre`/`post`/`footer_text`, Invoice-Status |
| 005\_auth | `organizations`, `users`, `organization_id` auf `eegs`; Standard-Organisation + Admin-Benutzer |
| 006\_member\_vat | Per-Mitglied `use_vat` / `vat_pct`-Overrides |
| 007\_member\_uid | `uid_nummer` (UID/Steuernummer) auf `members` |
| 008\_member\_address | `strasse`, `plz`, `ort` auf `members` |
| 009\_invoice\_breakdown | `consumption_kwh`, `generation_kwh` auf `invoices` |
| 010\_sepa\_fields | `iban`, `bic`, `sepa_creditor_id` auf `eegs` |
| 011\_billing\_runs | Tabelle `billing_runs`; Fremdschlüssel `invoice → billing_run_id` |
| 012\_user\_assignments | `user_eeg_assignments` (Zugriffskontrolle pro Benutzer/EEG) |
| 013\_tariff\_schedules | `tariff_schedules` + `tariff_entries`; partiell-eindeutiger Index (ein aktiver Plan pro EEG) |
| 014\_eda\_schema | `message_type`/`subject`/`body`/`processed_at` auf `eda_messages`; `eda_marktpartner_id`/`eda_netzbetreiber_id`/`eda_transition_date` auf `eegs` |
| 015\_eda\_processes | Tabelle `eda_processes` (Prozesstyp, Status-Lifecycle, `conversation_id`, `participation_factor`, `deadline_at`) |
| 016\_eda\_gaps | `source`-Spalte auf `energy_readings` (`xlsx`\|`eda`); `message_id`-Deduplizierung; `eda_errors`-Dead-Letter-Tabelle; `eda_worker_status`-Singleton |
| 017\_member\_status\_invstart | `status`-Spalte auf `members` (`ACTIVE`/`INACTIVE`); `invoice_number_start` auf `eegs` |
| 018\_energy\_quality | `quality`-Spalte auf `energy_readings` (`L0`/`L1`/`L2`/`L3`); L3 wird von der Abrechnung ausgeschlossen |
| 019\_credit\_notes | `generate_credit_notes`, `credit_note_number_prefix`/`digits` auf `eegs`; `document_type` auf `invoices` (`invoice`\|`credit_note`) |
| 020\_logo | `logo_path` auf `eegs` |
| 021\_mehrfachteilnahme | Tabelle `eeg_meter_participations` (`factor`, `share_type`, `valid_from`/`until`) für Mehrfach-EEG-Mitgliedschaft |
| 022\_onboarding | Tabelle `onboarding_requests` (Magic-Token-Flow, Status: `pending`→`approved`→`converted`/`rejected`) |
| 023\_member\_dates | `beitritt_datum`, `austritt_datum` auf `members` |
| 024\_onboarding\_beitritt | `beitritts_datum` auf `onboarding_requests` |
| 025\_accounting | `net_amount`/`vat_amount`/`vat_pct_applied` auf `invoices`; DATEV-Felder auf `eegs` (Erlös-/Aufwandskonto, Debitorenkonto, Berater-/Mandantennummer) |
| 026\_eeg\_address\_billing\_workflow | `strasse`/`plz`/`ort`/`uid_nummer` auf `eegs`; `billing_run`-Status `completed`→`finalized`; `storno_pdf_path` auf `invoices` |
| 027\_job\_retry | `retry_count` auf der `jobs`-Tabelle |
| 028\_eeg\_gruendungsdatum | `gruendungsdatum` (Gründungsdatum) auf `eegs` |
| 029\_meter\_point\_generation\_type | `generation_type` auf `meter_points` (PV, Windkraft, Wasserkraft etc.) |
| 030\_member\_portal | Tabelle `member_portal_sessions` (Magic-Link-Auth für Mitglieder-Self-Service) |
| 031\_onboarding\_contract | `onboarding_contract_text` auf `eegs` |
| 032\_member\_email\_campaigns | Tabelle `member_email_campaigns` + `member_email_campaign_attachments` (Massen-E-Mail-Kampagnen) |
| 033\_eeg\_documents | Tabelle `eeg_documents` (uploadbare PDFs für Onboarding-Seite) |
| 034\_onboarding\_email\_verify | Tabelle `onboarding_email_verifications` (Magic-Token E-Mail-Verifizierung beim Onboarding) |
| 035\_invoice\_split\_vat | Split-USt auf Rechnungen (separate Erzeugung-/Verbrauch-Zeilen) |
| 036\_invoice\_number\_uniqueness | Eindeutiger Index auf Rechnungsnummer pro EEG |
| 037\_document\_show\_in\_onboarding | `show_in_onboarding`-Flag auf `eeg_documents` |
| 038\_demo\_mode | `is_demo`-Flag auf `eegs` (blockiert E-Mail-Versand in Demo-EEGs) |
| 039\_eda\_message\_status | `status`-Spalte auf `eda_messages` (`pending`/`processed`/`error`) |
| 040\_eda\_message\_addresses | `from_address`/`to_address` auf `eda_messages` |
| 041\_eda\_process\_ecmplist\_fields | Zusätzliche Felder auf `eda_processes` für ECMPList |
| 042\_eda\_messages\_process\_id | `eda_process_id`-FK auf `eda_messages` |
| 043\_per\_eeg\_credentials | Per-EEG-Credentials auf `eegs`: `eda_imap_*`, `eda_smtp_*`, `smtp_*` (AES-256-GCM verschlüsselt) |
| 044\_eda\_error\_subject | `subject`-Spalte auf `eda_errors` |
| 045\_onboarding\_business\_fields | `business_role`, `uid_nummer`, `use_vat` auf `onboarding_requests` |
| 046\_onboarding\_reminder | `reminder_sent_at` auf `onboarding_requests` (72h-Follow-up) |
| 047\_meter\_point\_abgemeldet\_am | `abgemeldet_am` (DATE) auf `meter_points` |
| 048\_eda\_process\_error\_notification | `error_notification_sent_at` auf `eda_processes` |
| 049\_auto\_billing | `auto_billing_enabled`/`day_of_month`/`period`/`last_run_at` auf `eegs` |
| 050\_sepa\_return | `sepa_return_at`/`reason`/`note` auf `invoices` |
| 051\_gap\_alert | `gap_alert_enabled`/`threshold_days` auf `eegs`; `gap_alert_sent_at` auf `meter_points` |
| 052/053\_meter\_point\_notes | `notes` (TEXT, DEFAULT '') auf `meter_points` |
| 054\_rename\_ec\_einzel\_anm | Umbenennung `EC_EINZEL_ANM` → `EC_REQ_ONL` |
| 055\_meter\_point\_consent\_id | `consent_id` (TEXT, DEFAULT '') auf `meter_points` |
| 056\_ea\_buchhaltung | Tabellen `ea_konten`, `ea_buchungen`, `ea_belege`, `ea_uva_perioden`; EA-Felder auf `eegs` |
| 057\_ea\_banktransaktionen | Tabelle `ea_banktransaktionen` (Kontoauszug-Import MT940/CAMT.053) |
| 058\_uva\_kennzahlen | `kz_044` (10 % USt) + `kz_057` (RC §19) auf `ea_uva_perioden` |
| 059/060\_konto\_k1kz | `k1_kz` auf `ea_konten` (FinanzOnline K1-Kennzahl) |
| 061\_sepa\_mandate\_prenotification | `sepa_mandate_signed_at`/`ip`/`text` auf `members`; `sepa_pre_notification_days` auf `eegs` (Standard 14) |
| 062\_portal\_show\_full\_energy | `portal_show_full_energy` (BOOL, DEFAULT TRUE) auf `eegs` |
| 063\_invoice\_split\_amounts | `consumption_net_amount` + `generation_net_amount` auf `invoices` |
| 064\_invoice\_split\_amounts\_tariff\_backfill | Re-Backfill Prosumer-Split-Beträge für Tarifplan-EEGs |
| 065\_invoice\_split\_amounts\_vat\_recovery | Exakte Rückgewinnung Prosumer-Split-Beträge für Nicht-KU-EEGs |
| 066\_ea\_buchungen\_audit | Soft-Delete auf `ea_buchungen`; Tabelle `ea_buchungen_changelog` (BAO §131 Audit-Trail) |

---

## 1.5 Stack neu bauen

Nach Codeänderungen müssen die betroffenen Container neu gebaut werden:

```bash
# Nur API neu bauen und starten
docker compose build eegabrechnung-api && docker compose up -d eegabrechnung-api

# Nur Web neu bauen und starten
docker compose build eegabrechnung-web && docker compose up -d eegabrechnung-web

# Beide neu bauen und starten
docker compose build && docker compose up -d
```

<div class="tip">

`docker compose up -d` startet nur Container, die noch nicht laufen oder deren Image sich geändert hat. Laufende, unveränderte Container werden nicht neu gestartet.

</div>

---

## 1.6 Logs & Debugging

```bash
# API-Logs live verfolgen
docker compose logs -f eegabrechnung-api

# Web-Logs live verfolgen
docker compose logs -f eegabrechnung-web

# Alle Logs (letzte 100 Zeilen)
docker compose logs --tail=100

# Datenbankzugriff (psql)
docker compose exec eegabrechnung-postgres psql -U postgres -d eegabrechnung
```

<div class="tip">

Die API gibt strukturierte JSON-Logs aus. Für lesbare Ausgabe empfiehlt sich `docker compose logs -f eegabrechnung-api | jq .`.

</div>

---

## 1.7 EDA Worker

Der EDA Worker ist ein optionaler zweiter Dienst (`api/cmd/worker/`), der über ein separates Docker-Compose-Profil gestartet wird. Er ist nur erforderlich, wenn automatisierte EDA-Prozesskommunikation (MaKo-XML) genutzt wird.

```bash
# Worker zusammen mit dem Hauptstack starten
docker compose --profile eda up -d

# Worker einmalig ausführen (z. B. für Tests mit FILE-Transport)
docker compose --profile eda run --rm -e EDA_TRANSPORT=FILE eda-worker
```

### Transportmodi

| Modus | Aktivierung | Einsatz |
|-------|-------------|---------|
| `MAIL` | `EDA_TRANSPORT=MAIL` (Standard) | Produktion — IMAP-Polling für Eingang, SMTP-Versand für Ausgang |
| `PONTON` | `EDA_TRANSPORT=PONTON` | Produktion — HTTP-Endpunkt via Ponton XP |
| `FILE` | `EDA_TRANSPORT=FILE` | Lokales Testen — liest/schreibt XML-Dateien aus konfigurierten Verzeichnissen |

### FILE-Transport (Lokaltests)

```bash
# Eingehende XML-Dateien (z. B. CPDocument-Bestätigungen) ablegen unter:
#   test/eda-inbox/<dateiname>.xml

# Verarbeitete Dateien werden verschoben nach:
#   test/eda-inbox/processed/

# Ausgehende XML-Dateien werden geschrieben nach:
#   test/eda-outbox/<timestamp>_<prozess>.xml
```

<div class="warning">

Im MAIL-Modus müssen IMAP- und SMTP-Zugangsdaten vollständig konfiguriert sein, bevor der Worker gestartet wird. Fehlende Konfiguration führt zu einem sofortigen Absturz des Workers.

</div>

---

## 1.8 Upgrade auf eine neue Version

```bash
# 1. Aktuellen Code holen
git pull

# 2. Datenbank vorher sichern (empfohlen)
docker compose exec eegabrechnung-postgres pg_dump -U postgres eegabrechnung > backup_$(date +%Y%m%d).sql

# 3. Neue Images bauen und Stack neu starten
docker compose build && docker compose up -d
```

**Datenbankmigrationen laufen automatisch** beim Start der API — kein manueller Eingriff notwendig. Neue Migrationen werden beim Containerstart erkannt und eingespielt.

Falls der EDA-Worker ebenfalls aktualisiert werden soll:

```bash
docker compose build eegabrechnung-eda-worker && docker compose up -d --force-recreate eegabrechnung-eda-worker
```

<div class="warning">

Der EDA-Worker hat ein **separates Dockerfile** — `docker compose build` baut ihn nur, wenn er im aktiven Profil enthalten ist. Explizit bauen mit `docker compose build eegabrechnung-eda-worker`.

</div>
