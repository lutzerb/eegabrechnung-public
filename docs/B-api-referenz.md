# API-Referenz

---

## Basis-URL

| Umgebung | URL |
|----------|-----|
| Lokal (extern, Browser/curl) | `http://localhost:8101` |
| Intern (Docker-Netzwerk, Server-Side) | `http://eegabrechnung-api:8080` |

Alle Endpunkte beginnen mit dem Pfad-Präfix `/api/v1/`.

---

## Authentifizierung

Die meisten Endpunkte erfordern einen **Bearer-Token** im `Authorization`-Header. Der Token wird über den Login-Endpunkt bezogen und ist 8 Stunden gültig.

```bash
# Token holen
TOKEN=$(curl -s -X POST http://localhost:8101/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@eeg.at","password":"admin"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")

# Token verwenden
curl -H "Authorization: Bearer $TOKEN" http://localhost:8101/api/v1/eegs
```

<div class="tip">
Endpunkte unter `/api/v1/public/` sind ohne Authentifizierung erreichbar. Das Mitgliederportal verwendet statt eines Bearer-Tokens einen Magic-Link-Token.
</div>

---

## Auth

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/api/v1/auth/login` | E-Mail + Passwort → HS256-JWT | — |

**Request-Body:**
```json
{ "email": "admin@eeg.at", "password": "admin" }
```

**Response:**
```json
{ "token": "<jwt>" }
```

---

## EEGs

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs` | Alle EEGs der Organisation auflisten | Bearer |
| `POST` | `/api/v1/eegs` | Neues EEG anlegen | Bearer (admin) |
| `GET` | `/api/v1/eegs/{eegId}` | EEG-Details abrufen | Bearer |
| `PUT` | `/api/v1/eegs/{eegId}` | EEG-Einstellungen aktualisieren | Bearer (admin) |
| `DELETE` | `/api/v1/eegs/{eegId}` | EEG löschen | Bearer (admin) |
| `GET` | `/api/v1/eegs/{eegId}/logo` | Logo-Bild abrufen | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/logo` | Logo hochladen (`multipart/form-data`: `datei`) | Bearer (admin) |
| `GET` | `/api/v1/eegs/{eegId}/backup` | Vollständigen EEG-Snapshot als JSON herunterladen | Bearer (admin) |
| `POST` | `/api/v1/eegs/{eegId}/restore` | EEG aus JSON-Snapshot wiederherstellen (`multipart/form-data`: `datei`) | Bearer (admin) |
| `GET` | `/api/v1/eegs/{eegId}/search?q=` | Mitglieder, Zählpunkte und Rechnungen durchsuchen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/stats` | Kurzstatistik (Mitglieder, Rechnungen, EDA-Status) | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/gap-alerts` | Zählpunkte mit fehlenden Messdaten auflisten | Bearer |
| `GET` | `/api/v1/public/eegs/{eegId}/info` | Öffentliche EEG-Basisinfo (für Onboarding-Seite) | — |

<div class="warning">
`POST /restore` überschreibt alle bestehenden Daten des EEGs in einer Transaktion. Vorher unbedingt ein Backup anlegen.
</div>

---

## Mitglieder

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/members` | Alle Mitglieder des EEGs auflisten | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/members` | Neues Mitglied anlegen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/members/{memberId}` | Mitglied-Details abrufen | Bearer |
| `PUT` | `/api/v1/eegs/{eegId}/members/{memberId}` | Mitglied aktualisieren | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/members/{memberId}` | Mitglied löschen | Bearer (admin) |
| `POST` | `/api/v1/eegs/{eegId}/members/{memberId}/austritt` | Mitglied abmelden — setzt INACTIVE + triggert CM_REV_SP für alle aktiven Zählpunkte mit `consent_id`; idempotent | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/members/{memberId}/sepa-mandat` | SEPA-Lastschrift-Mandat als PDF herunterladen | Bearer |

**Austritt Request-Body:**
```json
{ "austritt_datum": "YYYY-MM-DD" }
```

**Mitglied-Typen:** `CONSUMER` · `PROSUMER` · `PRODUCER`

**Status:** `ACTIVE` · `INACTIVE`

---

