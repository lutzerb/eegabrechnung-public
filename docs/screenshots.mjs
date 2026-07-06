/**
 * docs/screenshots.mjs — Playwright screenshot generator for eegabrechnung docs
 *
 * Usage:  ADMIN_EMAIL=admin@example.at ADMIN_PASSWORD=yourpassword node docs/screenshots.mjs
 * Prereq: App running at http://localhost:3001; Playwright installed in /tmp/pw_test/
 *
 * Writes PNG files to docs/screenshots/
 */

import { chromium } from "/tmp/pw_test/node_modules/playwright/index.mjs";
import path from "path";
import { fileURLToPath } from "url";
import fs from "fs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const SCREENSHOTS_DIR = path.join(__dirname, "screenshots");
const BASE_URL = process.env.BASE_URL || "http://localhost:3001";
const ADMIN_EMAIL = process.env.ADMIN_EMAIL || "admin@example.at";
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD || "";
const EEG_ID = process.env.EEG_ID || "5d0151e8-8714-4605-9f20-70ec5d5d5b46";

// IDs for detail pages (EA-Buchhaltung)
const EA_BUCHUNG_ID = "8c0abccd-1a6d-45a9-b250-fb503d2565c5";
const EA_KONTO_ID = "e7a7d22a-9dc3-496c-bcc0-c88b80fc0d83";

fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });

const browser = await chromium.launch({
  executablePath: "/usr/bin/chromium-browser",
  args: ["--no-sandbox", "--disable-setuid-sandbox"],
});

const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: "de-AT",
});

const page = await context.newPage();

/** Navigate and take a full-page screenshot. Warns if we end up on the login page. */
async function shot(filename, url, { waitFor, beforeShot } = {}) {
  console.log(`  → ${filename}`);
  await page.goto(`${BASE_URL}${url}`, { waitUntil: "networkidle" });
  if (waitFor) {
    await page.waitForSelector(waitFor, { timeout: 8000 }).catch(() => {});
  }
  if (beforeShot) await beforeShot(page);

  // Warn if we got redirected to the login page
  if (page.url().includes("signin") || page.url().includes("login")) {
    console.warn(`    ⚠ Redirected to login for ${filename} — session may have expired`);
  }

  await page.screenshot({
    path: path.join(SCREENSHOTS_DIR, filename),
    fullPage: true,
  });
}

// ── Login ──────────────────────────────────────────────────────────────────

console.log("Login page...");
await shot("login.png", "/auth/signin");

// Perform login
await page.fill('input[name="email"]', ADMIN_EMAIL);
await page.fill('input[name="password"]', ADMIN_PASSWORD);
await page.click('button[type="submit"]');

// Wait until we are no longer on the signin page
await page.waitForURL(
  (url) => !url.toString().includes("signin"),
  { timeout: 15000 }
);
// Ensure the page is fully loaded after redirect
await page.waitForLoadState("networkidle");

console.log(`  Logged in — now at: ${page.url()}`);

// ── EEG list ───────────────────────────────────────────────────────────────

console.log("EEG overview...");
await shot("eegs-list.png", "/eegs");

// ── Dashboard / EEG detail ─────────────────────────────────────────────────

console.log("EEG dashboard...");
await shot("eeg-dashboard.png", `/eegs/${EEG_ID}`);

// ── Members ────────────────────────────────────────────────────────────────

console.log("Members...");
await shot("members-list.png", `/eegs/${EEG_ID}/members`);

// ── Import ─────────────────────────────────────────────────────────────────

console.log("Energy import...");
await shot("import.png", `/eegs/${EEG_ID}/import`);

// ── Tariffs ────────────────────────────────────────────────────────────────

console.log("Tariff schedules...");
await shot("tariffs.png", `/eegs/${EEG_ID}/tariffs`);

// ── Billing ────────────────────────────────────────────────────────────────

console.log("Billing runs...");
await shot("billing.png", `/eegs/${EEG_ID}/billing`);

// ── Reports ────────────────────────────────────────────────────────────────

console.log("Reports...");
await shot("reports.png", `/eegs/${EEG_ID}/reports`);

// ── Accounting ─────────────────────────────────────────────────────────────

console.log("Accounting...");
await shot("accounting.png", `/eegs/${EEG_ID}/accounting`);

// ── EDA processes ──────────────────────────────────────────────────────────

console.log("EDA...");
await shot("eda.png", `/eegs/${EEG_ID}/eda`);

// ── EDA message log (click Messages tab) ───────────────────────────────────

