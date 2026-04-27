# Anhang: Datenbankschema

---

## Migrations-Übersicht

| Nr. | Was wurde hinzugefügt |
|-----|-----------------------|
| 001 | `eegs`, `members`, `meter_points`, `energy_readings`, `invoices` |
| 002 | `eda_messages` |
| 003 | Preisfelder auf `eegs`: `producer_price`, `use_vat`, `vat_pct`, `meter_fee_eur`, `free_kwh`, `discount_pct`, `participation_fee_eur`, `billing_period` |
| 004 | Rechnungsformatierung: `invoice_number_prefix/digits`, `invoice_pre/post/footer_text`; Rechnungsstatus |
| 005 | `organizations`, `users`, `organization_id` auf `eegs`; Standard-Organisation + Admin-Benutzer |
| 006 | Mitglieder-VAT-Override: `use_vat` / `vat_pct` auf `members` |
| 007 | `uid_nummer` (UID/Steuer-ID) auf `members` |
| 008 | Adressfelder auf `members`: `strasse`, `plz`, `ort` |
| 009 | `consumption_kwh`, `generation_kwh` auf `invoices` |
| 010 | SEPA-Felder auf `eegs`: `iban`, `bic`, `sepa_creditor_id` |
| 011 | `billing_runs`; FK `billing_run_id` auf `invoices` |
| 012 | `user_eeg_assignments` (Benutzer-EEG-Zugriffskontrolle) |
| 013 | `tariff_schedules` + `tariff_entries`; Partial-Unique-Index (ein aktiver Plan pro EEG) |
| 014 | `message_type`, `subject`, `body`, `processed_at` auf `eda_messages`; EDA-Felder auf `eegs` |
| 015 | `eda_processes` (Prozesstyp, Status-Lifecycle, `conversation_id`, `participation_factor`, `deadline_at`) |
| 016 | `source` auf `energy_readings`; Dedup via `message_id` auf `eda_messages`; `eda_errors`; `eda_worker_status` |
| 017 | `status` (ACTIVE/INACTIVE) auf `members`; `invoice_number_start` auf `eegs` |
| 018 | `quality` (L0/L1/L2/L3) auf `energy_readings`; L3 von Abrechnung ausgeschlossen |
| 019 | `generate_credit_notes`, `credit_note_number_prefix/digits` auf `eegs`; `document_type` auf `invoices` |
| 020 | `logo_path` auf `eegs` |
| 021 | `eeg_meter_participations` (Mehrfachteilnahme: `factor`, `share_type`, `valid_from/until`) |
| 022 | `onboarding_requests` (Magic-Token-Flow, Status: pending→approved→converted/rejected) |
| 023 | `beitritt_datum`, `austritt_datum` auf `members` |
| 024 | `beitritts_datum` auf `onboarding_requests` |
| 025 | `net_amount`, `vat_amount`, `vat_pct_applied` auf `invoices`; DATEV-Felder auf `eegs` |
| 026 | `strasse/plz/ort/uid_nummer` auf `eegs`; Billing-Run-Status `completed`→`finalized`; `storno_pdf_path` auf `invoices` |
| 027 | `retry_count` auf `jobs` |
| 028 | `gruendungsdatum` auf `eegs` |
| 029 | `generation_type` auf `meter_points` (PV/Windkraft/Wasserkraft etc.) |
| 030 | `member_portal_sessions` (Magic-Link-Auth für Mitglieder-Self-Service) |
| 031 | `onboarding_contract_text` auf `eegs` (konfigurierbarer Vertragstext für Onboarding-Portal) |
| 032 | `member_email_campaigns` (Massen-E-Mail-Kampagnen mit Anhängen als JSONB) |
| 033 | `eeg_documents` (uploadbare PDFs für Onboarding-Seite) |
| 034 | `onboarding_email_verifications` (Magic-Token E-Mail-Verifizierung beim Onboarding) |
| 035 | Split-USt auf `invoices`: `consumption_vat_pct/amount` + `generation_vat_pct/amount` |
| 036 | Unique-Index auf Rechnungsnummer pro EEG und Dokumententyp |
| 037 | `show_in_onboarding` Flag auf `eeg_documents` |
| 038 | `is_demo` Flag auf `eegs` (blockiert E-Mail-Versand in Demo-EEGs); Demo-Organisation + Demo-EEG eingefügt |
| 039 | Status-Constraint auf `eda_messages` erweitert um `processed` |
| 040 | `from_address`/`to_address` auf `eda_messages` |
| 041 | Zusätzliche Felder auf `eda_processes` für ECMPList: `ec_dis_model`, `date_to`, `energy_direction`, `ec_share` |
| 042 | `eda_process_id` FK auf `eda_messages` (Korrelation Nachricht ↔ Prozess) |
| 043 | Per-EEG-Credentials auf `eegs`: `eda_imap_*`, `eda_smtp_*`, `smtp_*` (AES-256-GCM verschlüsselt) |
| 044 | `subject` auf `eda_errors` (MailSubject der referenzierten ausgehenden Nachricht) |
| 045 | Business-Felder auf `onboarding_requests`: `business_role`, `uid_nummer`, `use_vat` |
| 046 | `reminder_sent_at` auf `onboarding_requests` (72h-Folgeup-Erinnerung) |
| 047 | `abgemeldet_am` (DATE) auf `meter_points` (gesetzt wenn CM_REV_SP bestätigt) |
| 048 | `error_notification_sent_at` auf `eda_processes` |
| 049 | `auto_billing_enabled`, `auto_billing_day_of_month`, `auto_billing_period`, `auto_billing_last_run_at` auf `eegs` |
| 050 | `sepa_return_at`, `sepa_return_reason`, `sepa_return_note` auf `invoices` |
| 051 | `gap_alert_enabled`, `gap_alert_threshold_days` auf `eegs`; `gap_alert_sent_at` auf `meter_points` |
| 052/053 | `notes` (TEXT, NOT NULL DEFAULT '') auf `meter_points` |
| 054 | Umbenennung EC_EINZEL_ANM → EC_REQ_ONL in `eda_processes` und ausstehenden Jobs |
| 055 | `consent_id` (TEXT, DEFAULT '') auf `meter_points` (ConsentId vom NB aus ZUSTIMMUNG_ECON) |
| 056 | `ea_konten`, `ea_buchungen`, `ea_belege`, `ea_uva_perioden`; `ea_steuernummer`/`ea_finanzamt`/`ea_uva_periodentyp` auf `eegs` |
| 057 | `ea_banktransaktionen` (Kontoauszug-Import MT940/CAMT053; `match_status`: offen/auto/bestaetigt/ignoriert) |
| 058 | `kz_044` (USt 10%) + `kz_057` (RC §19 domestic) auf `ea_uva_perioden` |
| 059/060 | `k1_kz` auf `ea_konten` (K1-Kennzahl für FinanzOnline; 059 initial, 060 Korrekturen nach BMF K1 2025) |
| 061 | `sepa_mandate_signed_at`, `sepa_mandate_signed_ip`, `sepa_mandate_text` auf `members`; `sepa_pre_notification_days` auf `eegs` (Standard 14) |
| 062 | `portal_show_full_energy` (BOOL, DEFAULT TRUE) auf `eegs` |
| 063 | `consumption_net_amount` + `generation_net_amount` (DECIMAL 12,4) auf `invoices`; Backfill für reine Verbraucher/Erzeuger; Prosumer-Näherung über Flat-EEG-Preise |
| 064 | Re-Backfill Prosumer Split-Beträge für EEGs mit Tarifplänen (zeitgewichteter Durchschnittspreis) |
| 065 | Exakte Rückgewinnung Prosumer Split-Beträge für Nicht-KU EEGs (use_vat=true, vat_pct > 0) |
| 066 | Soft-Delete auf `ea_buchungen` (`deleted_at`, `deleted_by`); `ea_buchungen_changelog` (BAO §131 Audit-Trail) |