## Zählpunkte

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/api/v1/eegs/{eegId}/members/{memberId}/meter-points` | Neuen Zählpunkt anlegen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/meter-points/{meterPointId}` | Zählpunkt-Details abrufen | Bearer |
| `PUT` | `/api/v1/eegs/{eegId}/meter-points/{meterPointId}` | Zählpunkt aktualisieren | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/meter-points/{meterPointId}` | Zählpunkt löschen | Bearer |

**Richtungen:** `consumption` · `generation`

**Erzeugungstypen:** `PV` · `Windkraft` · `Wasserkraft` · `Biomasse` · `Sonstiges`

---

## Energiedaten

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/api/v1/eegs/{eegId}/import/stammdaten` | Stammdaten aus XLSX importieren (`multipart/form-data`) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/import/energiedaten` | Energiedaten aus XLSX importieren (`multipart/form-data`) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/import/energiedaten/preview` | Importvorschau — vergleicht XLSX-Zeilen mit DB-Bestand | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/readings/coverage` | Tagesabdeckung der Messdaten abrufen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/export/stammdaten` | Stammdaten als XLSX exportieren | Bearer |

**Qualitätsstufen:** `L0` · `L1` · `L2` · `L3` (L3 wird von der Abrechnung ausgeschlossen)

**Quellen:** `xlsx` · `eda`

---

## Berichte / Energie-Analytik

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/reports/energy` | Monatliche Energieübersicht | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/reports/members` | Mitglieder-Statistik | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/reports/annual` | Jahresbericht | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/energy/summary` | Energiezusammenfassung (gefiltert nach Zeitraum und Granularität) | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/energy/members` | Rohe Energie-Zeitreihe pro Mitglied | Bearer |

**Query-Parameter für Energie-Endpunkte:**

| Parameter | Werte | Beschreibung |
|-----------|-------|-------------|
| `from` | `YYYY-MM-DD` | Startdatum |
| `to` | `YYYY-MM-DD` | Enddatum |
| `granularity` | `year` · `month` · `day` · `15min` | Zeitliche Auflösung |

---

## Tarifpläne

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/tariffs` | Alle Tarifpläne auflisten | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/tariffs` | Neuen Tarifplan anlegen | Bearer (admin) |
| `GET` | `/api/v1/eegs/{eegId}/tariffs/{scheduleId}` | Tarifplan-Details abrufen | Bearer |
| `PUT` | `/api/v1/eegs/{eegId}/tariffs/{scheduleId}` | Tarifplan aktualisieren | Bearer (admin) |
| `DELETE` | `/api/v1/eegs/{eegId}/tariffs/{scheduleId}` | Tarifplan löschen | Bearer (admin) |
| `PUT` | `/api/v1/eegs/{eegId}/tariffs/{scheduleId}/entries` | Tarifeinträge (Preiszeilen) setzen | Bearer (admin) |
| `POST` | `/api/v1/eegs/{eegId}/tariffs/{scheduleId}/activate` | Tarifplan aktivieren | Bearer (admin) |
| `DELETE` | `/api/v1/eegs/{eegId}/tariffs/{scheduleId}/activate` | Tarifplan deaktivieren | Bearer (admin) |

<div class="tip">
Pro EEG kann nur ein Tarifplan gleichzeitig aktiv sein. Abrechnungszeiträume ohne Tarifplan-Abdeckung fallen auf die Fallback-Preise des EEGs zurück (`energy_price` / `producer_price`).
</div>

---

## Abrechnungsläufe

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/api/v1/eegs/{eegId}/billing/run` | Neuen Abrechnungslauf erstellen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/billing/runs` | Alle Abrechnungsläufe auflisten | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/billing/runs/{runId}/invoices` | Rechnungen eines Laufs auflisten | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/billing/runs/{runId}/zip` | Alle PDFs eines Laufs als ZIP herunterladen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/billing/runs/{runId}/export` | Lauf als XLSX exportieren | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/billing/runs/{runId}/send-all` | Alle Rechnungen eines Laufs versenden | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/billing/runs/{runId}/finalize` | Abrechnungslauf finalisieren | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/billing/runs/{runId}/cancel` | Abrechnungslauf stornieren | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/billing/runs/{runId}` | Abrechnungslauf (Entwurf) löschen | Bearer (admin) |

**Status-Lifecycle:** `draft` → `finalized` → `cancelled`

<div class="warning">
Das Erstellen eines Abrechnungslaufs für einen bereits abgerechneten Zeitraum wird mit HTTP 409 abgelehnt. Zur Umgehung steht der `force`-Parameter im Request-Body bereit (nur in Ausnahmefällen verwenden).
</div>

