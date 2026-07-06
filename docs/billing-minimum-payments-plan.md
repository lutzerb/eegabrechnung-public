# Mindestbetraege und Zahlungs-Ledger fuer Kleinbetragsrechnungen

## Zusammenfassung

Ziel ist, monatliche Rechnungen weiterhin sauber und zeitnah zu erzeugen, den tatsaechlichen
Zahlungsprozess aber von der Rechnung zu entkoppeln. Die Rechnung bleibt der fachliche und
buchhalterische Beleg. Zusaetzlich wird pro Mitglied ein Zahlungs-Ledger eingefuehrt, das offene
Salden, Zahlungsfaelligkeit, Einzuege, Auszahlungen, Ruecklastschriften und Vortraege verwaltet.
Der Zahlungslauf greift erst, wenn der saldierte Betrag eines Mitglieds die konfigurierte
Mindestsumme erreicht.

Gewaehlte Defaults:

- Schwelle gilt netto pro Mitglied ueber einen laufenden Saldo.
- Monatsrechnungen bleiben sichtbar und werden wie bisher erzeugt.
- Buchhaltung bleibt auf Rechnungsebene bestehen, zusaetzlich kommen getrennte
  Zahlungs-/Settlement-Ereignisse dazu.

## Wichtige Aenderungen

- Neues Zahlungsmodell einfuehren:
  - Neue Ledger-Ebene pro Mitglied innerhalb einer EEG, unabhaengig von einzelnen Rechnungen.
  - Neue Entitaeten:
    - `member_payment_accounts`: ein Konto je `eeg_id + member_id`
    - `payment_ledger_entries`: unveraenderliche Bewegungen. Zulaessige Entry-Typen:
      - `opening_balance` — einmaliger Eroefffnungssaldo bei Migration bestehender EEGs (ab Stichtag)
      - `invoice_posted` — Forderung gegen Mitglied bei finalisierter Rechnung
      - `credit_note_posted` — Gutschrift zugunsten Mitglied bei finalisierter Gutschrift
      - `direct_debit_created` — SEPA-Einzug initiiert (payment_run_item erzeugt)
      - `transfer_created` — SEPA-Auszahlung initiiert (payment_run_item erzeugt)
      - `payment_booked` — Zahlung erfolgreich verbucht (z. B. CAMT.054-Eingang)
      - `direct_debit_returned` — Ruecklastschrift; oeffnet Saldo wieder
      - `manual_adjustment` — manuelle Korrektur durch Admin
      - **Hinweis:** `payment_due` wird nicht als eigener Typ gefuehrt — die Forderung entsteht
        bereits durch `invoice_posted`. Ein separater `payment_due`-Eintrag wuerde dieselbe
        Bewegung doppelt abbilden und ist daher nicht vorgesehen.
    - `payment_runs`: gruppiert einen tatsaechlichen Einzugs- oder Auszahlungslauf; Status:
      `preview` → `finalized` → `submitted` → `booked`
    - `payment_run_items`: konkrete Auszahlungs-/Einzugspositionen je Mitglied mit Referenzen
      auf die abgedeckten Ledger-Eintraege
    - `sepa_mandates`: SEPA-Lastschriftmandate je Mitglied (siehe Abschnitt SEPA-Mandate)
- Rechnungen fachlich von Zahlung entkoppeln:
  - `invoices.status` nicht mehr als Proxy fuer Zahlung verwenden.
  - Rechnungsstatus nur noch fuer Dokumentlebenszyklus: `draft`, `finalized`, `cancelled`.
    `paid` darf nicht mehr die primaere Wahrheit sein; bestehende `paid`-Datensaetze werden
    bei der Migration in Ledger-Eintrag `opening_balance` oder `payment_booked` umgewandelt
    (siehe Abschnitt Stichtag-Migration).
  - Zahlungserfuellung kommt aus dem Ledger bzw. aus einem berechneten `settlement_status`
    auf Rechnungs-/Mitgliedsebene.
  - Der bestehende Filter `GET /invoices?sepa_returned=true` (migration 050, Felder
    `sepa_return_at/reason/note`) bleibt vorerst bestehen; CAMT.054-Import schreibt kuenftig
    *zusaetzlich* einen `direct_debit_returned`-Ledger-Eintrag. Die Felder auf `invoices`
    werden in v2 deprecated.
- EEG-Konfiguration erweitern:
  - neue Settings auf EEG-Ebene:
    - `minimum_direct_debit_amount_eur` — Schwelle fuer SEPA-Einzug (Konsumenten)
    - `minimum_payout_amount_eur` — Schwelle fuer SEPA-Auszahlung (Produzenten)
    - `payment_netting_enabled` — Bezug und Einspeisung eines PROSUMERs saldieren; default
      `true`; bei `false` kommen Einzug und Auszahlung separat
    - `payment_holdback_max_age_months` — Obergrenze fuer den Vortragszeitraum; v1 nicht
      implementieren, aber Spalte anlegen (NULL = unbegrenzt) damit spaeter keine Migration noetig
