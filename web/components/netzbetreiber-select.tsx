"use client";

import { useState, useRef, useEffect } from "react";

export interface Netzbetreiber {
  ecNummer: string;
  name: string;
}

export const NETZBETREIBER_LIST: Netzbetreiber[] = [
  { ecNummer: "AT001000", name: "Wiener Netze GmbH" },
  { ecNummer: "AT002000", name: "Netz Niederösterreich GmbH" },
  { ecNummer: "AT002110", name: "Stadtwerke Amstetten GmbH" },
  { ecNummer: "AT002120", name: "Licht- und Kraftvertrieb der Marktgemeinde Göstling an der Ybbs" },
  { ecNummer: "AT002130", name: "Licht- und Kraftvertrieb der Gemeinde Hollenstein an der Ybbs" },
  { ecNummer: "AT002210", name: "Elektrizitätswerke Eisenhuber GmbH & Co KG" },
  { ecNummer: "AT002220", name: "Anton Kittel Mühle Plaika GmbH" },
  { ecNummer: "AT002230", name: "wüsterstrom E-Werk GmbH" },
  { ecNummer: "AT002250", name: "E-Werk Schwaighofer GmbH" },
  { ecNummer: "AT002270", name: "Polsterer Kerres Ruttin Holding GmbH" },
  { ecNummer: "AT002280", name: "Forstverwaltung Seehof GmbH" },
  { ecNummer: "AT002290", name: "Forstverwaltung Neuhaus Alpl Kraftwerksbetrieb" },
  { ecNummer: "AT002300", name: "Licht- und Kraftstromvertrieb der Gemeinde Opponitz" },
  { ecNummer: "AT002400", name: "Heinrich Polsterer und Mitges GesnbR" },
  { ecNummer: "AT002900", name: "E-Werk Sarmingstein Ing. H. Engelmann & Co KG" },
  { ecNummer: "AT002910", name: "Elektrizitätswerk Clam, Carl Philip Clam Martinic e.U." },
  { ecNummer: "AT003000", name: "Netz Oberösterreich GmbH" },
  { ecNummer: "AT003100", name: "LINZ NETZ GmbH" },
  { ecNummer: "AT003200", name: "ENERGIE RIED GmbH" },
  { ecNummer: "AT003300", name: "eww ag" },
  { ecNummer: "AT003310", name: "Elektrizitätswerk Perg GmbH" },
  { ecNummer: "AT003460", name: "EBNER STROM GmbH" },
  { ecNummer: "AT003470", name: "KARLSTROM e.U." },
  { ecNummer: "AT003510", name: "Kraftwerk Glatzing-Rüstorf eGen" },
  { ecNummer: "AT003520", name: "K.u.F. Drack GmbH & Co KG" },
  { ecNummer: "AT003540", name: "Revertera'sches Elektrizitätswerk" },
  { ecNummer: "AT003570", name: "EVU Mathe Gerald e.U." },
  { ecNummer: "AT003580", name: "Energieversorgungs GmbH" },
  { ecNummer: "AT003590", name: "E-Werk Ranklleiten" },
  { ecNummer: "AT003900", name: "E-Werk Altenfelden GmbH" },
  { ecNummer: "AT003910", name: "E-Werk Redlmühle / Verteilnetzbetrieb" },
  { ecNummer: "AT003920", name: "E-Werk Dietrichschlag eingetragene Genossenschaft" },
  { ecNummer: "AT004000", name: "Salzburg Netz GmbH" },
  { ecNummer: "AT004110", name: "Elektrizitätswerk Bad Hofgastein" },
  { ecNummer: "AT004120", name: "Lichtgenossenschaft Neukirchen eGen" },
  { ecNummer: "AT004130", name: "Elektrizitätswerk Lechner August KG" },
  { ecNummer: "AT005000", name: "TINETZ-Tiroler Netze GmbH" },
  { ecNummer: "AT005100", name: "Innsbrucker Kommunalbetriebe Aktiengesellschaft" },
  { ecNummer: "AT005110", name: "Elektrizitätswerke Reutte AG" },
  { ecNummer: "AT005120", name: "HALLAG Kommunal GmbH" },
  { ecNummer: "AT005130", name: "Stadtwerke Kitzbühel e.U." },
  { ecNummer: "AT005140", name: "Stadtwerke Kufstein Gesellschaft m.b.H." },
  { ecNummer: "AT005150", name: "Stadtwerke Schwaz GmbH" },
  { ecNummer: "AT005160", name: "Stadtwerke Wörgl GmbH" },
  { ecNummer: "AT005210", name: "Kraftwerk Haim KG" },
  { ecNummer: "AT005320", name: "Energie und Wirtschaftsbetriebe der Gemeinde St. Anton GmbH" },
  { ecNummer: "AT005330", name: "Kommunalbetriebe Hopfgarten GmbH" },
  { ecNummer: "AT005340", name: "Elektrizitätswerk Schattwald e.U." },
  { ecNummer: "AT005350", name: "Gemeindewerke Kematen" },
  { ecNummer: "AT005370", name: "Stadtwerke Imst" },
  { ecNummer: "AT005440", name: "Johann Dandler GmbH & CO KG" },
  { ecNummer: "AT005460", name: "Elektrizitätswerk Prantl GmbH & CoKG" },
  { ecNummer: "AT005470", name: "E-Werk Stadler GmbH" },
  { ecNummer: "AT005480", name: "Elektrizitätswerk Winkler GmbH" },
  { ecNummer: "AT005490", name: "Elektrowerk Assling reg. Gen. m. b. H." },
  { ecNummer: "AT005540", name: "Elektritzitätswerk der Gemeinde Gries am Brenner" },
  { ecNummer: "AT005600", name: "Elektrowerkgenossenschaft Hopfgarten i. D. reg. Gen.m.b.H." },
  { ecNummer: "AT005610", name: "Wasserkraft Sölden eGen mbH" },
  { ecNummer: "AT005630", name: "Elektrogenossenschaft Weerberg reg.genmbH" },
  { ecNummer: "AT005650", name: "Kommunalbetriebe Rinn GmbH" },
  { ecNummer: "AT006000", name: "Vorarlberger Energienetze GmbH" },
  { ecNummer: "AT006110", name: "Stadtwerke Feldkirch" },
  { ecNummer: "AT006210", name: "Elektrizitätswerke Frastanz Gesellschaft mbH" },
  { ecNummer: "AT006220", name: "Montafonerbahn Aktiengesellschaft" },
  { ecNummer: "AT006230", name: "Energieversorgung Kleinwalsertal GesmbH" },
  { ecNummer: "AT006240", name: "Getzner, Mutter & Cie. GmbH & Co KG" },
  { ecNummer: "AT006250", name: "Alfenzwerke Elektrizitätserzeugung GmbH" },
  { ecNummer: "AT007000", name: "KNG-Kärnten Netz GmbH" },
  { ecNummer: "AT007100", name: "Energie Klagenfurt GmbH" },
  { ecNummer: "AT007240", name: "AAE Wasserkraft GmbH" },
  { ecNummer: "AT008000", name: "Energienetze Steiermark GmbH" },
  { ecNummer: "AT008100", name: "Stromnetz Graz GmbH & Co KG" },
  { ecNummer: "AT008110", name: "Elektrizitätswerk der Stadtgemeinde Kindberg" },
  { ecNummer: "AT008120", name: "Stadtwerke Voitsberg GmbH" },
  { ecNummer: "AT008130", name: "Feistritzwerke Steweag GmbH" },
  { ecNummer: "AT008140", name: "Stadtwerke Bruck an der Mur GmbH" },
  { ecNummer: "AT008150", name: "Stadtwerke Fürstenfeld GmbH" },
  { ecNummer: "AT008160", name: "Stadtwerke Judenburg AG" },
  { ecNummer: "AT008170", name: "Stadtwerke Kapfenberg GmbH" },
  { ecNummer: "AT008180", name: "Stadtwerke Köflach" },
  { ecNummer: "AT008190", name: "Stadtwerke Mürzzuschlag GmbH" },
  { ecNummer: "AT008210", name: "E-WERK GÖSTING STROMVERSORGUNGS GmbH" },
  { ecNummer: "AT008310", name: "Bad Gleichenberger Energie GmbH" },
  { ecNummer: "AT008350", name: "Städtische Betriebe Rottenmann GmbH" },
  { ecNummer: "AT008360", name: "EVU der Stadtgemeinde Mureck" },
  { ecNummer: "AT008370", name: "E-Werk Mürzsteg" },
  { ecNummer: "AT008390", name: "Elektroversorgungsunternehmen der Marktgemeinde Eibiswald" },
  { ecNummer: "AT008410", name: "EVU der Marktgemeinde Unzmarkt-Frauenburg" },
  { ecNummer: "AT008420", name: "Marktgemeinde Neumarkt Versorgungsbetriebsgesellschaft m.b.H." },
  { ecNummer: "AT008430", name: "Murauer Stadtwerke Gesellschaft m.b.H." },
  { ecNummer: "AT008440", name: "Stadtbetriebe Mariazell GmbH" },
  { ecNummer: "AT008470", name: "Stadtwerke Hartberg Energieversorgungs GmbH" },
  { ecNummer: "AT008490", name: "Stadtwerke Trofaiach Ges.m.b.H." },
  { ecNummer: "AT008510", name: "E-Genossenschaft Laintal eGen" },
  { ecNummer: "AT008540", name: "E-Werk SIGL GmbH & CO KG" },
  { ecNummer: "AT008560", name: "ENVESTA Energie- u. Dienstleistungs GmbH" },
  { ecNummer: "AT008570", name: "Elektrizitätswerk Fernitz Ing. Franz Purkarthofer GmbH & Co KG" },
  { ecNummer: "AT008580", name: "Kiendler Vulkanlandstrom GmbH" },
  { ecNummer: "AT008620", name: "Elektrizitätswerk Gröbming KG" },
  { ecNummer: "AT008650", name: "Elektrizitätswerk Mariahof GmbH" },
  { ecNummer: "AT008690", name: "Gertraud Schafler GmbH." },
  { ecNummer: "AT008720", name: "Elektrowerk Schöder GmbH" },
  { ecNummer: "AT008740", name: "E-Werk Gleinstätten GmbH" },
  { ecNummer: "AT008770", name: "Feistritzthaler Elektrizitätswerk eGen" },
  { ecNummer: "AT008850", name: "Schwarz, Wagendorffer & Co, Elektrizitätswerk GmbH" },
  { ecNummer: "AT008910", name: "E-Werk Stubenberg eGen" },
  { ecNummer: "AT008930", name: "Joh. Pengg Holding Gesellschaft m.b.H." },
  { ecNummer: "AT008950", name: "E-Werk Piwetz" },
  { ecNummer: "AT008960", name: "E-Werk Neudau Kottulinsky KG" },
  { ecNummer: "AT009000", name: "Netz Burgenland GmbH" },
  { ecNummer: "AT009220", name: "Energie Güssing GmbH" },
];