---

## Tabellen

### organizations

Multi-Tenancy-Root. Jede Instanz kann mehrere Organisationen verwalten; alle Daten sind nach `organization_id` partitioniert.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `name` | TEXT | Bezeichnung der Organisation |

---

### users

Benutzerkonten für den Admin-Bereich. Passwörter werden mit bcrypt gehasht gespeichert.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `organization_id` | UUID FK | Zugehörige Organisation |
| `email` | TEXT UNIQUE | Login-Name |
| `password_hash` | TEXT | bcrypt-Hash |
| `role` | TEXT | `admin` oder `user` |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### eegs

Energiegemeinschaften. Zentrale Konfigurationstabelle mit Preisen, Abrechnungsparametern, SEPA-, DATEV- und EDA-Einstellungen.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `organization_id` | UUID FK | Zugehörige Organisation |
| `name` | TEXT | Name der EEG |
| `netzbetreiber` | TEXT | Netzbetreiber-Bezeichnung |
| `gruendungsdatum` | DATE | Gründungsdatum der EEG |
| `strasse` | TEXT | Adresse (§11 UStG) |
| `plz` | TEXT | Postleitzahl |
| `ort` | TEXT | Ort |
| `uid_nummer` | TEXT | UID-Nummer der EEG |
| `energy_price` | NUMERIC | Fallback-Arbeitspreis (ct/kWh) |
| `producer_price` | NUMERIC | Fallback-Einspeisetarif (ct/kWh) |
| `use_vat` | BOOLEAN | USt-Pflicht der EEG |
| `vat_pct` | NUMERIC | USt-Satz (z. B. 20) |
| `meter_fee_eur` | NUMERIC | Zählpunktgebühr (EUR/Monat) |
| `free_kwh` | NUMERIC | Freimengen-kWh |
| `discount_pct` | NUMERIC | Rabatt in Prozent |
| `participation_fee_eur` | NUMERIC | Mitgliedsbeitrag (EUR) |
| `billing_period` | TEXT | Abrechnungsperiode |
| `invoice_number_prefix` | TEXT | Rechnungsnummernpräfix |
| `invoice_number_digits` | INT | Stellen der laufenden Nummer |
| `invoice_number_start` | INT | Startnummer für Rechnungen |
| `invoice_pre_text` | TEXT | Rechnungstext vor Positionen |
| `invoice_post_text` | TEXT | Rechnungstext nach Positionen |
| `invoice_footer_text` | TEXT | Fußzeile auf Rechnungen |
| `generate_credit_notes` | BOOLEAN | Gutschriften für Producer aktivieren |
| `credit_note_number_prefix` | TEXT | Gutschrift-Nummernpräfix |
| `credit_note_number_digits` | INT | Stellen der Gutschrift-Nummer |
| `iban` | TEXT | IBAN für SEPA |
| `bic` | TEXT | BIC für SEPA |
| `sepa_creditor_id` | TEXT | SEPA-Gläubiger-ID |
| `sepa_pre_notification_days` | INT DEFAULT 14 | Vorlaufzeit für SEPA-Pre-Notification in Tagen |
| `eda_marktpartner_id` | TEXT | EDA Marktpartner-ID |
| `eda_netzbetreiber_id` | TEXT | EDA Netzbetreiber-ID |
| `eda_transition_date` | DATE | EDA Umstellungsdatum |
| `eda_imap_host` | TEXT | EDA IMAP-Host (z. B. `mail.edanet.at:993`) |
| `eda_imap_user` | TEXT | EDA IMAP-Benutzername |
| `eda_imap_password_enc` | TEXT | EDA IMAP-Passwort (AES-256-GCM verschlüsselt) |
| `eda_smtp_host` | TEXT | EDA SMTP-Host (z. B. `mail.edanet.at:465`) |
| `eda_smtp_user` | TEXT | EDA SMTP-Benutzername |
| `eda_smtp_password_enc` | TEXT | EDA SMTP-Passwort (AES-256-GCM verschlüsselt) |
| `eda_smtp_from` | TEXT | EDA SMTP-Absenderadresse |
| `smtp_host` | TEXT | Rechnungs-SMTP-Host |
| `smtp_user` | TEXT | Rechnungs-SMTP-Benutzername |
| `smtp_password_enc` | TEXT | Rechnungs-SMTP-Passwort (AES-256-GCM verschlüsselt) |
| `smtp_from` | TEXT | Rechnungs-SMTP-Absenderadresse |
| `logo_path` | TEXT | Pfad zur Logo-Datei |
| `datev_revenue_account` | TEXT | DATEV Erlöskonto |
| `datev_expense_account` | TEXT | DATEV Aufwandskonto |
| `datev_debitor_account` | TEXT | DATEV Debitorenkonto |
| `datev_consultant_nr` | TEXT | DATEV Beraternummer |
| `datev_client_nr` | TEXT | DATEV Mandantennummer |
| `onboarding_contract_text` | TEXT DEFAULT '' | Konfigurierbarer Vertragstext für Onboarding-Portal; Platzhalter: `{iban}`, `{datum}` |
| `is_demo` | BOOLEAN DEFAULT FALSE | Demo-Modus — blockiert E-Mail-Versand |
| `auto_billing_enabled` | BOOLEAN DEFAULT FALSE | Automatische Abrechnung aktiviert |
| `auto_billing_day_of_month` | SMALLINT | Tag des Monats für automatische Abrechnung (1–28) |
| `auto_billing_period` | TEXT | `monthly` oder `quarterly` |
| `auto_billing_last_run_at` | TIMESTAMPTZ | Zeitpunkt des letzten automatischen Laufs |
| `gap_alert_enabled` | BOOLEAN DEFAULT TRUE | Lücken-Alarm aktiviert |
| `gap_alert_threshold_days` | SMALLINT DEFAULT 5 | Schwellenwert in Tagen ohne Messdaten |
| `portal_show_full_energy` | BOOLEAN DEFAULT TRUE | Mitgliederportal zeigt EEG-Gesamtenergie (nicht nur Mitgliedsanteil) |
| `ea_steuernummer` | VARCHAR(30) | Steuernummer beim Finanzamt |
| `ea_finanzamt` | VARCHAR(100) | Bezeichnung des Finanzamts |
| `ea_uva_periodentyp` | VARCHAR(10) DEFAULT 'QUARTAL' | UVA-Periodentyp: `MONAT` oder `QUARTAL` |

