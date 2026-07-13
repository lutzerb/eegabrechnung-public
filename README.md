# eegabrechnung

Selbst gehostete Abrechnungsplattform für österreichische Energiegemeinschaften:
Mitglieder, Zählpunkte, Energiedaten, Rechnungen, SEPA und EDA-Marktkommunikation in einem System.

Das Projekt ist für Betreiber gedacht, die ihre EEG nicht mit Excel, Mail-Threads und mehreren Insellösungen führen wollen, aber auch kein überdimensioniertes ERP einführen möchten.

## Motivation

Energiegemeinschaften sind fachlich und gesellschaftlich eine sehr spannende Sache. In der Praxis ist die laufende Abrechnung aber oft überraschend aufwändig: Mitglieder, Zählpunkte, Energiedaten, Rechnungen, Gutschriften, SEPA und Marktkommunikation müssen sauber zusammenpassen.

Es gibt bereits kommerzielle Anbieter am Markt. Für größere Setups kann das sinnvoll sein. Für kleine Energiegemeinschaften werden die laufenden Abrechnungskosten aber oft unverhältnismäßig hoch und machen das Modell damit operativ unattraktiver, als es eigentlich sein müsste.

Auch bestehende Open-Source- oder Source-Available-Ansätze haben mich persönlich nicht überzeugt. `eegfaktura` war für mich kein guter Ausgangspunkt, weil es nach meinem Eindruck nicht wirklich `out of the box` lief, wenn man das Repository einfach von GitHub klont, und weil mir dort einige Komfortfunktionen gefehlt haben, die ich in `eegabrechnung` für den echten Betrieb wichtig finde.

Genau aus dieser Motivation ist `eegabrechnung` entstanden: ein Werkzeug, das für reale österreichische Energiegemeinschaften praktikabel ist, selbst gehostet werden kann und gerade auch für kleinere Betreiber eine vernünftige Alternative sein soll.

## Für wen das gedacht ist

- kleine bis mittlere Energiegemeinschaften in Österreich
- Vereine, Hausgemeinschaften und Initiativen mit wenig interner IT-Kapazität
- Dienstleister, die EEGs technisch oder operativ betreuen
- Betreiber, die ein selbst hostbares System mit nachvollziehbarem Datenmodell wollen

## Was das Tool heute abdeckt

- Mitglieder- und Zählpunktverwaltung
- XLSX-Import und automatische EDA-Datenübernahme
- Tarifpläne und periodische Abrechnung
- Individuelle Tarif-Overrides pro Mitglied (eigener Arbeitspreis, reduzierte oder erlassene Fixkosten)
- Zählpunktsgebühr (Gebühr je aktivem Zählpunkt eines Mitglieds)
- PDF-Rechnungen, Gutschriften und SEPA-Dateien
- automatischer Zahlungsabgleich per CAMT.053-Kontoauszug (markiert Rechnungen als bezahlt)
- DATEV/XLSX-Buchhaltungsexport
- Einnahmen-Ausgaben-Buchhaltung für Vereine (wirtschaftlicher Geschäftsbetrieb, bis € 700.000 Umsatz)
- Onboarding-Portal für neue Mitglieder (inkl. manuell erstellbarem Anmeldelink durch Admins)
- passwortloses Mitgliederportal mit Selbstbedienung (IBAN-Änderung mit neuem SEPA-Mandat, E-Mail-Änderung mit Bestätigungslink)
- EDA-Prozesse für Anmeldung, Widerruf, Teilnahmefaktor und Datenanforderung
- automatische Plausibilitätswarnungen (vertauschte Energierichtung eines Zählpunkts, Ungleichgewicht zwischen Bezug und Einspeisung)
- automatische Mitglieder-Benachrichtigung bei fehlendem Smartmeter (§16e ElWOG)
- E-Mail-Kampagnen an Mitglieder mit Platzhaltersubstitution und Anhängen
- E-Mail-Protokoll: jeder Versand (Rechnungen, Kampagnen, Onboarding, EDA) wird nachvollziehbar geloggt
- Zählpunkt-Notizen und SEPA-Mandat-PDF
- Gemeinschaftstypen EEG, GEA und Bürgerenergiegemeinschaft (BEG); bei BEG mit mehreren Netzbetreibern und zählpunktweisem EDA-Routing

## Screenshots

Die folgenden Screenshots stammen aus der Demo-EEG im System.

### Dashboard

![Demo-EEG Dashboard](docs/screenshots/eeg-dashboard.png)