---

## Rechnungen

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/invoices` | Alle Rechnungen auflisten; `?sepa_returned=true` filtert auf Rücklastschriften | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/invoices/send-all` | Alle ausstehenden Rechnungen versenden | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/invoices/{invoiceId}/pdf` | Rechnungs-PDF herunterladen | Bearer |
| `PATCH` | `/api/v1/eegs/{eegId}/invoices/{invoiceId}/status` | Rechnungsstatus setzen | Bearer |
| `PATCH` | `/api/v1/eegs/{eegId}/invoices/{invoiceId}/sepa-return` | Rücklastschrift manuell erfassen oder löschen | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/invoices/{invoiceId}/resend` | Rechnung erneut versenden | Bearer |

**SEPA-Return Body** (zum Setzen):
```json
{ "sepa_return_at": "YYYY-MM-DD", "sepa_return_reason": "AC01", "sepa_return_note": "..." }
```
Zum Löschen: leerer Body oder `null`.

**Dokumenttypen:** `invoice` · `credit_note`

**Status:** `draft` · `finalized` · `sent` · `paid` · `cancelled`

---

## SEPA

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/sepa/pain001` | SEPA pain.001 (Überweisungen) herunterladen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/sepa/pain008` | SEPA pain.008 (Lastschriften) herunterladen | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/sepa/camt054` | CAMT.054-Bankdatei importieren — matched Rücklastschriften automatisch per `EndToEndId` = InvoiceUUID | Bearer |

---

## Buchhaltung (DATEV-Export)

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/accounting/export` | DATEV Buchungsstapel oder XLSX exportieren | Bearer |

**Query-Parameter:**

| Parameter | Werte | Beschreibung |
|-----------|-------|-------------|
| `from` | `YYYY-MM-DD` | Startdatum |
| `to` | `YYYY-MM-DD` | Enddatum |
| `format` | `datev` · `xlsx` | Exportformat |

---

## E/A-Buchhaltung

Alle Endpunkte erfordern einen Bearer-Token.

### Einstellungen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/settings` | EA-Einstellungen abrufen (Steuernummer, Finanzamt, UVA-Periodentyp) |
| `PUT` | `/api/v1/eegs/{eegId}/ea/settings` | EA-Einstellungen speichern |

**Settings-Body:**
```json
{ "steuernummer": "12/345/6789", "finanzamt": "FA Wien 1", "uva_periodentyp": "QUARTAL" }
```
`uva_periodentyp`: `QUARTAL` · `MONAT`

### Kontenplan

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/konten` | Kontenplan auflisten; `?aktiv=false` schließt inaktive Konten ein | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/ea/konten` | Konto anlegen | Bearer |
| `PUT` | `/api/v1/eegs/{eegId}/ea/konten/{kontoId}` | Konto bearbeiten | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/ea/konten/{kontoId}` | Konto löschen | Bearer |

**Konto Body:**
```json
{
  "nummer": "4000",
  "name": "Erlöse Energielieferung",
  "typ": "einnahme",
  "ust_relevanz": "steuerpflichtig",
  "standard_ust_pct": 13.0,
  "uva_kz": "022",
  "k1_kz": "777",
  "sortierung": 10,
  "aktiv": true
}
```

### Buchungen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/buchungen` | Buchungen auflisten (siehe Query-Parameter) |
| `POST` | `/api/v1/eegs/{eegId}/ea/buchungen` | Buchung anlegen |
| `GET` | `/api/v1/eegs/{eegId}/ea/buchungen/export` | XLSX-Export; Query: `?jahr=&konto_id=` |
| `GET` | `/api/v1/eegs/{eegId}/ea/buchungen/{buchungId}` | Buchungsdetail (inkl. Belege, `deleted_at`/`deleted_by`) |
| `PUT` | `/api/v1/eegs/{eegId}/ea/buchungen/{buchungId}` | Buchung bearbeiten; optionales Feld `reason` wird in Changelog protokolliert |
| `DELETE` | `/api/v1/eegs/{eegId}/ea/buchungen/{buchungId}` | Soft-Delete; optionaler Body `{"reason":"..."}` — schreibt Changelog-Eintrag |
| `GET` | `/api/v1/eegs/{eegId}/ea/buchungen/{buchungId}/changelog` | Audit-Trail einer Buchung (BAO §131); gibt `[]EABuchungChangelog` geordnet nach `changed_at ASC` zurück |