---

### members

Mitglieder einer EEG. Können Privatpersonen oder Unternehmen sein.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `firstname` | TEXT | Vorname |
| `lastname` | TEXT | Nachname |
| `company` | TEXT | Firmenname (bei juristischen Personen) |
| `email` | TEXT | E-Mail-Adresse |
| `phone` | TEXT | Telefonnummer |
| `strasse` | TEXT | Straße |
| `plz` | TEXT | Postleitzahl |
| `ort` | TEXT | Ort |
| `uid_nummer` | TEXT | UID-Nummer des Mitglieds |
| `iban` | TEXT | IBAN für SEPA-Zahlungen |
| `bic` | TEXT | BIC |
| `type` | TEXT | `CONSUMER` · `PROSUMER` · `PRODUCER` |
| `status` | TEXT | `ACTIVE` · `INACTIVE` |
| `use_vat` | BOOLEAN | Individuelle USt-Pflicht (überschreibt EEG-Standard) |
| `vat_pct` | NUMERIC | Individueller USt-Satz |
| `beitritt_datum` | DATE | Eintrittsdatum |
| `austritt_datum` | DATE | Austrittsdatum |
| `sepa_mandate_signed_at` | TIMESTAMPTZ | Zeitpunkt der SEPA-Mandat-Unterzeichnung (erfasst beim Onboarding) |
| `sepa_mandate_signed_ip` | TEXT | IP-Adresse bei Mandat-Unterzeichnung |
| `sepa_mandate_text` | TEXT | Volltext des unterzeichneten Mandats |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### meter_points