### Mitglieder und Zählpunkte

![Mitgliederliste](docs/screenshots/members-list.png)

### Abrechnung

![Billing](docs/screenshots/billing.png)

### Auswertungen

![Auswertungen Monat](docs/screenshots/reports-month.png)

### EDA-Prozesse

![EDA Prozesse](docs/screenshots/eda.png)

### Buchhaltung

![Accounting](docs/screenshots/accounting.png)

### Öffentliches Onboarding

![Onboarding](docs/screenshots/onboarding-public.png)

## Warum das für EEGs interessant ist

Eine EEG hat typischerweise dieselben operativen Reibungen:

- Mitgliedsdaten und Zählpunkte müssen konsistent bleiben
- Netzbetreiber-Daten kommen unregelmäßig und in verschiedenen Formaten
- Rechnungen, Gutschriften und Lastschriften müssen nachvollziehbar sein
- Marktkommunikation über EDA ist fachlich nötig, aber operativ mühsam
- für kleine Betreiber sind klassische ERP- oder Utility-Systeme oft zu schwergewichtig

`eegabrechnung` adressiert genau diesen Bereich: schlanke, selbst hostbare EEG-Verwaltung mit Fokus auf Österreich und auf reale operative Workflows.

## Funktionsüberblick

### Mitglieder und Zählpunkte

- Stammdaten inkl. IBAN, UID, Beitritts- und Austrittsdatum
- automatische Mitgliedsnummern
- Lebenszyklus-Status für Mitglieder und Zählpunkte
- vollständige Anmelde-/Abmelde-Historie je Zählpunkt
- Zählpunkt-Wiederverwendung bei Mieterwechsel (historische Messdaten bleiben erhalten)
- Notizfeld pro Zählpunkt
- Mehrfachteilnahme nach EAG
- automatischer Widerrufs-Workflow (CM_REV_SP) für austretende Mitglieder

### Energiedaten

- XLSX-Import mit Vorschau und Konflikterkennung
- automatische Übernahme eingehender EDA-Messdaten (inkl. korrekter Behandlung von G.01T bei Mehrfachteilnahme)
- historische Datenanforderung beim Netzbetreiber (auch als Mehrfachabfrage über ausgewählte Mitglieder/Zählpunkte)
- Datenabdeckung und Lücken-Erkennung
- automatische Warnung, wenn eingehende Messdaten nicht zur hinterlegten Energierichtung eines Zählpunkts passen (vertauschte Zählpunkte)

### Abrechnung und Rechnungen

- Tarifpläne mit verschiedenen Granularitäten
- echtes Time-of-Use-Billing: jede 15-Minuten-Messung wird mit dem zu ihrem Zeitpunkt gültigen Tarif bepreist
- individuelle Tarif-Overrides pro Mitglied (eigener Arbeitspreis, reduzierte/erlassene Fixgebühren) unabhängig vom EEG-Standardtarif
- Zählpunktsgebühr — Fixbetrag je aktivem Zählpunkt eines Mitglieds, immer als eigene Rechnungszeile
- Fixgebühren wahlweise pro angefangenem Kalendermonat oder einmal pro Abrechnungslauf
- Abrechnungsläufe mit Zeitraumsschutz
- konfigurierbare Warnung bei Ungleichgewicht zwischen Bezugs- und Einspeisungssumme (Hinweis auf fehlende Messdaten, blockiert die Abrechnung nicht)
- konfigurierbarer Zahlungshinweis auf Rechnungen (SEPA-Lastschrift, Überweisung oder kein Hinweis)
- Draft-, Finalisierungs- und Storno-Workflow
- PDF-Rechnungen und Gutschriften
- automatischer Mailversand mit wiederverwendeter SMTP-Verbindung beim Massenversand
- SEPA pain.001 (Überweisung) und pain.008 (Lastschrift)
- SEPA-Mandat-PDF mit Voranmeldungsfrist (konfigurierbar, Standard 14 Tage)
- Zahlungsabgleich per CAMT.053-Kontoauszug — markiert passende Rechnungen automatisch als bezahlt

### Buchhaltung und Auswertung

- DATEV-Export
- XLSX-Buchungsjournal
- Energieberichte pro EEG und pro Mitglied
- OeMAG-Marktpreisintegration

### Einnahmen-Ausgaben-Buchhaltung

EEGs als Verein können ihre steuerliche Buchhaltung direkt im System führen:

