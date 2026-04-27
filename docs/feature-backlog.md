# Feature-Backlog — eegabrechnung

Stand: 2026-04-17. Enthält alle geplanten Features mit Implementierungsdetails.

---

## Erledigte Backlog-Items ✅

### Ursprüngliche Planung (2026-03-21) — alle erledigt
- **B-1** EDA Deadline-Ampel ✅
- **B-2** Rechnungen per E-Mail + Bezahlstatus ✅
- **B-3** Import-UX (Coverage-Ansicht nach Import) ✅
- **B-4** Globale Suche ✅
- **B-5** Mobile-Verbesserungen ✅
- Dashboard mit Alerts & Status ✅
- Mitglieder-Detailseite ✅
- Einstellungen in Tabs ✅

### Sprint 2026-04-11
- **N-1** Mitglieder-Austritt-Workflow ✅ — `POST .../members/{id}/austritt`, setzt INACTIVE + CM_REV_SP für alle aktiven Zählpunkte mit consent_id, Idempotenzprüfung
- **N-2** EDA-Fehler-Benachrichtigung ✅ — E-Mail via EEG-SMTP wenn Prozess auf error/rejected; `error_notification_sent_at` verhindert Duplikate
- **N-4** Automatische Monatsabrechnung ✅ — täglicher Scheduler (06:00 Vienna), Vollständigkeits-Check, Draft-Run, E-Mail-Benachrichtigung; Settings-Tab „Auto-Abrechnung"
- **N-5** SEPA-Rücklastschrift-Tracking ✅ — Phase 1: manuell erfassen (PATCH .../sepa-return, Reason-Dropdown, Notiz); Phase 2: CAMT.054-Import (POST .../sepa/camt054, matched per EndToEndId=InvoiceUUID); Dashboard-Alert; Mig. 050
- **N-6** Lücken-Detektion ✅ — stündlicher GapChecker, E-Mail-Alarm wenn Zählpunkt > N Tage ohne Readings; Dashboard-Alert; oranges Icon in Member-Detailseite; Settings-Tab „Lücken-Alarm"; Mig. 051

### Sprint 2026-04 (E/A-Buchhaltung & weitere Features)

- **N-8** Zählerpunkt-Notizen ✅ — `notes`-Feld auf `meter_points`; Migration 052/053; Inline-Edit auf Mitglieds-Detailseite
- **E/A-Buchhaltung** ✅ — Vollständiges EAR-Modul für EEGs als Verein (wirtschaftlicher Geschäftsbetrieb, EAR bis € 700.000 Umsatz): Kontenplan, Buchungen, Belege, Rechnungsimport, Bankimport (MT940/CAMT.053), Saldenliste, Jahresabschluss, UVA, U1/K1/K2, BAO §131 Audit-Trail; Migrationen 056–066
- **K2 FinanzOnline XML-Export** ✅ — FinanzOnline-konforme XML-Körperschaftsteuer-Erklärung für den wirtschaftlichen Geschäftsbetrieb
- **BAO §131 Audit-Trail** ✅ — Revisionssicherer Changelog aller EA-Buchungsänderungen (create/update/delete) mit old/new JSON-Snapshots; Migration 066
- **SEPA-Mandat beim Onboarding** ✅ — SEPA-Lastschrift-Berechtigung im Onboarding-Formular; revisionssichere Speicherung (Zeitstempel, IP, Mandatstext); Migration 061
- **Demo-Modus** ✅ — `is_demo`-Flag auf EEGs; blockiert E-Mail-Versand; Migration 038
- **Per-EEG SMTP/IMAP-Credentials** ✅ — AES-256-GCM-verschlüsselte Credentials in DB; Migration 043
- **E-Mail-Kampagnen** ✅ — Massen-E-Mails an Mitglieder (alle/nach Typ/individuell); Migration 032
- **EEG-Dokumente für Onboarding** ✅ — Uploadbare PDFs auf Onboarding-Seite; Migration 033
- **E-Mail-Verifizierung beim Onboarding** ✅ — Magic-Token-Verifikation; Migration 034
- **Kontoauszug-Import (Bank)** ✅ — MT940/CAMT.053 Import + Transaktionsmatching; Migration 057

---

## Offene Features

---

### N-3 · Zählpunkte ohne Readings nach Anmeldung erkennen

**Hintergrund:** Die ABSCHLUSS_ECON-Bestätigung vom NB bedeutet, dass der Zählpunkt im EDA-System registriert ist. Die eigentliche Datenfreigabe (15-Minuten-Takt im NB-Kundenportal) ist je nach NB ein separater Schritt des Mitglieds — oder wird automatisch aktiviert. Ob Daten fließen, sieht man erst wenn tatsächlich Readings eintreffen.

Dieses Feature überlappt stark mit **N-6 (Lücken-Detektion)**: ein Zählpunkt ohne Readings 7+ Tage nach `registriert_seit` ist exakt dieselbe Erkennung. N-3 als separates Feature ist damit redundant.

**Empfehlung:** N-3 in N-6 integrieren. Die Lücken-Detektion filtert bereits auf `registriert_seit IS NOT NULL` — für neu registrierte Zählpunkte greift der Alert also automatisch. Kein separates `datenfreigabe_erteilt`-Feld nötig.

**Status: In N-6 aufgegangen — separat nicht umsetzen.**

---

### N-7 · Jahresbericht (intern, kein gesetzlicher Pflichtbericht)