Zählpunkte der Mitglieder. Ein Mitglied kann mehrere Zählpunkte haben.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `member_id` | UUID FK | Zugehöriges Mitglied |
| `metering_point_id` | TEXT | Zählpunkt-ID (AT-Netz-Format) |
| `direction` | TEXT | `consumption` (Verbrauch) · `generation` (Einspeisung) |
| `generation_type` | TEXT | `PV` · `Windkraft` · `Wasserkraft` · `Biomasse` · `Sonstiges` |
| `abgemeldet_am` | DATE | Abmeldedatum (gesetzt wenn CM_REV_SP bestätigt) |
| `consent_id` | TEXT DEFAULT '' | ConsentId vom Netzbetreiber (aus ZUSTIMMUNG_ECON); erforderlich für CM_REV_SP |
| `notes` | TEXT DEFAULT '' | Interne Notizen |
| `gap_alert_sent_at` | TIMESTAMPTZ | Zeitpunkt des letzten gesendeten Lücken-Alarms |

---

### energy_readings

Energiemesswerte pro Zählpunkt und Zeitstempel.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `meter_point_id` | UUID FK | Zugehöriger Zählpunkt |
| `measured_at` | TIMESTAMPTZ | Messzeitpunkt |
| `wh_total` | NUMERIC | Gesamtmenge in **kWh** (siehe Hinweis unten) |
| `wh_self` | NUMERIC | Eigenversorgungsanteil in **kWh** |
| `wh_community` | NUMERIC | Gemeinschaftsanteil in **kWh** |
| `source` | TEXT | `xlsx` · `eda` |
| `quality` | TEXT | `L0` · `L1` · `L2` · `L3` |
| `created_at` | TIMESTAMPTZ | Importzeitpunkt |

---

### tariff_schedules

Tarifpläne einer EEG. Pro EEG kann maximal ein Plan gleichzeitig aktiv sein.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `name` | TEXT | Bezeichnung des Tarifplans |
| `active` | BOOLEAN | Ist dieser Plan derzeit aktiv? |
| `valid_from` | DATE | Gültig ab |
| `valid_until` | DATE | Gültig bis (NULL = unbegrenzt) |

---

### tariff_entries

Einzelne Preiseinträge innerhalb eines Tarifplans, mit zeitlicher Auflösung.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `schedule_id` | UUID FK | Zugehöriger Tarifplan |
| `valid_from` | TIMESTAMPTZ | Gültig ab |
| `valid_until` | TIMESTAMPTZ | Gültig bis |
| `energy_price` | NUMERIC | Arbeitspreis (ct/kWh) |
| `producer_price` | NUMERIC | Einspeisetarif (ct/kWh) |
| `granularity` | TEXT | `annual` · `monthly` · `daily` · `15min` |

---

### billing_runs

Abrechnungsläufe — jede Abrechnungsoperation erzeugt genau einen Lauf.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `period_from` | DATE | Abrechnungszeitraum Start |
| `period_to` | DATE | Abrechnungszeitraum Ende |
| `status` | TEXT | `draft` · `finalized` · `cancelled` |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |
| `finalized_at` | TIMESTAMPTZ | Finalisierungszeitpunkt |

---