- Kontenplan mit konfigurierbaren Konten (Einnahmen, Ausgaben, Sonstig)
- Buchungsjournal mit Belegdatum, Zahlungsdatum und Beleg-Upload
- Saldenliste und Jahresabschluss (Überschussrechnung nach §4 Abs. 3 EStG)
- Umsatzsteuervoranmeldung (UVA) mit Kennzahlen und FinanzOnline-XML-Export
- Jahreserklärungen U1 und K1 als Datenbasis
- automatischer Import von EEG-Ausgangsrechnungen als Buchungen
- Bankimport (MT940/CAMT.053) mit Buchungszuordnung
- Unterstützung für Reverse Charge (§2 Z 2 UStBBKV und §22 UStG)
- Audit-Trail für alle Buchungsänderungen (Anlegen, Ändern, Löschen) nach BAO §131; Soft-Delete mit Pflichtangabe eines Löschgrunds; EEG-weites Änderungsprotokoll

### Onboarding und Portal

- öffentliches Beitrittsformular mit E-Mail-Verifizierung und SEPA-Mandat-Erfassung
- alternativ: manuell durch den Admin erstellter Anmeldelink, ohne dass sich das Mitglied selbst über das öffentliche Formular meldet
- Admin-Freigabe und automatische Anlage von Mitgliedern und Zählpunkten
- automatische Benachrichtigung, wenn der Netzbetreiber laut Rückmeldung noch keinen fernauslesbaren Zähler installiert hat (§16e ElWOG, 2-Monats-Frist)
- Mitgliederportal mit Magic Link, Rechnungen und Energieübersicht (wahlweise eigene Quote oder volle EEG-Energie)
- Selbstbedienung im Portal: IBAN-Änderung mit neu signiertem SEPA-Mandat (alte Mandate bleiben als Historie erhalten) und E-Mail-Änderung mit Bestätigungslink an die neue Adresse
- Hinweis auf das Mitgliederportal in Rechnungs- und Willkommensmails
- E-Mail-Kampagnen an Mitglieder mit Zielgruppenauswahl, Platzhaltern und Anhängen
- E-Mail-Protokoll pro EEG: Status, Empfänger und Fehlermeldung jedes Versands nachvollziehbar

### EDA-Marktkommunikation

Im Zentrum stehen derzeit Online-Anmeldung, Teilnahmefaktoränderung, Zählpunktlistenabfrage, Messdatenanforderung und Zustimmungswiderruf. Die Prozessnamen folgen der offiziellen ebutilities-Prozessliste; einzig `EC_PRTFACT_CHG` wird intern als Kurzform des offiziellen Namens `EC_PRTFACT_CHANGE` verwendet.

### Ausgehende EDA-Prozesse

| Prozess | Tatsächlich erzeugte Nachricht | Was der Prozess fachlich tut |
|---|---|---|
| `EC_REQ_ONL` | `CMRequest 01.30` mit `MessageCode=ANFORDERUNG_ECON` | Startet die Online-Anmeldung eines Zählpunkts zur Energiegemeinschaft. |
| `EC_PRTFACT_CHG` (offiziell `EC_PRTFACT_CHANGE`) | `ECMPList 01.10` mit `MessageCode=ANFORDERUNG_CPF` | Ändert den Teilnahmefaktor eines bereits zugeordneten Zählpunkts. |
| `CR_REQ_PT` | `CPRequest 01.12` mit `MessageCode=ANFORDERUNG_PT` | Fordert historische Zählpunktdaten bzw. Messwerte für einen Zeitraum an. |
| `EC_PODLIST` | `CPRequest 01.12` mit `MessageCode=ANFORDERUNG_ECP` | Fordert die aktuelle Zählpunktliste der Energiegemeinschaft beim Netzbetreiber an. |
| `CM_REV_SP` | `CMRevoke 01.10` mit `MessageCode=AUFHEBUNG_CCMS` | Widerruft eine zuvor erteilte Zustimmung. Dieser Prozess wird insbesondere beim Austritt eines Mitglieds verwendet und setzt eine gespeicherte `ConsentId` voraus. |

Im SMTP-Betreff werden dazu aktuell diese edanet-`Prozess-Id`-Mappings verwendet: `EC_PRTFACT_CHANGE_01.00`, `EC_REQ_ONL_02.30`, `CR_REQ_PT_04.10`, `EC_PODLIST_01.00` und `CM_REV_SP_01.00`.

### Eingehende EDA-Nachrichten

