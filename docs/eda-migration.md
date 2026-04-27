# EDA Produktivbetrieb — Migrationsanleitung

## Voraussetzungen besorgen (beim Netzbetreiber / edanet.at)

Du brauchst folgende Informationen von edanet.at bzw. deinem Netzbetreiber:

| Was | Woher | Beispiel |
|-----|-------|---------|
| **Marktpartner-ID** (deine EEG) | edanet.at Registrierung | `AT9983000000000` |
| **Netzbetreiber-ID** (dein Netz) | Netzbetreiber (z.B. EVN, Wiener Netze) | `AT9999999999999` |
| **IMAP-Zugangsdaten** | edanet.at Mailbox-Zugang | Host, User, Passwort |
| **SMTP-Zugangsdaten** | edanet.at Mailbox-Zugang | Host, User, Passwort, Absender-E-Mail |
| **EDA-Übergangsdatum** | Datum, ab dem EDA aktiv ist | z.B. `2026-04-01` |

---

## Schritt 1 — Daten sichern (Backup erstellen)

**Jetzt, bevor du irgendetwas änderst:**

1. Im Browser: **Einstellungen** der Energiegemeinschaft öffnen
2. Ganz unten: **"Backup herunterladen"** klicken
3. Die `.json`-Datei sicher ablegen (z.B. `backup_wnw_vor_migration.json`)

Alternativ direktes DB-Backup:
```bash
cd /mnt/HC_Volume_103451728/eegabrechnung
docker compose exec eegabrechnung-postgres pg_dump \
  -U eegabrechnung eegabrechnung \
  > backup_$(date +%Y%m%d_%H%M%S).sql
```

---

## Schritt 2 — `.env` befüllen

Öffne `/mnt/HC_Volume_103451728/eegabrechnung/.env` und trage deine echten Werte ein:

```bash
# ── Sicherheit (wenn noch nicht geändert — JETZT ändern!) ─────────────────────
JWT_SECRET=<openssl rand -base64 32>
NEXTAUTH_SECRET=<openssl rand -base64 32>

# ── Datenbank (Passwort ändern empfohlen) ─────────────────────────────────────
POSTGRES_PASSWORD=<starkes-passwort>

# ── E-Mail für Rechnungsversand ────────────────────────────────────────────────
SMTP_HOST=<dein-smtp-server:587>
SMTP_FROM=<rechnungen@deine-domain.at>
SMTP_USER=<smtp-benutzer>
SMTP_PASSWORD=<smtp-passwort>
INVOICE_SEND_EMAIL=true          # auf true setzen wenn Rechnungen per Mail

# ── Web ────────────────────────────────────────────────────────────────────────
WEB_BASE_URL=https://deine-domain.at   # deine echte URL

# ── EDA Worker ────────────────────────────────────────────────────────────────
EDA_TRANSPORT=MAIL

# IMAP (vom edanet.at Mailbox-Zugang)
EDA_IMAP_HOST=<imap.edanet.at:993>     # ggf. anpassen
EDA_IMAP_USER=<deine-edanet-email>
EDA_IMAP_PASSWORD=<imap-passwort>

# SMTP für MaKo-Nachrichten (vom edanet.at Mailbox-Zugang)
EDA_SMTP_HOST=<smtp.edanet.at:587>     # ggf. anpassen
EDA_SMTP_USER=<deine-edanet-email>
EDA_SMTP_PASSWORD=<smtp-passwort>
EDA_SMTP_FROM=<deine-edanet-email>     # muss mit der registrierten EDA-Adresse übereinstimmen

EDA_POLL_INTERVAL=60s
```

Neue Secrets generieren:
```bash
openssl rand -base64 32   # für JWT_SECRET
openssl rand -base64 32   # für NEXTAUTH_SECRET
```

---

## Schritt 3 — EEG in der App konfigurieren

Im Browser unter **Einstellungen → "Energiegemeinschaft Wiener Neustadt West"**:

1. **Marktpartner-ID** eintragen (das bist du als EEG)
2. **Netzbetreiber-ID** eintragen (dein zuständiger Netzbetreiber)
3. **EDA-Übergangsdatum** setzen — das ist der Tag, ab dem der EDA-Worker Messdaten importieren soll. Blöcke vor diesem Datum werden ignoriert (Schutz vor doppeltem Import von bereits manuell eingetragenen Daten)
4. Speichern

In der Datenbank entspricht das:
```
eda_marktpartner_id   → deine Marktpartner-ID
eda_netzbetreiber_id  → ID des Netzbetreibers
eda_transition_date   → z.B. 2026-04-01
```

---

## Schritt 4 — Gemeinschaft-ID prüfen

Die `gemeinschaft_id` der EEG muss mit der offiziellen ID übereinstimmen, die edanet.at vergeben hat. Aktuell ist `RC105970` eingetragen.

Prüfe: Ist das deine offizielle EDA-Gemeinschafts-ID? Wenn nicht, in den Einstellungen korrigieren.

---

## Schritt 5 — Stack neu starten mit EDA Worker

```bash
cd /mnt/HC_Volume_103451728/eegabrechnung

# Stack komplett neu starten (lädt neue .env)
docker compose down
docker compose up -d

# EDA Worker starten (im Produktivbetrieb dauerhaft aktivieren)
docker compose --profile eda up -d
```

**Wichtig:** Wenn du `JWT_SECRET` oder `POSTGRES_PASSWORD` geändert hast, musst du dich nach dem Neustart neu einloggen.

---

## Schritt 6 — EDA Worker verifizieren

```bash
# Worker-Logs ansehen
docker compose logs -f eegabrechnung-eda-worker

# Worker-Status im Browser prüfen
# → EEG → "EDA" Tab → Worker-Status sollte "MAIL" und letzten Poll-Zeitstempel zeigen
```

Erwartete Log-Ausgabe beim Start:
```
EDA worker starting  poll_interval=60s
IMAP poll complete   messages_received=0
```

---

## Schritt 7 — Erste EDA-Anmeldung senden

Sobald der Worker läuft, kannst du für jeden Zählpunkt eine **Anmeldung** auslösen:

1. Im Browser: Mitglied öffnen → Zählpunkt auswählen → **"EDA Anmeldung"**
2. Gültig-ab-Datum eintragen
3. Absenden → Status wird `pending` → nach nächstem Poll `sent`

Der Worker sendet die MaKo-XML per SMTP an den Netzbetreiber. Wenn er antwortet (CPDocument), ändert sich der Status automatisch auf `first_confirmed` → `confirmed`.

---

## Datensicherheit — Was bleibt erhalten?

| Was | Status |
|-----|--------|
| Alle Mitglieder und Zählpunkte | ✅ bleiben unverändert |
| Alle Messdaten (energy_readings) | ✅ bleiben unverändert |
| Alle Rechnungen und Billing Runs | ✅ bleiben unverändert |
| Einstellungen der EEG | ✅ bleiben unverändert |
| Passwörter der Benutzer | ✅ bleiben unverändert |
| EDA-Übergangsdatum | nur neue Blöcke werden importiert — kein Überschreiben alter Daten |

Die Datenbank läuft in einem Docker Volume (`eegabrechnung_postgres_data`) — solange du `docker compose down` (ohne `-v`) verwendest, bleibt alles erhalten.

---

## Wichtigste Sicherheitsregel

**Niemals** `docker compose down -v` ausführen — das `-v` löscht alle Volumes inkl. der Datenbank. Ohne `-v` ist alles safe.