### invoices

Rechnungen und Gutschriften, die einem Abrechnungslauf zugeordnet sind.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `billing_run_id` | UUID FK | Zugehöriger Abrechnungslauf |
| `member_id` | UUID FK | Zugehöriges Mitglied |
| `period_from` | DATE | Abrechnungszeitraum Start |
| `period_to` | DATE | Abrechnungszeitraum Ende |
| `invoice_number` | TEXT | Rechnungsnummer |
| `document_type` | TEXT | `invoice` · `credit_note` |
| `status` | TEXT | `draft` · `finalized` · `sent` · `paid` · `cancelled` |
| `consumption_kwh` | NUMERIC | Abgerechnete Verbrauchsmenge (kWh) |
| `generation_kwh` | NUMERIC | Abgerechnete Einspeisemenge (kWh) |
| `net_amount` | NUMERIC | Nettobetrag gesamt (EUR) |
| `vat_amount` | NUMERIC | USt-Betrag gesamt (EUR) |
| `vat_pct_applied` | NUMERIC | Angewendeter USt-Satz |
| `consumption_vat_pct` | NUMERIC DEFAULT 0 | USt-Satz auf Verbrauchsanteil |
| `consumption_vat_amount` | NUMERIC DEFAULT 0 | USt-Betrag auf Verbrauchsanteil (EUR) |
| `generation_vat_pct` | NUMERIC DEFAULT 0 | USt-Satz auf Erzeugungsanteil |
| `generation_vat_amount` | NUMERIC DEFAULT 0 | USt-Betrag auf Erzeugungsanteil (EUR) |
| `consumption_net_amount` | DECIMAL(12,4) DEFAULT 0 | Netto-Verbrauchsanteil (EUR) |
| `generation_net_amount` | DECIMAL(12,4) DEFAULT 0 | Netto-Erzeugungsanteil (EUR) |
| `sepa_return_at` | TIMESTAMPTZ | Zeitpunkt der Rücklastschrift |
| `sepa_return_reason` | TEXT | Rückgabegrund (SEPA-Code, z. B. AC01, AM04) |
| `sepa_return_note` | TEXT | Interne Notiz zur Rücklastschrift |
| `pdf_path` | TEXT | Pfad zur PDF-Datei |
| `storno_pdf_path` | TEXT | Pfad zur Storno-PDF (bei Stornierung) |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### eda_processes

EDA-Prozesse für die Kommunikation mit dem Netzbetreiber (MaKo XML).

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `process_type` | TEXT | `EC_REQ_ONL` · `EC_PRTFACT_CHG` · `CM_REV_SP` · `EC_REQ_PT` · `EC_PODLIST` |
| `status` | TEXT | `pending` · `sent` · `first_confirmed` · `confirmed` · `completed` · `rejected` · `error` |
| `conversation_id` | TEXT | MaKo Konversations-ID |
| `meter_point_id` | UUID FK | Betroffener Zählpunkt |
| `participation_factor` | NUMERIC | Teilnahmefaktor (bei EC_PRTFACT_CHG) |
| `ec_dis_model` | TEXT DEFAULT '' | Verteilungsmodell (ECMPList-Feld) |
| `date_to` | DATE | Enddatum (ECMPList-Feld) |
| `energy_direction` | TEXT DEFAULT '' | Energierichtung (ECMPList-Feld) |
| `ec_share` | NUMERIC | EC-Anteil (ECMPList-Feld) |
| `deadline_at` | TIMESTAMPTZ | Frist für Netzbetreiber-Antwort |
| `error_notification_sent_at` | TIMESTAMPTZ | Zeitpunkt der Fehlerbenachrichtigung |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### eda_messages

Protokoll aller ein- und ausgehenden EDA-Nachrichten.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `eda_process_id` | UUID FK | Zugehöriger EDA-Prozess (optional; für Korrelation) |
| `message_type` | TEXT | Nachrichtentyp (z. B. `CPDocument`, `CPRequest`) |
| `subject` | TEXT | Betreff / Konversationsreferenz |
| `body` | TEXT | Vollständiger XML-Inhalt |
| `status` | TEXT | `pending` · `sent` · `ack` · `processed` · `error` |
| `from_address` | TEXT DEFAULT '' | Absenderadresse |
| `to_address` | TEXT DEFAULT '' | Empfängeradresse |
| `processed_at` | TIMESTAMPTZ | Zeitpunkt der Verarbeitung |
| `created_at` | TIMESTAMPTZ | Empfangs-/Sendezeitpunkt |

---

### eeg_meter_participations