- Ableitungen und Regeln:
  - Bei jeder finalen Monatsrechnung entsteht genau ein `invoice_posted`-Ledger-Eintrag.
    **`PostInvoiceToLedger` muss idempotent sein**: UNIQUE-Constraint auf
    `(invoice_id, entry_type)` mit `ON CONFLICT DO NOTHING` verhindert Doppelbuchung bei
    Worker-Retries.
  - Positive Rechnung erhoeht Forderung gegen das Mitglied.
  - Negative Rechnung/Gutschrift reduziert Forderung oder erzeugt Guthaben.
  - Der zahlungswirksame Saldo eines Mitglieds ist die Summe aller offenen Ledger-Bewegungen.
  - Erst wenn `abs(offener Saldo) >= Schwelle`, wird das Mitglied in einen Zahlungslauf aufgenommen.
  - **PROSUMER-Richtungslogik**: Nach dem Netting bestimmt das Vorzeichen des Saldos die
    Richtung des Payment Run Items: positiver Saldo (Mitglied schuldet der EEG) → `pain.008`
    Lastschrift; negativer Saldo (EEG schuldet dem Mitglied) → `pain.001` Ueberweisung. Diese
    Logik muss in `CreatePaymentRunPreview` explizit implementiert sein.
  - Ruecklastschriften erzeugen `direct_debit_returned`-Ledger-Eintrag und oeffnen den Saldo
    wieder, statt nur Felder auf der Rechnung zu setzen.
- SEPA-Flows umbauen:
  - `pain.008` und `pain.001` kuenftig aus `payment_run_items` statt direkt aus `invoices` erzeugen.
  - Verwendungszweck/Referenz muss auf Rechnungen rueckfuehrbar bleiben; daher in
    `payment_run_items` Zuordnung zu den enthaltenen Ledger-Eintraegen speichern.
  - CAMT.054 und manuelle Zahlungsverbuchung schreiben kuenftig in Payment-Run-Items und Ledger
    (zusaetzlich zur bestehenden `sepa_return_at`-Logik, bis diese deprecated wird).
- UI und API anpassen:
  - Billing-Ansicht:
    - Rechnungsliste bleibt bestehen.
    - Zusaetzlicher Zahlungsstatus je Mitglied: `unter Schwelle vorgetragen`,
      `fuer Einzug vorgemerkt`, `eingezogen`, `Ruecklastschrift`.
  - Neue Zahlungs-/Settlement-Ansicht unter Abrechnung oder Buchhaltung:
    - offene Salden pro Mitglied
    - ausloesbare Zahlungslaufe
    - Historie der Zahlungslaufe
    - Detailansicht welche Monatsrechnungen in welchem Lauf gesammelt wurden
  - Mitgliederansicht/Portal:
    - Monatsrechnung sichtbar
    - Hinweis ob Betrag nur vorgemerkt oder bereits in einem Zahlungslauf enthalten
- Buchhaltung sauber halten:
  - bestehender Accounting-Export bleibt rechnungsbasiert.
  - eigener Export fuer Zahlungsbewegungen, damit Bank-/Zahlungsausgleich abbildbar ist.
  - DATEV/XLSX muss zwischen Belegbuchung und Zahlungsausgleich unterscheiden koennen.
- Bestehende Sonderfaelle:
  - Storno eines Abrechnungslaufs darf nicht nur Rechnungen canceln, sondern muss offene
    Ledger-Eintraege reversieren. **Constraint**: Storno ist blockiert, wenn der zugehoerige
    Payment Run bereits den Status `submitted` oder `booked` hat — die SEPA-Bank-Transaktion
    ist zu diesem Zeitpunkt unwiderruflich raus. In diesem Fall muss eine manuelle
    Gegenbuchung (`manual_adjustment`) verwendet werden.
  - Credit Notes, Ruecklastschriften und manuelle Statusaenderungen muessen auf Ledger gemappt
    werden.
  - Such-, Statistik- und Report-Queries duerfen Zahlung nicht aus `invoices.status` ableiten.

## SEPA-Mandate

Aktuell hat die DB nur EEG-seitige SEPA-Felder (`iban`, `bic`, `sepa_creditor_id` auf `eegs`,
migration 010). Fuer `pain.008`-Lastschriften werden pro Mitglied benoetigt:

- IBAN und BIC des Mitglieds
- Mandatsreferenznummer (eindeutig je Glaeubiger-ID)
- Datum der Mandatsunterzeichnung (`signed_at`)
- Mandatsart (`CORE` / `B2B`)