| Eingehende Nachricht | Was die Verarbeitung im System tut |
|---|---|
| `CMNotification` | Verarbeitet Rückmeldungen im Anmelde- und Widerrufspfad: `ANTWORT_ECON` als Zwischenrückmeldung, `ZUSTIMMUNG_ECON` als Zustimmung, `ABLEHNUNG_ECON` als Ablehnung sowie `AUFHEBUNG_CCMS_OK` bzw. `AUFHEBUNG_CCMS_ABGEL` für den Widerruf. |
| `CPDocument` | Verarbeitet Status- und Bestätigungsnachrichten des Netzbetreibers wie `ERSTE_ANM`, `FINALE_ANM`, `ABSCHLUSS_ECON` und Ablehnungen und setzt den zugehörigen Prozessstatus entsprechend. |
| `ECMPList` | Verarbeitet `SENDEN_ECP` als Antwort bzw. Push der aktuellen Zählpunktliste und backfilled dabei fehlende `ConsentId`s. `ABSCHLUSS_ECON` bestätigt final eine Anmeldung und aktualisiert u. a. `registriert_seit` und `ConsentId`. |
| `DATEN_CRMSG` | Importiert eingehende Messdaten in das System und schließt zugehörige Datenanforderungen ab. |
| `CM_REV_CUS` (`AUFHEBUNG_CCMS`) | Kunde widerruft seine Zustimmung über den Netzbetreiber. Markiert den ursprünglichen Anmeldeprozess als `completed`, setzt `abgemeldet_am` auf dem Zählpunkt aus dem `ConsentEnd`-Datum. |
| `CM_REV_IMP` (`AUFHEBUNG_CCMS_IMP`) | Netzbetreiber hebt die Anmeldung wegen Unmöglichkeit auf (z. B. Zählpunkt-Übertragung). Gleiche Effekte wie `CM_REV_CUS`, zusätzlich wird eine Operator-E-Mail zur Information versendet. |
| unaufgeforderte NB-Widerrufe (z. B. `AUFHEBUNG_CCMI`) | Auch Widerrufe ohne eigenen vorausgehenden `CM_REV_SP`-Prozess (etwa nach einem Lieferantenwechsel am Zählpunkt) werden automatisch verarbeitet: Zuordnung über Zählpunkt und `ConsentId`, Abmeldedatum wird gesetzt, bei unfreiwilligen Varianten wird der Betreiber per E-Mail informiert. |
| `EDASendError` | Verarbeitet Gateway- oder Versandfehler und markiert den betroffenen Prozess bzw. die Nachricht als Fehlerfall. |

Fachlich wichtig: Die Abmeldung erfolgt ausschließlich über den Zustimmungswiderruf `CM_REV_SP` beziehungsweise den Mitglieder-Austritt, der daraus automatisch Widerrufe für alle aktiven Zählpunkte erzeugt. Eine gespeicherte `ConsentId` (aus der Anmeldebestätigung) ist dafür Voraussetzung.

Hinweis: Eine Ponton-XP-Anbindung ist im Code angelegt, aber derzeit nicht als produktionsreif dokumentiert.

## Architektur in Kurzform

- `web/`: Next.js 16 Frontend
- `api/`: Go REST API
- `api/cmd/worker/`: separater EDA-Worker
- `docker-compose.yaml`: Postgres, API, Web, optionale Profile für EDA, Caddy und Tests

Der Standardweg ist Selbsthosting via Docker Compose.

## Welche Infrastruktur man dafür braucht

Für den Betrieb braucht es keine große Spezialinfrastruktur. In vielen Fällen reicht bereits:

- ein kleiner VPS
- oder ein eigener Server bzw. Mini-PC zuhause
- Docker und Docker Compose
- genügend Speicherplatz für Datenbank, Dokumente und Rechnungs-PDFs

Für kleine bis mittlere Energiegemeinschaften ist der Ressourcenbedarf grundsätzlich überschaubar. Wichtiger als rohe Rechenleistung sind ein stabiler Betrieb und eine saubere Datensicherung.

Besonders wichtig ist:

- regelmäßige Backups der PostgreSQL-Datenbank
- Sicherung der erzeugten Dokumente und Rechnungs-PDFs
- Aufbewahrung der `.env`-Konfiguration und Secrets an einem sicheren Ort
- idealerweise ein getesteter Restore-Prozess, nicht nur ein Backup

Wer das System zuhause hostet, sollte zusätzlich auf Stromausfallsicherheit, Internetverfügbarkeit und externe Backups achten. Für viele produktive Setups ist deshalb ein kleiner VPS oder dedizierter Server mit sauberer Backup-Strategie meist der robustere Weg.