Mehrfachteilnahme: ein Zählpunkt kann gleichzeitig in mehreren EEGs teilnehmen (österreichisches EAG ab April 2024).

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `meter_point_id` | UUID FK | Zugehöriger Zählpunkt |
| `factor` | NUMERIC | Teilnahmefaktor (0–1) |
| `share_type` | TEXT | `GC` · `RC_R` · `RC_L` · `CC` |
| `valid_from` | DATE | Gültig ab |
| `valid_until` | DATE | Gültig bis (NULL = unbegrenzt) |

---

### onboarding_requests

Mitgliedsanträge über das öffentliche Self-Service-Formular.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `magic_token` | TEXT UNIQUE | Token für E-Mail-Verifikation |
| `status` | TEXT | `pending` · `approved` · `converted` · `rejected` |
| `firstname` | TEXT | Vorname |
| `lastname` | TEXT | Nachname |
| `email` | TEXT | E-Mail-Adresse |
| `strasse` | TEXT | Straße |
| `plz` | TEXT | Postleitzahl |
| `ort` | TEXT | Ort |
| `beitritts_datum` | DATE | Gewünschtes Eintrittsdatum |
| `business_role` | TEXT DEFAULT 'privat' | Unternehmensrolle: `privat` oder `unternehmen` |
| `uid_nummer` | TEXT DEFAULT '' | UID/Steuernummer (bei Firmenmitgliedern) |
| `use_vat` | BOOLEAN DEFAULT FALSE | USt-pflichtig |
| `reminder_sent_at` | TIMESTAMPTZ | Zeitpunkt der 72h-Folgeup-Erinnerung |
| `created_at` | TIMESTAMPTZ | Einreichungszeitpunkt |

---

### onboarding_email_verifications

Magic-Token-basierte E-Mail-Verifizierung beim Onboarding. Wird vor der eigentlichen Antragsstellung angelegt, um die E-Mail-Adresse zu bestätigen.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `email` | TEXT | Zu verifizierende E-Mail-Adresse |
| `name1` | TEXT DEFAULT '' | Vorname (zwischengespeichert) |
| `name2` | TEXT DEFAULT '' | Nachname (zwischengespeichert) |
| `token` | TEXT UNIQUE | Magic-Token |
| `expires_at` | TIMESTAMPTZ | Token-Ablauf |
| `verified_at` | TIMESTAMPTZ | Verifizierungszeitpunkt (NULL = noch nicht verifiziert) |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### member_portal_sessions

Sessions für das Mitgliederportal (Magic-Link-Authentifizierung ohne Passwort).

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `member_id` | UUID FK | Zugehöriges Mitglied |
| `token` | TEXT UNIQUE | Session-Token aus Magic-Link |
| `expires_at` | TIMESTAMPTZ | Ablaufzeitpunkt der Session |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### member_email_campaigns

Gesendete Massen-E-Mail-Kampagnen an EEG-Mitglieder.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `subject` | TEXT | E-Mail-Betreff |
| `html_body` | TEXT | HTML-Inhalt der E-Mail |
| `recipient_count` | INT DEFAULT 0 | Anzahl der Empfänger |
| `attachments_json` | JSONB DEFAULT '[]' | Anhänge als JSON-Array (Dateiname, Pfad, MIME-Typ) |
| `created_at` | TIMESTAMPTZ | Sendezeitpunkt |

---

### eeg_documents

Uploadbare Dokumente (PDFs) pro EEG, die auf der Onboarding-Seite angezeigt werden können.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `title` | TEXT | Anzeigename des Dokuments |
| `description` | TEXT DEFAULT '' | Beschreibungstext |
| `filename` | TEXT | Originaler Dateiname |
| `file_path` | TEXT | Speicherpfad auf dem Server |
| `mime_type` | TEXT DEFAULT 'application/pdf' | MIME-Typ |
| `file_size_bytes` | BIGINT DEFAULT 0 | Dateigröße in Bytes |
| `sort_order` | INT DEFAULT 0 | Anzeigereihenfolge |
| `show_in_onboarding` | BOOLEAN DEFAULT FALSE | Anzeige auf der öffentlichen Onboarding-Seite |
| `created_at` | TIMESTAMPTZ | Hochladezeitpunkt |

---

### user_eeg_assignments

Explizite Zugriffskontrolle: welcher Benutzer darf auf welche EEG zugreifen.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `user_id` | UUID FK | Zugehöriger Benutzer |
| `eeg_id` | UUID FK | Zugehörige EEG |

<div class="tip">
Admin-Benutzer ignorieren diese Tabelle und haben automatisch Zugriff auf alle EEGs der Organisation.
</div>

---

### ea_konten