**Neue Tabelle `sepa_mandates`** (neue Migration):

```sql
CREATE TABLE sepa_mandates (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  eeg_id        UUID NOT NULL REFERENCES eegs(id),
  member_id     UUID NOT NULL REFERENCES members(id),
  iban          TEXT NOT NULL,
  bic           TEXT,
  reference     TEXT NOT NULL,          -- MandatsreferenzNr
  signed_at     DATE NOT NULL,
  mandate_type  TEXT NOT NULL DEFAULT 'CORE',  -- CORE | B2B
  revoked_at    DATE,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (eeg_id, reference)
);
```

Mitglieder ohne aktives Mandat werden bei `CreatePaymentRunPreview` fuer Einzuege
uebersprungen (Warnung im Preview-Response). Auszahlungen (`pain.001`) benoetigen nur IBAN/BIC
des Mitglieds — diese koennen entweder aus dem Mandat oder separat gespeichert werden.

## Stichtag-Migration bestehender EEGs

Fuer v1 keine vollstaendige retroaktive Migration; stattdessen:

1. Admin waehlt Stichtag je EEG (z. B. Beginn des naechsten Abrechnungsmonats).
2. Alle Rechnungen mit `status = 'paid'` vor dem Stichtag erhalten einen einmaligen
   `opening_balance`-Ledger-Eintrag mit Netto-Saldo 0 (bereits ausgeglichen).
3. Alle Rechnungen mit `status = 'finalized'` vor dem Stichtag, die noch offen sind,
   erhalten einen `invoice_posted`-Eintrag mit dem offenen Betrag.
4. Ab Stichtag laeuft alles ueber das neue Ledger.

Dieser Prozess laeuft als einmalige Admin-Aktion (API-Endpunkt oder CLI-Skript), nicht
automatisch beim Deployment.

## Parallelitaet und Datenkonsistenz

- `CreatePaymentRunPreview` und `FinalizePaymentRun` muessen das `member_payment_account`
  mit `SELECT FOR UPDATE` sperren, um Race Conditions zu vermeiden (kein Mitglied in zwei
  parallele Payment Runs).
- `PostInvoiceToLedger` ist idempotent (UNIQUE-Constraint + ON CONFLICT DO NOTHING).
- Alle Ledger-Schreiboperationen laufen in einer Transaktion zusammen mit dem ausloesenden
  Ereignis (z. B. Billing-Run-Finalisierung).

## Implementierungsreihenfolge

1. Datenmodell und Migrationen
   - `sepa_mandates`, `member_payment_accounts`, `payment_ledger_entries`, `payment_runs`,
     `payment_run_items` anlegen.
   - EEG-Settings fuer Mindestbetraege ergaenzen (inkl. `payment_holdback_max_age_months`
     als NULL-Spalte).
   - UNIQUE-Constraints und Foreign Keys definieren.
   - Ledger als append-only modellieren; Korrekturen ueber Gegenbuchungen.
2. Backend-Domain und Repository
   - Domain-Typen und Repositories fuer Ledger/Payment Runs.
   - Services:
     - `PostInvoiceToLedger` (idempotent)
     - `ComputeMemberOpenBalance`
     - `CreatePaymentRunPreview` (inkl. PROSUMER-Richtungslogik, Mandat-Check)
     - `FinalizePaymentRun` (SELECT FOR UPDATE)
     - `BookPaymentResult`
     - `RecordReturn`
   - Billing-Finalisierung erweitern: beim finalen Beleg automatisch Ledger-Eintrag erzeugen.
3. SEPA-Integration
   - `SEPAHandler` auf Payment-Run-Basis umbauen.
   - `pain.008`/`pain.001` weiterhin aus vorhandenen Generatoren, aber Input aus
     `payment_run_items` + `sepa_mandates`.
   - CAMT.054 und Ruecklastschrift-Import an Payment Runs/Ledger koppeln
     (bestehende `sepa_return_at`-Felder parallel befuellen bis deprecated).
4. Stichtag-Migration
   - Admin-API oder CLI-Skript fuer einmalige Opening-Balance-Erstellung je EEG.
5. UI und API
   - Neue Endpunkte fuer offene Salden, Payment-Run-Vorschau, -Erzeugung, -Liste, -Detail.
   - Mandat-CRUD in Mitgliedsverwaltung.
   - Billing-Seite um Zahlungsstatus/Vortrag ergaenzen.
   - Neue Verwaltungsoberflaeche fuer Zahlungslaufe und Mindestbetragslogik.
   - Portal-Anzeige um Saldo-/Vortragsinfo ergaenzen.
   - E-Mail-Benachrichtigung beim Ausloesen eines Payment Runs (analog zu Billing-Run-Mails).