console.log("EDA messages...");
await page.goto(`${BASE_URL}/eegs/${EEG_ID}/eda`, { waitUntil: "networkidle" });
const msgTab = page.locator("button, a").filter({ hasText: /Nachrichten|Messages|Protokoll/ }).first();
if (await msgTab.count()) {
  await msgTab.click();
  await page.waitForLoadState("networkidle");
}
await page.screenshot({
  path: path.join(SCREENSHOTS_DIR, "eda-messages.png"),
  fullPage: true,
});

// ── Onboarding admin ───────────────────────────────────────────────────────

console.log("Onboarding admin...");
await shot("onboarding-admin.png", `/eegs/${EEG_ID}/onboarding`);

// ── Mehrfachteilnahme ──────────────────────────────────────────────────────

console.log("Mehrfachteilnahme...");
await shot("participations.png", `/eegs/${EEG_ID}/participations`);

// ── Settings tabs ──────────────────────────────────────────────────────────

console.log("Settings — Allgemein...");
await shot("settings-allgemein.png", `/eegs/${EEG_ID}/settings?tab=allgemein`);

console.log("Settings — Rechnungen...");
await shot("settings-rechnungen.png", `/eegs/${EEG_ID}/settings?tab=rechnungen`);

console.log("Settings — SEPA...");
await shot("settings-sepa.png", `/eegs/${EEG_ID}/settings?tab=sepa`);

console.log("Settings — EDA...");
await shot("settings-eda.png", `/eegs/${EEG_ID}/settings?tab=eda`);

// ── Admin users ────────────────────────────────────────────────────────────

console.log("User admin...");
await shot("admin-users.png", "/admin/users");

// ── E/A-Buchhaltung ────────────────────────────────────────────────────────

console.log("EA — Dashboard...");
await shot("ea-dashboard.png", `/eegs/${EEG_ID}/ea`);

console.log("EA — Buchungen...");
await shot("ea-buchungen.png", `/eegs/${EEG_ID}/ea/buchungen`);

console.log("EA — Buchung Detail...");
await shot("ea-buchung-detail.png", `/eegs/${EEG_ID}/ea/buchungen/${EA_BUCHUNG_ID}`);

console.log("EA — Konten...");
await shot("ea-konten.png", `/eegs/${EEG_ID}/ea/konten`);

console.log("EA — Saldenliste...");
await shot("ea-saldenliste.png", `/eegs/${EEG_ID}/ea/saldenliste`);

console.log("EA — Kontenblatt...");
await shot("ea-kontenblatt.png", `/eegs/${EEG_ID}/ea/kontenblatt/${EA_KONTO_ID}`);

console.log("EA — Jahresabschluss...");
await shot("ea-jahresabschluss.png", `/eegs/${EEG_ID}/ea/jahresabschluss`);

console.log("EA — UVA...");
await shot("ea-uva.png", `/eegs/${EEG_ID}/ea/uva`);

console.log("EA — Erklärungen...");
await shot("ea-erklaerungen.png", `/eegs/${EEG_ID}/ea/erklaerungen`);

console.log("EA — Import...");
await shot("ea-import.png", `/eegs/${EEG_ID}/ea/import`);

console.log("EA — Bank...");
await shot("ea-bank.png", `/eegs/${EEG_ID}/ea/bank`);

console.log("EA — Changelog...");
await shot("ea-changelog.png", `/eegs/${EEG_ID}/ea/changelog`);

console.log("EA — Settings...");
await shot("ea-settings.png", `/eegs/${EEG_ID}/ea/settings`);

// ── Public onboarding form (no auth needed) ────────────────────────────────

console.log("Public onboarding form...");
// Open in a fresh context so it doesn't redirect to dashboard
const publicContext = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: "de-AT",
});
const publicPage = await publicContext.newPage();
await publicPage.goto(`${BASE_URL}/onboarding/${EEG_ID}`, { waitUntil: "networkidle" });
await publicPage.screenshot({
  path: path.join(SCREENSHOTS_DIR, "onboarding-public.png"),
  fullPage: true,
});
await publicContext.close();

// ── Member portal request-link page ────────────────────────────────────────

console.log("Member portal...");
const portalContext = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: "de-AT",
});
const portalPage = await portalContext.newPage();
await portalPage.goto(`${BASE_URL}/portal`, { waitUntil: "networkidle" });
await portalPage.screenshot({
  path: path.join(SCREENSHOTS_DIR, "portal-login.png"),
  fullPage: true,
});
await portalContext.close();

await browser.close();

const count = fs.readdirSync(SCREENSHOTS_DIR).filter(f => f.endsWith(".png")).length;
console.log(`\nDone — ${count} screenshots in docs/screenshots/`);