Kontenplan der E/A-Buchhaltung. Pro EEG vollständig konfigurierbar.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `nummer` | VARCHAR(10) | Kontonummer (eindeutig pro EEG) |
| `name` | VARCHAR(200) | Kontobezeichnung |
| `typ` | VARCHAR(20) DEFAULT 'AUSGABE' | `EINNAHME` · `AUSGABE` · `SONSTIG` |
| `ust_relevanz` | VARCHAR(20) DEFAULT 'KEINE' | `KEINE` · `STEUERBAR` · `VST` · `RC` |
| `standard_ust_pct` | DECIMAL(5,2) | Standard-USt-Satz für dieses Konto |
| `uva_kz` | VARCHAR(10) | UVA-Kennzahl |
| `k1_kz` | VARCHAR(10) DEFAULT '' | FinanzOnline K1-Kennzahl (z. B. `9040`, `9230`) |
| `sortierung` | INT DEFAULT 0 | Anzeigereihenfolge |
| `aktiv` | BOOLEAN DEFAULT TRUE | Konto aktiv |
| `created_at` | TIMESTAMPTZ | Erstellungszeitpunkt |

---

### ea_buchungen

Buchungsjournal der E/A-Buchhaltung. Eine Zeile entspricht einer Geldbewegung (IST-Prinzip: Buchungsdatum = Zahlungsdatum). Unterstützt Soft-Delete für BAO §131-Konformität.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `geschaeftsjahr` | INT | Geschäftsjahr |
| `buchungsnr` | VARCHAR(20) | Buchungsnummer (optional) |
| `zahlung_datum` | DATE | Zahlungsdatum (NULL = noch nicht bezahlt) |
| `beleg_datum` | DATE | Belegdatum |
| `belegnr` | VARCHAR(100) | Belegnummer |
| `beschreibung` | TEXT DEFAULT '' | Buchungstext |
| `konto_id` | UUID FK | Buchungskonto (→ `ea_konten`) |
| `richtung` | VARCHAR(10) | `EINNAHME` oder `AUSGABE` |
| `betrag_brutto` | DECIMAL(12,4) DEFAULT 0 | Bruttobetrag in EUR |
| `ust_code` | VARCHAR(20) DEFAULT 'KEINE' | USt-Code |
| `ust_pct` | DECIMAL(5,2) | USt-Satz |
| `ust_betrag` | DECIMAL(12,4) DEFAULT 0 | USt-Betrag in EUR |
| `betrag_netto` | DECIMAL(12,4) DEFAULT 0 | Nettobetrag in EUR |
| `gegenseite` | VARCHAR(200) | Name der Gegenseite (Bank, Lieferant, etc.) |
| `quelle` | VARCHAR(20) DEFAULT 'manual' | `manual` · `eeg_rechnung` · `eeg_gutschrift` · `bankimport` |
| `quelle_id` | UUID | Referenz auf Quellobjekt (→ `invoices.id` oder `ea_banktransaktionen.id`) |
| `beleg_id` | UUID | Zugehöriger Beleg (→ `ea_belege.id`, nach Upload gesetzt) |
| `notizen` | TEXT | Interne Notizen |
| `erstellt_von` | UUID FK | Erstellender User (→ `users.id`) |
| `deleted_at` | TIMESTAMPTZ | Soft-Delete-Zeitstempel (NULL = aktiv; BAO §131) |
| `deleted_by` | TEXT | E-Mail des löschenden Users |
| `erstellt_am` | TIMESTAMPTZ DEFAULT NOW() | Erstellungszeitpunkt |
| `aktualisiert_am` | TIMESTAMPTZ DEFAULT NOW() | Zeitpunkt der letzten Änderung |

---

### ea_buchungen_changelog

Audit-Trail für alle Mutationen an `ea_buchungen` (BAO §131). Jede Erstellung, Änderung und Löschung wird mit altem/neuem Zustand protokolliert.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `buchung_id` | UUID FK | Referenzierte Buchung (→ `ea_buchungen`) |
| `operation` | TEXT | `create` · `update` · `delete` |
| `changed_at` | TIMESTAMPTZ DEFAULT NOW() | Zeitstempel der Änderung |
| `changed_by` | TEXT DEFAULT '' | E-Mail des ändernden Users |
| `old_values` | JSONB | Vorheriger Zustand als JSON (NULL bei create) |
| `new_values` | JSONB | Neuer Zustand als JSON (NULL bei delete) |
| `reason` | TEXT | Optionaler Änderungsgrund |

---

### ea_belege

Belegverwaltung: hochgeladene Dokumente (z. B. Rechnungen, Quittungen) zu Buchungen.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `buchung_id` | UUID FK | Zugehörige Buchung (→ `ea_buchungen`; ON DELETE SET NULL) |
| `dateiname` | TEXT | Originaler Dateiname |
| `pfad` | TEXT | Speicherpfad auf dem Server |
| `groesse` | INT | Dateigröße in Bytes |
| `mime_typ` | VARCHAR(100) | MIME-Typ |
| `beschreibung` | TEXT | Beschreibung des Belegs |
| `hochgeladen_am` | TIMESTAMPTZ DEFAULT NOW() | Hochladezeitpunkt |
| `hochgeladen_von` | UUID FK | Hochladender User (→ `users.id`) |

