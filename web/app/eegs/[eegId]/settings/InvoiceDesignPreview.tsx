"use client";

import { useEffect, useRef, useState } from "react";

interface Props {
  eegId: string;
  initialDesign: string;
  initialAccentColor: string;
  initialLogoLeft: boolean;
  initialFontFamily: string;
  initialFontSize: number;
  initialRowSpacing: number;
  initialFooterText: string;
  initialPreText: string;
  initialPostText: string;
  initialLabelZeitraumVon: string;
  initialLabelZeitraumBis: string;
  initialLabelGesamtverbrauch: string;
  initialLabelNetzbezug: string;
  initialLabelCommunityVerbrauch: string;
  initialLabelGesamteinspeisung: string;
  initialLabelAbnahmeEnergiegemeinschaft: string;
  initialLabelResteinspeisung: string;
  initialShowMonthlyBreakdown: boolean;
  initialLogoScale: number;
  initialAlwaysShowZaehlpunkt: boolean;
  initialChartType: string;
  initialChartTitle: string;
  initialChartColorCommunityBezug: string;
  initialChartColorNetzbezug: string;
  initialChartColorCommunityEinspeisung: string;
  initialChartColorResteinspeisung: string;
  initialChartLabelCommunity: string;
  initialChartLabelCommunityBezug: string;
  initialChartLabelCommunityEinspeisung: string;
  initialChartLabelNetzbezug: string;
  initialChartLabelResteinspeisung: string;
  initialChartLabelBezug: string;
  initialChartLabelEinspeisung: string;
}

const DEFAULT_CHART_TITLES: Record<string, string> = {
  absolute: "Wie hat sich Ihr Verbrauch / Ihre Einspeisung entwickelt?",
  percentage: "Wie hat sich Ihr Verbrauch / Ihre Einspeisung und Ihr Community-Anteil entwickelt?",
};

const FONT_FAMILIES = [
  { value: "dejavu", label: "DejaVu Sans (Standard)" },
  { value: "roboto", label: "Roboto" },
  { value: "opensans", label: "Open Sans" },
  { value: "ptserif", label: "PT Serif" },
];

const FONT_SIZES = [8, 9, 10, 11, 12];

const ROW_SPACINGS = [0.7, 0.8, 0.9, 1.0, 1.1, 1.2, 1.3];

const LOGO_SCALES = [0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0, 2.25, 2.5];

const inputClass =
  "w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent";
const labelClass = "block text-sm font-medium text-slate-700 mb-1.5";

