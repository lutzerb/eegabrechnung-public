"use client";

import { useState, useRef, useCallback } from "react";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
} from "recharts";

function useContainerWidth() {
  const [width, setWidth] = useState(0);
  const roRef = useRef<ResizeObserver | null>(null);

  const ref = useCallback((node: HTMLDivElement | null) => {
    if (roRef.current) { roRef.current.disconnect(); roRef.current = null; }
    if (!node) return;
    setWidth(Math.round(node.getBoundingClientRect().width));
    const ro = new ResizeObserver(([entry]) => {
      setWidth(Math.round(entry.contentRect.width));
    });
    ro.observe(node);
    roRef.current = ro;
  }, []);

  return [ref, width] as const;
}

interface ChartRow {
  label: string;
  "Bezug EEG": number;
  "Restbezug"?: number;
  "Einspeisung EEG"?: number;
  "Resteinspeisung"?: number;
}

interface Props {
  data: ChartRow[];
  hasGeneration: boolean;
  showFullEnergy: boolean;
}

const ALL_LEGEND_ITEMS = [
  { label: "Bezug EEG",       color: "#3b82f6", always: true,  generation: false },
  { label: "Restbezug",       color: "#cbd5e1", always: false, generation: false },
  { label: "Einspeisung EEG", color: "#10b981", always: true,  generation: true  },
  { label: "Resteinspeisung", color: "#6ee7b7", always: false, generation: true  },
];

export default function PortalEnergyChart({ data, hasGeneration, showFullEnergy }: Props) {
  const [containerRef, width] = useContainerWidth();
  const legendItems = ALL_LEGEND_ITEMS.filter(item =>
    (item.always || showFullEnergy) && (!item.generation || hasGeneration)
  );

  // Reduce tick density for large datasets (e.g. 96 rows for 15min view)
  const xAxisInterval = data.length > 50
    ? Math.ceil(data.length / 10) - 1
    : data.length > 20
      ? Math.ceil(data.length / 8) - 1
      : 0;

  return (
    <div ref={containerRef} style={{ width: "100%" }}>
      {width > 0 && (
        <BarChart width={width} height={400} data={data} margin={{ top: 5, right: 10, bottom: 5, left: 10 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
          <XAxis
            orientation="bottom" type="category" scale="auto" height={30} mirror={false}
            dataKey="label" tick={{ fontSize: 11 }} interval={xAxisInterval}
          />
          <YAxis orientation="left" type="number" scale="auto" mirror={false} tick={{ fontSize: 11 }} tickFormatter={(v: number) => `${v} kWh`} width={75} />
          <Tooltip
            formatter={(v: number, name: string) => [`${v.toLocaleString("de-AT")} kWh`, name]}
            contentStyle={{ fontSize: 12 }}
          />
          <Bar dataKey="Bezug EEG" stackId="bezug" fill="#3b82f6" maxBarSize={40} minPointSize={0} isAnimationActive={false} radius={showFullEnergy ? [0,0,0,0] : [2,2,0,0]} />
          {showFullEnergy && <Bar dataKey="Restbezug" stackId="bezug" fill="#cbd5e1" maxBarSize={40} minPointSize={0} radius={[2,2,0,0]} isAnimationActive={false} />}
          {hasGeneration && <Bar dataKey="Einspeisung EEG" stackId="einsp" fill="#10b981" maxBarSize={40} minPointSize={0} isAnimationActive={false} radius={showFullEnergy ? [0,0,0,0] : [2,2,0,0]} />}
          {hasGeneration && showFullEnergy && <Bar dataKey="Resteinspeisung" stackId="einsp" fill="#6ee7b7" maxBarSize={40} minPointSize={0} radius={[2,2,0,0]} isAnimationActive={false} />}
        </BarChart>
      )}
      {/* Custom legend outside recharts so spacing is fully controllable */}
      <div className="flex flex-wrap justify-center gap-4 mt-4 text-xs text-slate-500">
        {legendItems.map(item => (
          <div key={item.label} className="flex items-center gap-1.5">
            <span className="w-3 h-2.5 rounded-sm inline-block flex-shrink-0" style={{ backgroundColor: item.color }} />
            {item.label}
          </div>
        ))}
      </div>
    </div>
  );
}