## Schnellstart

### Voraussetzungen

- Docker (≥ 24)
- Docker Compose v2 (bei Docker Desktop enthalten; bei Linux-Server: `docker compose version` prüfen)

### 1. Repository klonen

```bash
git clone https://github.com/lutzerb/eegabrechnung-public
cd eegabrechnung-public
```

### 2. Konfigurationsdatei anlegen

```bash
cp .env.example .env
```

### 3. Secrets generieren

Alle vier Werte müssen in `.env` eingetragen werden. Die folgenden Befehle erzeugen kryptografisch sichere Zufallswerte:

```bash
echo "JWT_SECRET=$(openssl rand -base64 32)"
echo "NEXTAUTH_SECRET=$(openssl rand -base64 32)"
echo "CREDENTIAL_ENCRYPTION_KEY=$(openssl rand -base64 32)"
echo "POSTGRES_PASSWORD=$(openssl rand -base64 24)"
```

Die Ausgaben direkt in die entsprechenden Zeilen in `.env` eintragen.

### 4. Basis-URL setzen

In `.env` die öffentliche Adresse des Frontends eintragen — wird für Magic-Links im Onboarding und Mitgliederportal benötigt:

```
WEB_BASE_URL=https://meine-eeg.at
```

Für einen lokalen Test ohne öffentliche Domain ist `http://localhost:3001` möglich.

### 5. Starten

```bash
docker compose up --build -d
```

Beim ersten Start werden alle Datenbankmigrationen automatisch eingespielt. Das dauert erfahrungsgemäß 30–60 Sekunden.

### Standard-URLs lokal

| Service | URL |
|---|---|
| Frontend | http://localhost:3001 |
| API | http://localhost:8101 |
| Postgres | localhost:26433 |

### Erster Login

Der Standard-Admin aus den Migrationen lautet:

- E-Mail: `admin@eeg.at`
- Passwort: `admin`

**Das Passwort sollte nach dem ersten Login sofort geändert werden**, bevor das System öffentlich erreichbar ist.

### Nächste Schritte nach dem Login

1. Unter **Einstellungen** die EEG-Stammdaten (Name, Adresse, UID) hinterlegen
2. Tarif und Abrechnungsparameter konfigurieren
3. Mitglieder und Zählpunkte anlegen oder importieren
4. Optional: SMTP-Zugangsdaten für Rechnungsversand in den EEG-Einstellungen hinterlegen

Die ausführliche Ersteinrichtung ist in [`docs/02-erste-schritte.md`](docs/02-erste-schritte.md) beschrieben.

## Update auf eine neue Version

Wenn auf GitHub eine neue Version verfügbar ist, reichen folgende Schritte:

```bash
# 1. Neue Version holen
git pull

# 2. Images neu bauen und Container neu starten
docker compose up --build -d
```

Datenbankmigrationen werden beim Start automatisch eingespielt — kein manueller Eingriff nötig.

**Hinweis:** Wer `docker compose --profile eda up -d` nutzt, muss beim Update denselben Profilaufruf verwenden, damit der EDA-Worker ebenfalls neu gebaut und gestartet wird:

```bash
docker compose --profile eda up --build -d
```

### Was dabei passiert

1. `git pull` holt die aktuellen Commits inkl. aller Code- und Konfigurationsänderungen.
2. `--build` baut nur die Container-Images neu, die sich geändert haben.
3. Die Container werden rolling neu gestartet; bestehende Daten (Postgres-Volume, Dokumente) bleiben unangetastet.
4. Neue Datenbankmigrationen werden beim nächsten API-Start automatisch angewendet.

### Vor dem Update empfohlen

```bash
# Aktuellen Stand sichern (optional, aber empfohlen)
docker exec eegabrechnung-eegabrechnung-postgres-1 \
  pg_dump -U eegabrechnung -Fc eegabrechnung > backup-vor-update.dump
```

Bei größeren Updates lohnt sich ein kurzer Blick in die Commit-History (`git log --oneline HEAD..origin/main` vor dem Pull), um zu sehen, was sich geändert hat.

## Optionale Compose-Profile