---

### ea_uva_perioden

UVA-Perioden (Umsatzsteuervoranmeldung) mit gecachten Kennzahlen.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `jahr` | INT | Kalenderjahr |
| `periodentyp` | VARCHAR(10) DEFAULT 'QUARTAL' | `MONAT` oder `QUARTAL` |
| `periode_nr` | INT | Periodennummer (1–12 bei MONAT, 1–4 bei QUARTAL) |
| `datum_von` | DATE | Periodenstart |
| `datum_bis` | DATE | Periodenende |
| `status` | VARCHAR(20) DEFAULT 'entwurf' | `entwurf` oder `eingereicht` |
| `kz_000` | DECIMAL(12,2) DEFAULT 0 | KZ 000 — Gesamtumsatz |
| `kz_022` | DECIMAL(12,2) DEFAULT 0 | KZ 022 — Steuerpflichtige Umsätze (Basis) |
| `kz_029` | DECIMAL(12,2) DEFAULT 0 | KZ 029 — Summe Bemessungsgrundlagen |
| `kz_044` | DECIMAL(12,2) DEFAULT 0 | KZ 044 — Steuer 10%-Umsätze |
| `kz_056` | DECIMAL(12,2) DEFAULT 0 | KZ 056 — Ausgangs-USt |
| `kz_057` | DECIMAL(12,2) DEFAULT 0 | KZ 057 — Steuerschuld gem. §19 Abs. 1 UStG (Reverse Charge domestic) |
| `kz_060` | DECIMAL(12,2) DEFAULT 0 | KZ 060 — Vorsteuer gesamt |
| `kz_065` | DECIMAL(12,2) DEFAULT 0 | KZ 065 — Vorsteuer RC (§19) |
| `kz_066` | DECIMAL(12,2) DEFAULT 0 | KZ 066 — Steuerschuld RC |
| `kz_083` | DECIMAL(12,2) DEFAULT 0 | KZ 083 — RC Bemessungsgrundlage |
| `zahllast` | DECIMAL(12,2) DEFAULT 0 | Zahllast (positiv = Zahlung an FA, negativ = Gutschrift) |
| `eingereicht_am` | TIMESTAMPTZ | Einreichungszeitpunkt bei FinanzOnline |
| `erstellt_am` | TIMESTAMPTZ DEFAULT NOW() | Erstellungszeitpunkt |

---

### ea_banktransaktionen

Importierte Banktransaktionen aus MT940- oder CAMT.053-Kontoauszügen. Dienen dem automatischen und manuellen Abgleich mit EA-Buchungen.

| Spalte | Typ | Beschreibung |
|--------|-----|-------------|
| `id` | UUID PK | Primärschlüssel |
| `eeg_id` | UUID FK | Zugehörige EEG |
| `import_am` | TIMESTAMPTZ DEFAULT NOW() | Importzeitpunkt |
| `import_format` | VARCHAR(20) DEFAULT 'MT940' | `MT940` · `CAMT053` · `CSV` |
| `konto_iban` | VARCHAR(34) | IBAN des Bankkontos |
| `buchungsdatum` | DATE | Buchungsdatum der Bank |
| `valutadatum` | DATE | Valutadatum |
| `betrag` | DECIMAL(12,4) | Betrag in EUR (positiv = Eingang, negativ = Ausgang) |
| `waehrung` | CHAR(3) DEFAULT 'EUR' | Währung |
| `verwendungszweck` | TEXT | Verwendungszweck |
| `auftraggeber_empfaenger` | VARCHAR(200) | Name des Auftraggebers oder Empfängers |
| `referenz` | TEXT | Bankreferenz oder EndToEndId |
| `matched_buchung_id` | UUID FK | Gematchte EA-Buchung (→ `ea_buchungen`; ON DELETE SET NULL) |
| `match_konfidenz` | DECIMAL(5,2) | Konfidenz des automatischen Matchings (0–100) |
| `match_status` | VARCHAR(20) DEFAULT 'offen' | `offen` · `auto` · `bestaetigt` · `ignoriert` |

---

## Hinweis: Energieeinheiten in `wh_*`-Spalten

<div class="warning">
Die Spalten `wh_total`, `wh_self` und `wh_community` in der Tabelle `energy_readings` tragen historisch bedingt ein `wh_`-Präfix, speichern die Werte jedoch in **Kilowattstunden (kWh)** — nicht in Wattstunden. Es darf **keine Division durch 1000** vorgenommen werden. Die Referenzimplementierung für die Anzeige findet sich in `web/components/energy-charts.tsx` (`fmtKwh()`): Werte werden direkt als kWh dargestellt und erst ab 100.000 kWh in MWh umgerechnet.
</div>