**Query-Parameter für `GET /buchungen`:**

| Parameter | Werte | Beschreibung |
|-----------|-------|-------------|
| `jahr` | `YYYY` | Geschäftsjahr filtern |
| `von` | `YYYY-MM-DD` | Datumsbereich Anfang |
| `bis` | `YYYY-MM-DD` | Datumsbereich Ende |
| `konto_id` | UUID | Auf Konto einschränken |
| `richtung` | `einnahme` · `ausgabe` | Richtung filtern |
| `bezahlt` | `true` · `false` | Bezahltstatus filtern |
| `incl_deleted` | `true` | Soft-gelöschte Buchungen einschließen |

**Buchung Body (POST/PUT):**
```json
{
  "beleg_datum": "2025-03-15",
  "zahlung_datum": "2025-03-20",
  "konto_id": "<uuid>",
  "beschreibung": "Stromlieferung März",
  "betrag_brutto": 123.45,
  "ust_code": "UST13",
  "richtung": "einnahme",
  "gegenseite": "E-Werk Muster GmbH",
  "notizen": "Interne Notiz",
  "reason": "Korrektur Betrag"
}
```

### EEG-weiter Changelog

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/changelog` | EEG-weiter Changelog aller Buchungsmutationen |

**Query-Parameter:**

| Parameter | Werte | Beschreibung |
|-----------|-------|-------------|
| `von` | `YYYY-MM-DD` | Datumsbereich Anfang |
| `bis` | `YYYY-MM-DD` | Datumsbereich Ende |
| `user` | UUID | Auf Benutzer einschränken |
| `operation` | `create` · `update` · `delete` | Auf Operationstyp einschränken |
| `limit` | Zahl | Anzahl Einträge (Standard 200, max 500) |
| `offset` | Zahl | Paginierungs-Offset |

### Belege

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `POST` | `/api/v1/eegs/{eegId}/ea/belege` | Beleg hochladen; `multipart/form-data`: `datei` + `buchung_id` |
| `GET` | `/api/v1/eegs/{eegId}/ea/belege/{belegId}` | Beleg herunterladen |
| `DELETE` | `/api/v1/eegs/{eegId}/ea/belege/{belegId}` | Beleg löschen |

### Auswertungen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/saldenliste` | Saldenliste; `?jahr=YYYY`; gibt `[]EASaldenlisteEintrag` zurück |
| `GET` | `/api/v1/eegs/{eegId}/ea/kontenblatt/{kontoId}` | Kontenblatt; `?von=YYYY-MM-DD&bis=YYYY-MM-DD`; gibt `{konto, eintraege, summe}` zurück |
| `GET` | `/api/v1/eegs/{eegId}/ea/jahresabschluss` | Jahresabschluss (EAR); `?jahr=YYYY&format=xlsx`; gibt `{jahr, total_einnahmen, total_ausgaben, ueberschuss, einnahmen[], ausgaben[]}` zurück |

### UVA (Umsatzsteuervoranmeldung)

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/uva` | UVA-Perioden auflisten |
| `POST` | `/api/v1/eegs/{eegId}/ea/uva` | UVA-Periode erstellen |
| `GET` | `/api/v1/eegs/{eegId}/ea/uva/{uvaId}/kennzahlen` | UVA-Kennzahlen aus Buchungen berechnen |
| `PATCH` | `/api/v1/eegs/{eegId}/ea/uva/{uvaId}/eingereicht` | UVA als bei FinanzOnline eingereicht markieren |
| `GET` | `/api/v1/eegs/{eegId}/ea/uva/{uvaId}/export` | FinanzOnline XML-Export |

### Jahreserklärungen

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/erklaerungen/u1` | U1 Jahres-USt-Zusammenfassung; `?jahr=YYYY` |
| `GET` | `/api/v1/eegs/{eegId}/ea/erklaerungen/k1` | K1 Körperschaftsteuer-Basis (wirtschaftlicher Geschäftsbetrieb); `?jahr=YYYY` |
| `GET` | `/api/v1/eegs/{eegId}/ea/erklaerungen/k2` | K2 Körperschaftsteuer-Erklärung als FinanzOnline XML; `?jahr=YYYY` |