| Profil | Zweck |
|---|---|
| `eda` | produktiver EDA-Worker |
| `caddy` | Reverse Proxy mit TLS |
| `cloudflare` | Cloudflare Tunnel (kein offener Port nötig) |
| `test` | Integrations-Teststack mit Mailpit und FILE-Worker |
| `demo` | Demo-Reset-Job |
| `forecast` | ML-Prognose-Service für die 7-Tage-Prognose auf der Berichte-Seite (benötigt ≥ 200 Stundenwerte historische Messdaten) |

Beispiele:

```bash
docker compose --profile eda up -d
docker compose --profile caddy up -d
docker compose --profile cloudflare up -d
docker compose --profile test up -d
```

## Dokumentation

Ausführlichere Dokumentation liegt unter `docs/`, unter anderem:

- [Installation](docs/01-installation.md)
- [Erste Schritte](docs/02-erste-schritte.md)
- [Abrechnung](docs/07-abrechnung.md)
- [Buchhaltung](docs/09-buchhaltung.md)
- [SEPA](docs/10-sepa.md)
- [EDA](docs/11-eda.md)
- [Mitgliederportal](docs/13-mitgliederportal.md)

## Tests

Das Repo enthält:

- Go-Unit-Tests in `api/internal/...`
- einen Python-Integrationstest-Harness unter `test/`

Der Integrationstest-Harness läuft gegen den Docker-Stack und deckt Billing, SEPA, EDA, Onboarding, Import und Accounting ab.

## Roadmap

Alle drei Gemeinschaftstypen — Erneuerbare-Energie-Gemeinschaft (EEG), gemeinschaftliche Erzeugungsanlage (GEA) und Bürgerenergiegemeinschaft (BEG) — werden unterstützt, inklusive des BEG-Spezialfalls mehrerer Netzbetreiber mit zählpunktweisem EDA-Routing.

Themen, die als Nächstes spannend sind und bei denen ich mich auch über Beiträge freue:

- Ponton X/P-Anbindung
- P2P-Austausch laut ElWOG-Novelle
  - fachlich besonders spannend im Hinblick auf die Änderungen ab Oktober 2026

## Grenzen und ehrliche Hinweise

- Das System ist stark auf den österreichischen EEG-Kontext zugeschnitten.
- Das Verteilungsmodell (statisch/dynamisch) wird EEG-weit festgelegt und bei EDA-Prozessen an den Netzbetreiber gemeldet. Im realen Betrieb erprobt ist bisher nur das dynamische Modell.
- Die BEG-Unterstützung (mehrere Netzbetreiber, zählpunktweises EDA-Routing) ist implementiert, aber noch weniger praxiserprobt als der EEG-/GEA-Betrieb.
- EDA per Mail ist der primäre reale Betriebsweg.
- Ponton ist derzeit kein offiziell produktionsreifer Standardpfad.
- Wer das Tool öffentlich oder produktiv betreibt, sollte TLS, Backups, Monitoring und Secret-Management sauber selbst aufsetzen.

## Support

Das Projekt ist derzeit auf `best effort`-Basis gedacht:

- Issues für reproduzierbare Bugs
- Pull Requests für konkrete Verbesserungen
- keine garantierten Reaktionszeiten

## Lizenz

Dieses Repository ist **source-available** unter:

- GNU AGPL v3.0 only
- plus Commons Clause License Condition v1.0

Wichtig: Diese Kombination ist **nicht** OSI-open-source. Details stehen in [LICENSE](LICENSE) und [LICENSE-AGPL-3.0.txt](LICENSE-AGPL-3.0.txt).

### Praktisch gesprochen

Erlaubt ist insbesondere:

- Nutzung der Software durch Energiegemeinschaften jeglicher Rechtsform für den eigenen Betrieb
- Selbsthosting und interner Einsatz durch Vereine, Genossenschaften, GmbHs, Gemeinden oder sonstige EEG-Träger
- Anpassung des Codes für den eigenen Einsatz
- Weiterentwicklung und Beiträge zum Projekt im Rahmen der Lizenz

Nicht erlaubt ist insbesondere:

- ein SaaS-Angebot auf Basis dieses Codes, für das Dritte bezahlen
- die Organisation oder Durchführung von Abrechnung für Energiegemeinschaften als bezahlte Dienstleistung auf Basis dieser Software
- Hosting-, Consulting- oder Support-Angebote, deren wirtschaftlicher Wert im Wesentlichen aus dieser Software abgeleitet wird, ohne gesonderte Vereinbarung mit dem Rechteinhaber

Diese Beispiele sind nur eine alltagsnahe Zusammenfassung. Maßgeblich ist der Lizenztext in [LICENSE](LICENSE).