**Hinweis:** Es konnte keine konkrete gesetzliche Grundlage für eine jährliche Berichtspflicht österreichischer EEGs gegenüber E-Control gefunden werden. Vor Implementierung mit EDA GmbH oder Rechtsberater klären ob und in welcher Form eine Meldepflicht besteht. Das Feature ist als freiwilliger interner Bericht trotzdem nützlich — z.B. für Mitgliederversammlungen oder eigene Dokumentation.

**Nutzen:** Zusammenfassende Jahresübersicht über Energiemengen, Mitglieder und Abrechnungen — für interne Zwecke oder als Basis für eventuelle Behördenmeldungen.

**Inhalt:**
- EEG-Stammdaten (Name, Gründungsdatum, Adresse, UID)
- Mitgliederliste mit Typ (Verbraucher/Erzeuger/Prosumer), Beitrittsdatum, Zählpunktnummer
- Energiestatistik: Gesamtbezug, -einspeisung, Eigendeckungsgrad, Gemeinschaftsanteil
- Abrechnungszusammenfassung: Anzahl Rechnungen, Gesamtbetrag

**DB:** Keine Migration — alle Daten bereits vorhanden.

**Backend:**
- Neuer Endpunkt: `GET /api/v1/eegs/{eegId}/reports/annual?year=2025&format=xlsx|pdf`
- Handler aggregiert Daten aus mehreren Repositories
- XLSX: via `excelize` (bereits als Dependency vorhanden für DATEV-Export)
- PDF: analog zu bestehenden Invoice-PDFs via `fpdf`

**Frontend:**
- Neuer Menüpunkt unter Berichte: „Jahresbericht"
- Formular: Jahresauswahl + Format (XLSX / PDF) → Download

**Aufwand:** 2 Tage (abhängig von genauen E-Control-Anforderungen; ggf. Abstimmung nötig)

---

### N-9 · Audit-Log — Teilweise umgesetzt

**Nutzen:** Nachvollziehbarkeit: wer hat wann was geändert (Tarif geändert, Mitglied bearbeitet, Abrechnung storniert). Wichtig bei mehreren Benutzern und für Haftungsfragen.

Für die **E/A-Buchhaltung** wurde ein vollständiger BAO §131-konformer Audit-Trail implementiert (Migration 066). Ein EEG-weiter Audit-Log für andere Bereiche (Tarife, Mitglieder, Abrechnungen) ist noch nicht umgesetzt.

**Status: EA-Buchungen ✅ abgedeckt; allgemeiner Audit-Log noch offen**

**Restumfang (allgemeiner Audit-Log):**

**DB-Migration:**
```sql
CREATE TABLE audit_log (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  eeg_id      uuid REFERENCES eegs(id),
  table_name  text NOT NULL,
  record_id   uuid,
  action      text NOT NULL CHECK (action IN ('create','update','delete')),
  old_value   jsonb,
  new_value   jsonb,
  user_id     uuid REFERENCES users(id),
  user_email  text,
  created_at  timestamptz NOT NULL DEFAULT NOW()
);
CREATE INDEX audit_log_eeg_id_created_at ON audit_log(eeg_id, created_at DESC);
CREATE INDEX audit_log_record_id ON audit_log(record_id);
```

**Backend:**
- Neue Funktion `audit.Log(ctx, db, entry AuditEntry)` in eigenem Package `api/internal/audit/`
- In den wichtigsten Handlern aufrufen: Member create/update/delete, Tarif create/update, Billing Run finalize/storno, EEG-Settings update
- User-ID aus JWT-Claims in Context ziehen (bereits vorhanden via `auth.ClaimsFromContext`)
- Neuer Endpunkt: `GET /api/v1/eegs/{eegId}/audit?table=&record_id=&limit=50`

**Frontend:**
- EEG-Settings → neuer Tab „Protokoll"
- Tabelle: Zeitpunkt | Benutzer | Aktion | Objekt | Details
- Filter nach Tabelle / Zeitraum
- Optional: Audit-Tab auf Mitglieder-Detailseite für änderungsrelevante Events

**Aufwand:** 2 Tage

---

### Ponton X/P Migration

**Trigger:** E-Mail-Gateway erlaubt max. 2.500 Nachrichten/Monat (ein- + ausgehend). Ab ~40 Mitgliedern mit je 2 Zählpunkten wird diese Grenze erreicht. Danach ist KEP-Betreiber-Anbindung via Ponton X/P Pflicht (lt. EDA GmbH Regelwerk).

**Status: Phase 1 (Verträge) noch nicht gestartet.**

Vollständiger Migrationsplan: siehe CLAUDE.md → Abschnitt „Ponton X/P Migration Plan".

**Aufwand Code (Phase 3+4):** ~1 Tag. Zeitfresser ist Phase 1 (Vertragsabschluss, Zertifikat-Beantragung, SIA-Einrichtung): ca. 2–4 Wochen.

---

## Priorisierungsvorschlag (offene Items)

| # | Feature | Nutzen | Aufwand | Priorität |
|---|---------|--------|---------|-----------|
| ~~N-6~~ | ~~Lücken-Detektion~~ | ✅ erledigt | — | — |
| ~~N-8~~ | ~~Zählerpunkt-Notizen~~ | ✅ erledigt | — | — |
| ~~N-5~~ | ~~SEPA-Rücklastschrift~~ | ✅ erledigt | — | — |
| N-7 | Jahresbericht | Niedrig — Anforderungen unklar | 2 Tage | ⭐ |
| N-9 | Audit-Log (Restumfang) | Niedrig — nice to have | 2 Tage | ⭐ |
| Ponton X/P | EDA-Transport >2.500 Nachrichten/Monat | Hoch ab Skalierung | Phase 1: extern | ⭐⭐ (wenn relevant) |
