import { chromium } from "playwright";
import path from "path";
import { fileURLToPath } from "url";
import fs from "fs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const outputPath = path.join(repoRoot, "docs", "screenshots", "reports-month.png");

const BASE_URL = process.env.SCREENSHOT_BASE_URL || "http://localhost:3001";
const EEG_ID = process.env.SCREENSHOT_EEG_ID || "5d0151e8-8714-4605-9f20-70ec5d5d5b46";
const EMAIL = process.env.SCREENSHOT_EMAIL || "admin@eeg.at";
const PASSWORD = process.env.SCREENSHOT_PASSWORD || "admin";
const MONTH = process.env.SCREENSHOT_MONTH || "2026-03";
const EXECUTABLE_PATH = process.env.PLAYWRIGHT_EXECUTABLE_PATH || "/usr/bin/chromium-browser";

fs.mkdirSync(path.dirname(outputPath), { recursive: true });

const browser = await chromium.launch({
  executablePath: EXECUTABLE_PATH,
  args: ["--no-sandbox", "--disable-setuid-sandbox"],
});

const context = await browser.newContext({
  viewport: { width: 1440, height: 900 },
  locale: "de-AT",
});

const page = await context.newPage();

await page.goto(`${BASE_URL}/auth/signin`, { waitUntil: "networkidle" });
await page.fill('input[name="email"]', EMAIL);
await page.fill('input[name="password"]', PASSWORD);
await page.click('button[type="submit"]');
await page.waitForURL((url) => !url.toString().includes("signin"), { timeout: 15000 });
await page.waitForLoadState("networkidle");

await page.goto(`${BASE_URL}/eegs/${EEG_ID}/reports`, { waitUntil: "networkidle" });
await page.getByRole("button", { name: "Monat" }).click();
await page.waitForLoadState("networkidle");
await page.locator('input[type="month"]').fill(MONTH);
await page.waitForLoadState("networkidle");
await page.waitForTimeout(1200);

await page.screenshot({
  path: outputPath,
  fullPage: true,
});

await browser.close();

console.log(`Saved ${outputPath}`);