6. Buchhaltung und Reports
   - Bestehende Exports rechnungsbasiert belassen, aber offen dokumentieren.
   - Neuen Zahlungsausgleichs-Export ergaenzen.
   - Reports anpassen: offene Forderungen/Guthaben aus Ledger, nicht aus `invoices.status`.

## Oeffentliche APIs, Interfaces und Typen

- Neue Backend-Typen:
  - `MemberPaymentAccount`
  - `PaymentLedgerEntry` (mit `EntryType`-Enum fuer alle zulaessigen Typen)
  - `SEPAMandate`
  - `PaymentRun`
  - `PaymentRunItem`
- Neue oder geaenderte API-Endpunkte:
  - `GET  /eegs/{eegID}/payments/accounts`
  - `GET  /eegs/{eegID}/payments/accounts/{memberID}`
  - `POST /eegs/{eegID}/payments/runs/preview`
  - `POST /eegs/{eegID}/payments/runs`
  - `GET  /eegs/{eegID}/payments/runs`
  - `GET  /eegs/{eegID}/payments/runs/{runID}`
  - `POST /eegs/{eegID}/payments/runs/{runID}/book`
  - `POST /eegs/{eegID}/payments/runs/{runID}/returns`
  - `GET  /eegs/{eegID}/members/{memberID}/mandates`
  - `POST /eegs/{eegID}/members/{memberID}/mandates`
  - `DELETE /eegs/{eegID}/members/{memberID}/mandates/{mandateID}`
  - `POST /eegs/{eegID}/payments/migrate` — Stichtag-Migration fuer bestehende EEG
- Bestehende Rechnungs-API semantisch anpassen:
  - `invoice.status` bleibt Dokumentstatus.
  - Ergaenzende Felder im Response: `settlement_status`, `open_balance_effect`,
    `payment_run_item_id` oder aggregiert `payment_state`.
- EEG-Settings-API um Mindestbetraege und Netting-Flag erweitern.

## Testplan

- Billing erzeugt Monatsrechnungen unveraendert.
- Finalisierung erzeugt genau einen `invoice_posted`-Ledger-Eintrag; zweiter Aufruf ist No-Op
  (Idempotenz).
- Mehrere Monatsrechnungen unter der Schwelle werden vorgetragen, kein Payment Run.
- Sobald die Summe die Schwelle erreicht, entsteht genau ein Payment Run Item mit dem
  aggregierten Betrag.
- Positive und negative Belege saldieren sich korrekt auf Mitgliedsebene.
- PROSUMER mit negativem Nettosaldo erzeugt `pain.001`-Item (Auszahlung), nicht `pain.008`.
- PROSUMER mit positivem Nettosaldo erzeugt `pain.008`-Item (Einzug), nicht `pain.001`.
- `pain.008`/`pain.001` enthalten nur Payment Run Items, nicht rohe Rechnungen.
- Mitglied ohne aktives SEPA-Mandat erscheint in Preview als Warnung, nicht im finalen Run.
- `FinalizePaymentRun` unter gleichzeitigen Anfragen (parallele Requests) bucht ein Mitglied
  nur einmal (SELECT FOR UPDATE Test).
- CAMT.054/Ruecklastschrift erzeugt `direct_debit_returned`-Ledger-Eintrag und oeffnet Saldo.
- Storno eines nicht-submitted Billing Runs reversiert Ledger-Eintraege.
- Storno eines bereits submitted Payment Runs ist blockiert (HTTP 409).
- Stichtag-Migration: `paid`-Rechnungen erzeugen Opening-Balance mit Saldo 0;
  offene `finalized`-Rechnungen erzeugen `invoice_posted` mit korrektem Betrag.
- Mitgliederportal zeigt Monatsrechnung plus Vortrag/Zahlungsstatus korrekt.
- Accounting-Export bleibt fuer Rechnungsbelege stabil; Zahlungs-Export bildet
  Zahlungsereignisse separat ab.

## Annahmen

- Monatsrechnungen sollen weiterhin regulaer erstellt und fuer Mitglieder sichtbar sein.
- Zahlungen und Auszahlungen werden je Mitglied netto ueber einen gemeinsamen Saldo gefuehrt.
- Buchhaltung bleibt rechnungsbasiert; Zahlungsereignisse werden separat gefuehrt.
- Fuer v1 keine rueckwirkende Migration alter Rechnungen; Start ab Stichtag mit
  Opening-Balance-Eintrag je Mitglied (einmalige Admin-Aktion).
- Fuer v1 wird die Schwelle auf EEG-Ebene konfiguriert, nicht individuell pro Mitglied.
- `payment_holdback_max_age_months` wird in v1 nicht ausgewertet (Spalte NULL = unbegrenzt),
  aber die Spalte wird von Anfang an angelegt um spaetere Migrationen zu vermeiden.