### Rechnungsimport

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/eegs/{eegId}/ea/import/preview` | Vorschau: EEG-Rechnungen die noch nicht als Buchungen importiert wurden; `?jahr=YYYY` |
| `POST` | `/api/v1/eegs/{eegId}/ea/import/rechnungen` | Ausgewählte Rechnungen als Buchungen importieren |

**Import Body:**
```json
{ "invoice_ids": ["<uuid>", "<uuid>"] }
```

### Bankkontoabgleich

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `POST` | `/api/v1/eegs/{eegId}/ea/bank/import` | Kontoauszug importieren; `multipart/form-data`: `datei` + `format=mt940\|camt053` |
| `GET` | `/api/v1/eegs/{eegId}/ea/bank/transaktionen` | Bank-Transaktionen auflisten; `?status=offen\|ignoriert` |
| `POST` | `/api/v1/eegs/{eegId}/ea/bank/match` | Transaktion mit Buchung matchen |
| `DELETE` | `/api/v1/eegs/{eegId}/ea/bank/transaktionen/{transaktionId}` | Transaktion als ignoriert markieren |

**Match Body:**
```json
{ "transaktion_id": "<uuid>", "buchung_id": "<uuid>" }
```

---

## EDA (Energiedaten-Austausch)

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/eda/processes` | Alle EDA-Prozesse auflisten | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/eda/anmeldung` | Zählpunkt beim Netzbetreiber anmelden (EC_REQ_ONL) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/eda/widerruf` | Widerruf (CM_REV_SP) — erfordert gespeicherte `consent_id` am Zählpunkt | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/eda/teilnahmefaktor` | Teilnahmefaktor ändern (EC_PRTFACT_CHG) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/eda/zaehlerstandsgang` | Historische Messdaten anfordern (EC_REQ_PT) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/eda/podlist` | Zählpunktliste abfragen (EC_PODLIST) | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/eda/messages` | EDA-Nachrichtenprotokoll abrufen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/eda/messages/{msgId}/xml` | Rohe XML einer EDA-Nachricht herunterladen | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/eda/errors` | EDA-Fehlermeldungen (Dead-Letter) auflisten | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/eda/poll-now` | IMAP-Poll sofort auslösen (MAIL-Modus) | Bearer |
| `GET` | `/api/v1/eda/worker-status` | Worker-Gesundheitsstatus (`last_poll_at`, `last_error`) | Bearer |

**Widerruf Request-Body:**
```json
{ "meter_point_id": "<uuid>", "valid_from": "YYYY-MM-DD" }
```

**Zaehlerstandsgang Request-Body:**
```json
{ "zaehlpunkt": "AT...", "date_from": "2026-03-01", "date_to": "2026-03-31" }
```

**Prozess-Status-Lifecycle:** `pending` → `sent` → `first_confirmed` → `confirmed` / `completed` / `rejected` / `error`

---

## Mehrfachteilnahme

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/participations` | Alle Teilnahmen auflisten | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/participations` | Neue Teilnahme anlegen | Bearer |
| `PUT` | `/api/v1/eegs/{eegId}/participations/{id}` | Teilnahme aktualisieren | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/participations/{id}` | Teilnahme löschen | Bearer |

**Share-Typen:** `GC` · `RC_R` · `RC_L` · `CC`

---

## Onboarding

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/api/v1/public/eegs/{eegId}/onboarding` | Mitgliedsantrag einreichen (öffentlich) | — |
| `POST` | `/api/v1/public/eegs/{eegId}/onboarding/verify-email` | E-Mail-Verifizierungs-Token anfordern | — |
| `POST` | `/api/v1/public/eegs/{eegId}/onboarding/verify/{token}` | E-Mail-Adresse bestätigen | — |
| `GET` | `/api/v1/public/onboarding/status/{token}` | Status eines Antrags per Magic-Token prüfen | — |
| `POST` | `/api/v1/public/onboarding/resend-token` | Bestätigungs-E-Mail erneut senden | — |
| `GET` | `/api/v1/eegs/{eegId}/onboarding` | Onboarding-Anträge auflisten | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/onboarding/{id}` | Einzelnen Antrag abrufen | Bearer |
| `PATCH` | `/api/v1/eegs/{eegId}/onboarding/{id}` | Antrag-Status aktualisieren (z. B. genehmigen → convert) | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/onboarding/{id}` | Antrag ablehnen / löschen | Bearer |

---

## EEG-Dokumente

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/documents` | EEG-Dokumente auflisten | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/documents` | Dokument hochladen; `multipart/form-data`: `datei`, `name`, `show_in_onboarding` | Bearer |
| `PATCH` | `/api/v1/eegs/{eegId}/documents/{docId}` | Dokument-Metadaten aktualisieren | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/documents/{docId}/download` | Dokument herunterladen | Bearer |
| `DELETE` | `/api/v1/eegs/{eegId}/documents/{docId}` | Dokument löschen | Bearer |
| `GET` | `/api/v1/public/eegs/{eegId}/documents/{docId}` | Dokument öffentlich herunterladen (für Onboarding-AGB-Links) | — |