export default function InvoiceDesignPreview({
  eegId,
  initialDesign,
  initialAccentColor,
  initialLogoLeft,
  initialFontFamily,
  initialFontSize,
  initialRowSpacing,
  initialFooterText,
  initialPreText,
  initialPostText,
  initialLabelZeitraumVon,
  initialLabelZeitraumBis,
  initialLabelGesamtverbrauch,
  initialLabelNetzbezug,
  initialLabelCommunityVerbrauch,
  initialLabelGesamteinspeisung,
  initialLabelAbnahmeEnergiegemeinschaft,
  initialLabelResteinspeisung,
  initialShowMonthlyBreakdown,
  initialLogoScale,
  initialAlwaysShowZaehlpunkt,
  initialChartType,
  initialChartTitle,
  initialChartColorCommunityBezug,
  initialChartColorNetzbezug,
  initialChartColorCommunityEinspeisung,
  initialChartColorResteinspeisung,
  initialChartLabelCommunity,
  initialChartLabelCommunityBezug,
  initialChartLabelCommunityEinspeisung,
  initialChartLabelNetzbezug,
  initialChartLabelResteinspeisung,
  initialChartLabelBezug,
  initialChartLabelEinspeisung,
}: Props) {
  const [design, setDesign] = useState(initialDesign || "standard");
  const [accentColor, setAccentColor] = useState(initialAccentColor || "#c9b89a");
  const [logoLeft, setLogoLeft] = useState(initialLogoLeft);
  const [fontFamily, setFontFamily] = useState(initialFontFamily || "dejavu");
  const [fontSize, setFontSize] = useState(initialFontSize || 10);
  const [rowSpacing, setRowSpacing] = useState(initialRowSpacing || 1.0);
  const [footerText, setFooterText] = useState(initialFooterText || "");
  const [preText, setPreText] = useState(initialPreText || "");
  const [postText, setPostText] = useState(initialPostText || "");
  const [labelZeitraumVon, setLabelZeitraumVon] = useState(initialLabelZeitraumVon || "Zeitraum von");
  const [labelZeitraumBis, setLabelZeitraumBis] = useState(initialLabelZeitraumBis || "Zeitraum bis");
  const [labelGesamtverbrauch, setLabelGesamtverbrauch] = useState(initialLabelGesamtverbrauch || "Gesamtverbrauch kWh");
  const [labelNetzbezug, setLabelNetzbezug] = useState(initialLabelNetzbezug || "Netzbezug kWh");
  const [labelCommunityVerbrauch, setLabelCommunityVerbrauch] = useState(initialLabelCommunityVerbrauch || "Community-Verbrauch kWh");
  const [labelGesamteinspeisung, setLabelGesamteinspeisung] = useState(initialLabelGesamteinspeisung || "Gesamteinspeisung kWh");
  const [labelAbnahmeEnergiegemeinschaft, setLabelAbnahmeEnergiegemeinschaft] = useState(
    initialLabelAbnahmeEnergiegemeinschaft || "Abnahme durch Energiegemeinschaft kWh"
  );
  const [labelResteinspeisung, setLabelResteinspeisung] = useState(initialLabelResteinspeisung || "Resteinspeisung kWh");
  const [showMonthlyBreakdown, setShowMonthlyBreakdown] = useState(initialShowMonthlyBreakdown);
  const [logoScale, setLogoScale] = useState(initialLogoScale || 1.0);
  const [alwaysShowZaehlpunkt, setAlwaysShowZaehlpunkt] = useState(initialAlwaysShowZaehlpunkt);
  const [chartType, setChartType] = useState(initialChartType || "absolute");
  const [chartTitle, setChartTitle] = useState(initialChartTitle || "");
  const [chartColorCommunityBezug, setChartColorCommunityBezug] = useState(initialChartColorCommunityBezug || "#22c55e");
  const [chartColorNetzbezug, setChartColorNetzbezug] = useState(initialChartColorNetzbezug || "#f59e0b");
  const [chartColorCommunityEinspeisung, setChartColorCommunityEinspeisung] = useState(
    initialChartColorCommunityEinspeisung || "#22c55e"
  );
  const [chartColorResteinspeisung, setChartColorResteinspeisung] = useState(initialChartColorResteinspeisung || "#3b82f6");
  const [chartLabelCommunity, setChartLabelCommunity] = useState(initialChartLabelCommunity || "");
  const [chartLabelCommunityBezug, setChartLabelCommunityBezug] = useState(initialChartLabelCommunityBezug || "");
  const [chartLabelCommunityEinspeisung, setChartLabelCommunityEinspeisung] = useState(
    initialChartLabelCommunityEinspeisung || ""
  );
  const [chartLabelNetzbezug, setChartLabelNetzbezug] = useState(initialChartLabelNetzbezug || "");
  const [chartLabelResteinspeisung, setChartLabelResteinspeisung] = useState(initialChartLabelResteinspeisung || "");
  const [chartLabelBezug, setChartLabelBezug] = useState(initialChartLabelBezug || "");
  const [chartLabelEinspeisung, setChartLabelEinspeisung] = useState(initialChartLabelEinspeisung || "");

  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const objectUrlRef = useRef<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    const timer = setTimeout(async () => {
      try {
        const res = await fetch(`/api/eegs/${eegId}/invoice-design/preview`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            invoice_design: design,
            invoice_accent_color: accentColor,
            invoice_logo_left: logoLeft,
            invoice_font_family: fontFamily,
            invoice_font_size: fontSize,
            invoice_row_spacing: rowSpacing,
            invoice_footer_text: footerText,
            invoice_pre_text: preText,
            invoice_post_text: postText,
            invoice_energy_label_zeitraum_von: labelZeitraumVon,
            invoice_energy_label_zeitraum_bis: labelZeitraumBis,
            invoice_energy_label_gesamtverbrauch: labelGesamtverbrauch,
            invoice_energy_label_netzbezug: labelNetzbezug,
            invoice_energy_label_community_verbrauch: labelCommunityVerbrauch,
            invoice_energy_label_gesamteinspeisung: labelGesamteinspeisung,
            invoice_energy_label_abnahme_energiegemeinschaft: labelAbnahmeEnergiegemeinschaft,
            invoice_energy_label_resteinspeisung: labelResteinspeisung,
            invoice_show_monthly_breakdown: showMonthlyBreakdown,
            invoice_logo_scale: logoScale,
            invoice_always_show_zaehlpunkt: alwaysShowZaehlpunkt,
            invoice_chart_type: chartType,
            invoice_chart_title: chartTitle,
            invoice_chart_color_community_bezug: chartColorCommunityBezug,
            invoice_chart_color_netzbezug: chartColorNetzbezug,
            invoice_chart_color_community_einspeisung: chartColorCommunityEinspeisung,
            invoice_chart_color_resteinspeisung: chartColorResteinspeisung,
            invoice_chart_label_community: chartLabelCommunity,
            invoice_chart_label_community_bezug: chartLabelCommunityBezug,
            invoice_chart_label_community_einspeisung: chartLabelCommunityEinspeisung,
            invoice_chart_label_netzbezug: chartLabelNetzbezug,
            invoice_chart_label_resteinspeisung: chartLabelResteinspeisung,
            invoice_chart_label_bezug: chartLabelBezug,
            invoice_chart_label_einspeisung: chartLabelEinspeisung,
          }),
        });
        if (cancelled) return;
        if (!res.ok) {
          setError("Vorschau konnte nicht geladen werden.");
          setLoading(false);
          return;
        }
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        if (objectUrlRef.current) URL.revokeObjectURL(objectUrlRef.current);
        objectUrlRef.current = url;
        if (!cancelled) {
          setPreviewUrl(url);
          setLoading(false);
        }
      } catch {
        if (!cancelled) {
          setError("Netzwerkfehler beim Laden der Vorschau.");
          setLoading(false);
        }
      }
    }, 400);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    eegId,
    design,
    accentColor,
    logoLeft,
    fontFamily,
    fontSize,
    rowSpacing,
    footerText,
    preText,
    postText,
    labelZeitraumVon,
    labelZeitraumBis,
    labelGesamtverbrauch,
    labelNetzbezug,
    labelCommunityVerbrauch,
    labelGesamteinspeisung,
    labelAbnahmeEnergiegemeinschaft,
    labelResteinspeisung,
    showMonthlyBreakdown,
    logoScale,
    alwaysShowZaehlpunkt,
    chartType,
    chartTitle,
    chartColorCommunityBezug,
    chartColorNetzbezug,
    chartColorCommunityEinspeisung,
    chartColorResteinspeisung,
    chartLabelCommunity,
    chartLabelCommunityBezug,
    chartLabelCommunityEinspeisung,
    chartLabelNetzbezug,
    chartLabelResteinspeisung,
    chartLabelBezug,
    chartLabelEinspeisung,
  ]);

  useEffect(() => {
    return () => {
      if (objectUrlRef.current) URL.revokeObjectURL(objectUrlRef.current);
    };
  }, []);

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div className="space-y-6">
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-1">Rechnungsdesign</h2>
          <p className="text-xs text-slate-500 mb-4">
            Legt fest, ob PDFs (Rechnung, Gutschrift, Storno) im bisherigen Standard-Layout oder im
            anpassbaren "Individuell"-Layout erzeugt werden.
          </p>
          <div className="space-y-3">
            <label className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 cursor-pointer has-[:checked]:border-blue-500 has-[:checked]:bg-blue-50">
              <input
                type="radio"
                name="invoice_design"
                value="standard"
                checked={design === "standard"}
                onChange={() => setDesign("standard")}
                className="mt-0.5 h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
              />
              <div>
                <p className="text-sm font-medium text-slate-800">Standard</p>
                <p className="text-xs text-slate-500 mt-0.5">Das bisherige Layout — unverändert.</p>
              </div>
            </label>
            <label className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 cursor-pointer has-[:checked]:border-blue-500 has-[:checked]:bg-blue-50">
              <input
                type="radio"
                name="invoice_design"
                value="individuell"
                checked={design === "individuell"}
                onChange={() => setDesign("individuell")}
                className="mt-0.5 h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
              />
              <div>
                <p className="text-sm font-medium text-slate-800">Individuell</p>
                <p className="text-xs text-slate-500 mt-0.5">
                  Alternatives Layout mit Akzentfarbe, Logo-Position und wählbarer Schrift.
                </p>
              </div>
            </label>
          </div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-4">
          <div>
            <h2 className="text-base font-semibold text-slate-900 mb-1">Rechnungstexte</h2>
            <p className="text-xs text-slate-500">
              Freitexte auf Rechnung/Gutschrift/Storno. Gilt für beide Designs.
            </p>
          </div>
          <div>
            <label className={labelClass}>Text vor Positionen</label>
            <textarea
              name="invoice_pre_text"
              rows={3}
              value={preText}
              onChange={(e) => setPreText(e.target.value)}
              placeholder="Text der vor den Rechnungspositionen erscheint..."
              className={inputClass}
            />
          </div>
          <div>
            <label className={labelClass}>Text nach Positionen</label>
            <textarea
              name="invoice_post_text"
              rows={3}
              value={postText}
              onChange={(e) => setPostText(e.target.value)}
              placeholder="Text der nach den Rechnungspositionen erscheint..."
              className={inputClass}
            />
          </div>
          <div>
            <label className={labelClass}>Fußzeile</label>
            <textarea
              name="invoice_footer_text"
              rows={2}
              value={footerText}
              onChange={(e) => setFooterText(e.target.value)}
              placeholder="Erstellt von eegabrechnung"
              className={inputClass}
            />
          </div>
        </div>

        {design === "individuell" && (
          <div className="bg-white rounded-xl border border-slate-200 p-6 space-y-4">
            <h2 className="text-base font-semibold text-slate-900 mb-1">Individuell-Einstellungen</h2>

            <div>
              <label className={labelClass}>Akzentfarbe</label>
              <div className="flex items-center gap-3">
                <input
                  type="color"
                  value={accentColor}
                  onChange={(e) => setAccentColor(e.target.value)}
                  className="h-10 w-14 rounded border border-slate-300 cursor-pointer"
                />
                <input
                  type="text"
                  name="invoice_accent_color"
                  value={accentColor}
                  onChange={(e) => setAccentColor(e.target.value)}
                  pattern="^#[0-9a-fA-F]{6}$"
                  className={inputClass}
                />
              </div>
            </div>

            <div>
              <label className={labelClass}>Logo-Position</label>
              <div className="flex gap-4">
                <label className="flex items-center gap-2 text-sm text-slate-700">
                  <input
                    type="radio"
                    name="invoice_logo_left"
                    value="true"
                    checked={logoLeft}
                    onChange={() => setLogoLeft(true)}
                    className="h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  Links
                </label>
                <label className="flex items-center gap-2 text-sm text-slate-700">
                  <input
                    type="radio"
                    name="invoice_logo_left"
                    value="false"
                    checked={!logoLeft}
                    onChange={() => setLogoLeft(false)}
                    className="h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  Rechts
                </label>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className={labelClass}>Schriftart</label>
                <select
                  name="invoice_font_family"
                  value={fontFamily}
                  onChange={(e) => setFontFamily(e.target.value)}
                  className={inputClass}
                >
                  {FONT_FAMILIES.map((f) => (
                    <option key={f.value} value={f.value}>
                      {f.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className={labelClass}>Schriftgröße</label>
                <select
                  name="invoice_font_size"
                  value={fontSize}
                  onChange={(e) => setFontSize(parseInt(e.target.value))}
                  className={inputClass}
                >
                  {FONT_SIZES.map((s) => (
                    <option key={s} value={s}>
                      {s} pt
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div>
              <label className={labelClass}>Zeilenabstand / Tabellenzeilen-Padding</label>
              <select
                name="invoice_row_spacing"
                value={rowSpacing}
                onChange={(e) => setRowSpacing(parseFloat(e.target.value))}
                className={inputClass}
              >
                {ROW_SPACINGS.map((s) => (
                  <option key={s} value={s}>
                    {Math.round(s * 100)} % {s === 1.0 ? "(Standard)" : s < 1.0 ? "(kompakter)" : "(geräumiger)"}
                  </option>
                ))}
              </select>
              <p className="text-xs text-slate-400 mt-1">
                Skaliert alle Zeilenhöhen und Tabellenzeilen-Paddings gemeinsam — weniger Prozent
                bedeutet weniger Leerraum zwischen Zeilen.
              </p>
            </div>

            <div>
              <label className={labelClass}>Logogröße</label>
              <select
                name="invoice_logo_scale"
                value={logoScale}
                onChange={(e) => setLogoScale(parseFloat(e.target.value))}
                className={inputClass}
              >
                {LOGO_SCALES.map((s) => (
                  <option key={s} value={s}>
                    {Math.round(s * 100)} % {s === 1.0 ? "(Standard)" : s < 1.0 ? "(kleiner)" : "(größer)"}
                  </option>
                ))}
              </select>
              <p className="text-xs text-slate-400 mt-1">
                Skaliert die Höhe des Firmenlogos relativ zur Höhe des Adressblocks. Bei sehr großen Werten und
                langen Adressen kann das Logo mit dem Empfängerblock kollidieren — im Zweifel in der Vorschau
                prüfen.
              </p>
            </div>

            <div>
              <label className={labelClass}>Verbrauchsentwicklungsgrafik</label>
              <div className="flex gap-4">
                <label className="flex items-center gap-2 text-sm text-slate-700">
                  <input
                    type="radio"
                    name="invoice_chart_type"
                    value="absolute"
                    checked={chartType === "absolute"}
                    onChange={() => setChartType("absolute")}
                    className="h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  Absolute Werte (kWh)
                </label>
                <label className="flex items-center gap-2 text-sm text-slate-700">
                  <input
                    type="radio"
                    name="invoice_chart_type"
                    value="percentage"
                    checked={chartType === "percentage"}
                    onChange={() => setChartType("percentage")}
                    className="h-4 w-4 border-slate-300 text-blue-700 focus:ring-blue-500"
                  />
                  Community-Anteil (%)
                </label>
              </div>
              <p className="text-xs text-slate-400 mt-1">
                Zeigt statt absoluter kWh-Balken pro Monat den prozentualen Anteil von Community-Bezug/Netzbezug
                (und bei Einspeisung: Community-Abnahme/Resteinspeisung) als gestapelten Balken.
              </p>
            </div>

            <div>
              <label className={labelClass}>Grafik-Titel</label>
              <input
                type="text"
                name="invoice_chart_title"
                value={chartTitle}
                onChange={(e) => setChartTitle(e.target.value)}
                placeholder={DEFAULT_CHART_TITLES[chartType] || DEFAULT_CHART_TITLES.absolute}
                className={inputClass}
              />
              <p className="text-xs text-slate-400 mt-1">
                Überschrift über der Verbrauchsentwicklungsgrafik. Leer lassen für die Standard-Überschrift
                (abhängig von der oben gewählten Grafik-Variante).
              </p>
            </div>

            <div>
              <p className="text-sm font-medium text-slate-800 mb-1">Grafikfarben</p>
              <p className="text-xs text-slate-500 mb-3">
                Community-Anteil (Bezug/Einspeisung) wirken auf beide Grafik-Varianten; Netzbezug und
                Resteinspeisung nur auf die Community-Anteil-(%)-Variante.
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Community-Anteil (Bezug)</label>
                  <div className="flex items-center gap-3">
                    <input
                      type="color"
                      value={chartColorCommunityBezug}
                      onChange={(e) => setChartColorCommunityBezug(e.target.value)}
                      className="h-10 w-14 rounded border border-slate-300 cursor-pointer"
                    />
                    <input
                      type="text"
                      name="invoice_chart_color_community_bezug"
                      value={chartColorCommunityBezug}
                      onChange={(e) => setChartColorCommunityBezug(e.target.value)}
                      pattern="^#[0-9a-fA-F]{6}$"
                      className={inputClass}
                    />
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Netzbezug</label>
                  <div className="flex items-center gap-3">
                    <input
                      type="color"
                      value={chartColorNetzbezug}
                      onChange={(e) => setChartColorNetzbezug(e.target.value)}
                      className="h-10 w-14 rounded border border-slate-300 cursor-pointer"
                    />
                    <input
                      type="text"
                      name="invoice_chart_color_netzbezug"
                      value={chartColorNetzbezug}
                      onChange={(e) => setChartColorNetzbezug(e.target.value)}
                      pattern="^#[0-9a-fA-F]{6}$"
                      className={inputClass}
                    />
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Community-Anteil (Einspeisung)</label>
                  <div className="flex items-center gap-3">
                    <input
                      type="color"
                      value={chartColorCommunityEinspeisung}
                      onChange={(e) => setChartColorCommunityEinspeisung(e.target.value)}
                      className="h-10 w-14 rounded border border-slate-300 cursor-pointer"
                    />
                    <input
                      type="text"
                      name="invoice_chart_color_community_einspeisung"
                      value={chartColorCommunityEinspeisung}
                      onChange={(e) => setChartColorCommunityEinspeisung(e.target.value)}
                      pattern="^#[0-9a-fA-F]{6}$"
                      className={inputClass}
                    />
                  </div>
                </div>
                <div>
                  <label className={labelClass}>Resteinspeisung</label>
                  <div className="flex items-center gap-3">
                    <input
                      type="color"
                      value={chartColorResteinspeisung}
                      onChange={(e) => setChartColorResteinspeisung(e.target.value)}
                      className="h-10 w-14 rounded border border-slate-300 cursor-pointer"
                    />
                    <input
                      type="text"
                      name="invoice_chart_color_resteinspeisung"
                      value={chartColorResteinspeisung}
                      onChange={(e) => setChartColorResteinspeisung(e.target.value)}
                      pattern="^#[0-9a-fA-F]{6}$"
                      className={inputClass}
                    />
                  </div>
                </div>
              </div>
            </div>

            <div>
              <p className="text-sm font-medium text-slate-800 mb-1">Grafik-Beschriftungen</p>
              <p className="text-xs text-slate-500 mb-3">
                Legendentexte der Verbrauchsentwicklungsgrafik. Community-Anteil (Bezug/Einspeisung) wirken nur,
                wenn die beiden Community-Farben oben unterschiedlich sind — sonst gilt "Community-Anteil"
                gemeinsam. Bezug/Einspeisung gelten nur für die Absolute-Werte-Variante. Leer lassen für die
                Standardbezeichnung.
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div className="col-span-2">
                  <label className={labelClass}>Community-Anteil (bei gleicher Farbe)</label>
                  <input
                    type="text"
                    name="invoice_chart_label_community"
                    value={chartLabelCommunity}
                    onChange={(e) => setChartLabelCommunity(e.target.value)}
                    placeholder="Community-Anteil"
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Community-Anteil (Bezug)</label>
                  <input
                    type="text"
                    name="invoice_chart_label_community_bezug"
                    value={chartLabelCommunityBezug}
                    onChange={(e) => setChartLabelCommunityBezug(e.target.value)}
                    placeholder="Community-Anteil (Bezug)"
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Community-Anteil (Einspeisung)</label>
                  <input
                    type="text"
                    name="invoice_chart_label_community_einspeisung"
                    value={chartLabelCommunityEinspeisung}
                    onChange={(e) => setChartLabelCommunityEinspeisung(e.target.value)}
                    placeholder="Community-Anteil (Einspeisung)"
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Netzbezug</label>
                  <input
                    type="text"
                    name="invoice_chart_label_netzbezug"
                    value={chartLabelNetzbezug}
                    onChange={(e) => setChartLabelNetzbezug(e.target.value)}
                    placeholder="Netzbezug"
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Resteinspeisung</label>
                  <input
                    type="text"
                    name="invoice_chart_label_resteinspeisung"
                    value={chartLabelResteinspeisung}
                    onChange={(e) => setChartLabelResteinspeisung(e.target.value)}
                    placeholder="Resteinspeisung"
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Bezug (kWh)</label>
                  <input
                    type="text"
                    name="invoice_chart_label_bezug"
                    value={chartLabelBezug}
                    onChange={(e) => setChartLabelBezug(e.target.value)}
                    placeholder="Bezug (kWh)"
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Einspeisung (kWh)</label>
                  <input
                    type="text"
                    name="invoice_chart_label_einspeisung"
                    value={chartLabelEinspeisung}
                    onChange={(e) => setChartLabelEinspeisung(e.target.value)}
                    placeholder="Einspeisung (kWh)"
                    className={inputClass}
                  />
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3 pt-2 border-t border-slate-100">
              <input
                type="checkbox"
                id="invoice_show_monthly_breakdown"
                name="invoice_show_monthly_breakdown"
                checked={showMonthlyBreakdown}
                onChange={(e) => setShowMonthlyBreakdown(e.target.checked)}
                className="h-4 w-4 rounded border-slate-300 text-blue-700 focus:ring-blue-500"
              />
              <label htmlFor="invoice_show_monthly_breakdown" className="text-sm text-slate-700">
                Bei mehrmonatigen Rechnungen jeden Monat einzeln anzeigen
                <span className="block text-xs text-slate-500 mt-0.5">
                  Aus: Mess- und Preistabelle zeigen nur die Periodensumme je Zählpunkt/Richtung statt einer
                  Zeile pro Monat. Hat sich der Tarif innerhalb der Periode tatsächlich monatlich geändert (z.B.
                  monatlicher Tarifplan bei quartalsweiser Abrechnung), wird trotzdem automatisch monatlich
                  angezeigt — sonst wäre der Preis je kWh nicht mehr eindeutig.
                </span>
              </label>
            </div>

            <div className="flex items-center gap-3">
              <input
                type="checkbox"
                id="invoice_always_show_zaehlpunkt"
                name="invoice_always_show_zaehlpunkt"
                checked={alwaysShowZaehlpunkt}
                onChange={(e) => setAlwaysShowZaehlpunkt(e.target.checked)}
                className="h-4 w-4 rounded border-slate-300 text-blue-700 focus:ring-blue-500"
              />
              <label htmlFor="invoice_always_show_zaehlpunkt" className="text-sm text-slate-700">
                Zählpunkt(e) immer anzeigen
                <span className="block text-xs text-slate-500 mt-0.5">
                  Normalerweise wird die Zählpunkt-Zwischenüberschrift nur angezeigt, wenn eine Rechnung mehr
                  als einen Zählpunkt umfasst. Diese Option zeigt sie immer an, auch bei nur einem Zählpunkt.
                </span>
              </label>
            </div>

            <div className="pt-2 border-t border-slate-100">
              <p className="text-sm font-medium text-slate-800 mb-1">Bezeichnungen der Energie-Tabelle</p>
              <p className="text-xs text-slate-500 mb-3">
                Spaltenüberschriften der Netzbezug/Community-Verbrauch-Tabelle.
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Zeitraum von</label>
                  <input
                    type="text"
                    name="invoice_energy_label_zeitraum_von"
                    value={labelZeitraumVon}
                    onChange={(e) => setLabelZeitraumVon(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Zeitraum bis</label>
                  <input
                    type="text"
                    name="invoice_energy_label_zeitraum_bis"
                    value={labelZeitraumBis}
                    onChange={(e) => setLabelZeitraumBis(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Gesamtverbrauch</label>
                  <input
                    type="text"
                    name="invoice_energy_label_gesamtverbrauch"
                    value={labelGesamtverbrauch}
                    onChange={(e) => setLabelGesamtverbrauch(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Community-Verbrauch</label>
                  <input
                    type="text"
                    name="invoice_energy_label_community_verbrauch"
                    value={labelCommunityVerbrauch}
                    onChange={(e) => setLabelCommunityVerbrauch(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div className="col-span-2">
                  <label className={labelClass}>Netzbezug</label>
                  <input
                    type="text"
                    name="invoice_energy_label_netzbezug"
                    value={labelNetzbezug}
                    onChange={(e) => setLabelNetzbezug(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
            </div>

            <div className="pt-2 border-t border-slate-100">
              <p className="text-sm font-medium text-slate-800 mb-1">Bezeichnungen der Einspeise-Tabelle</p>
              <p className="text-xs text-slate-500 mb-3">
                Spaltenüberschriften der Gesamteinspeisung/Resteinspeisung-Tabelle (nur bei Mitgliedern mit Einspeisung).
              </p>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>Gesamteinspeisung</label>
                  <input
                    type="text"
                    name="invoice_energy_label_gesamteinspeisung"
                    value={labelGesamteinspeisung}
                    onChange={(e) => setLabelGesamteinspeisung(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div>
                  <label className={labelClass}>Resteinspeisung</label>
                  <input
                    type="text"
                    name="invoice_energy_label_resteinspeisung"
                    value={labelResteinspeisung}
                    onChange={(e) => setLabelResteinspeisung(e.target.value)}
                    className={inputClass}
                  />
                </div>
                <div className="col-span-2">
                  <label className={labelClass}>Abnahme durch Energiegemeinschaft</label>
                  <input
                    type="text"
                    name="invoice_energy_label_abnahme_energiegemeinschaft"
                    value={labelAbnahmeEnergiegemeinschaft}
                    onChange={(e) => setLabelAbnahmeEnergiegemeinschaft(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="bg-white rounded-xl border border-slate-200 p-4 lg:sticky lg:top-4 self-start">
        <p className="text-xs font-medium text-slate-700 mb-2">Vorschau mit Beispieldaten</p>
        <div className="relative border border-slate-200 rounded-lg overflow-hidden bg-slate-50" style={{ height: "70vh" }}>
          {loading && (
            <div className="absolute inset-0 flex items-center justify-center text-sm text-slate-400">
              Vorschau wird geladen…
            </div>
          )}
          {error && !loading && (
            <div className="absolute inset-0 flex items-center justify-center text-sm text-red-600 p-4 text-center">
              {error}
            </div>
          )}
          {previewUrl && (
            <iframe src={previewUrl} className="w-full h-full border-0" title="Rechnungsdesign-Vorschau" />
          )}
        </div>
      </div>
    </div>
  );
}