interface Props {
  value: string;
  ecNummer: string;
  onChange: (name: string, ecNummer: string) => void;
  required?: boolean;
}

export default function NetzbetreiberSelect({ value, ecNummer, onChange, required }: Props) {
  const [query, setQuery] = useState(value);
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const filtered = query.length < 1
    ? NETZBETREIBER_LIST
    : NETZBETREIBER_LIST.filter(
        (nb) =>
          nb.name.toLowerCase().includes(query.toLowerCase()) ||
          nb.ecNummer.toLowerCase().includes(query.toLowerCase())
      );

  useEffect(() => {
    setQuery(value);
  }, [value]);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  function select(nb: Netzbetreiber) {
    onChange(nb.name, nb.ecNummer);
    setQuery(nb.name);
    setOpen(false);
  }

  function handleInput(e: React.ChangeEvent<HTMLInputElement>) {
    setQuery(e.target.value);
    setOpen(true);
    if (e.target.value === "") {
      onChange("", "");
    }
  }

  return (
    <div ref={ref} className="relative">
      <input
        type="text"
        required={required}
        value={query}
        onChange={handleInput}
        onFocus={() => setOpen(true)}
        placeholder="Name oder EC-Nummer eingeben…"
        autoComplete="off"
        className="w-full px-3 py-2 border border-slate-300 rounded-lg text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
      />
      {ecNummer && (
        <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-400 font-mono">
          {ecNummer}
        </span>
      )}
      {open && filtered.length > 0 && (
        <ul className="absolute z-50 mt-1 w-full max-h-64 overflow-y-auto bg-white border border-slate-200 rounded-lg shadow-lg">
          {filtered.map((nb) => (
            <li
              key={nb.ecNummer}
              onMouseDown={() => select(nb)}
              className="flex items-center justify-between px-3 py-2 hover:bg-blue-50 cursor-pointer text-sm"
            >
              <span className="text-slate-800 truncate pr-3">{nb.name}</span>
              <span className="text-slate-400 font-mono text-xs shrink-0">{nb.ecNummer}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