---

## Kommunikation / E-Mail-Kampagnen

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/eegs/{eegId}/communications` | Gesendete E-Mail-Kampagnen auflisten | Bearer |
| `GET` | `/api/v1/eegs/{eegId}/communications/{id}` | Kampagnen-Detail (inkl. Empfänger, Anhänge) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/communications` | Kampagne senden; `multipart/form-data`: `subject`, `body` (HTML), `member_ids[]` oder `member_type`, `attachments[]` | Bearer |

**`member_type`-Werte:** `all` · `consumer` · `producer` · `prosumer` · `individual`

---

## Mitgliederportal (Magic-Link)

Diese Endpunkte verwenden **keinen Bearer-Token**, sondern einen zeitlich begrenzten Magic-Link-Token.

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `POST` | `/api/v1/public/portal/request-link` | Magic-Link-E-Mail an Mitglied senden | — |
| `POST` | `/api/v1/public/portal/exchange` | Magic-Link-Token gegen Session-Token tauschen | — |
| `GET` | `/api/v1/public/portal/me` | Profildaten des Mitglieds abrufen | Portal-Session |
| `GET` | `/api/v1/public/portal/energy` | Energie-Zeitreihe des Mitglieds abrufen | Portal-Session |
| `GET` | `/api/v1/public/portal/invoices` | Rechnungsliste des Mitglieds abrufen | Portal-Session |
| `GET` | `/api/v1/public/portal/invoices/{invoiceId}/pdf` | Rechnungs-PDF herunterladen | Portal-Session |
| `GET` | `/api/v1/public/portal/documents` | EEG-Dokumente im Portal anzeigen | Portal-Session |
| `GET` | `/api/v1/public/portal/documents/{docId}` | EEG-Dokument aus Portal herunterladen | Portal-Session |
| `GET` | `/api/v1/public/portal/meter-points` | Zählpunkte des Mitglieds abrufen | Portal-Session |
| `POST` | `/api/v1/public/portal/change-factor` | Teilnahmefaktor-Anfrage stellen | Portal-Session |

---

## OeMAG-Marktpreis

| Methode | Pfad | Beschreibung | Auth |
|---------|------|-------------|------|
| `GET` | `/api/v1/oemag/marktpreis` | Aktuellen OeMAG-Marktpreis abrufen | Bearer |
| `POST` | `/api/v1/oemag/refresh` | OeMAG-Marktpreis aktualisieren (Scraping) | Bearer |
| `POST` | `/api/v1/eegs/{eegId}/oemag/sync` | EEG-Einspeisetarif mit OeMAG-Marktpreis synchronisieren | Bearer |

---

## Benutzerverwaltung

Alle Endpunkte erfordern **Bearer + Admin-Rolle**.

| Methode | Pfad | Beschreibung |
|---------|------|-------------|
| `GET` | `/api/v1/admin/users` | Alle Benutzer der Organisation auflisten |
| `POST` | `/api/v1/admin/users` | Neuen Benutzer anlegen |
| `GET` | `/api/v1/admin/users/{userId}` | Benutzer-Details abrufen |
| `PUT` | `/api/v1/admin/users/{userId}` | Benutzer aktualisieren |
| `DELETE` | `/api/v1/admin/users/{userId}` | Benutzer löschen |
| `GET` | `/api/v1/admin/users/{userId}/eegs` | EEG-Zuweisungen eines Benutzers abrufen |
| `PUT` | `/api/v1/admin/users/{userId}/eegs` | EEG-Zuweisungen eines Benutzers setzen |
