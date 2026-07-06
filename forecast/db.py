import asyncpg
import os

DATABASE_URL = os.getenv("DATABASE_URL", "")

_pool = None


async def get_pool():
    global _pool
    if _pool is None:
        _pool = await asyncpg.create_pool(DATABASE_URL, min_size=1, max_size=5)
    return _pool


async def get_eeg_info(eeg_id: str) -> dict | None:
    p = await get_pool()
    row = await p.fetchrow(
        "SELECT id::text, name, plz, ort FROM eegs WHERE id = $1::uuid",
        eeg_id,
    )
    return dict(row) if row else None


async def get_hourly_readings(eeg_id: str, days_back: int = 365) -> list[dict]:
    """Hourly aggregated energy readings for an EEG, last N days.

    Returns consumption_kwh (sum of CONSUMPTION ZPs wh_total) and
    generation_kwh (sum of GENERATION ZPs wh_total), both in kWh.
    Quality L3 (faulty) readings are excluded.
    """
    p = await get_pool()
    rows = await p.fetch(
        """
        SELECT
            date_trunc('hour', r.ts AT TIME ZONE 'Europe/Vienna')::text AS hour,
            SUM(CASE WHEN mp.energierichtung = 'CONSUMPTION' THEN r.wh_total ELSE 0 END)
                AS consumption_kwh,
            SUM(CASE WHEN mp.energierichtung = 'GENERATION' THEN r.wh_total ELSE 0 END)
                AS generation_kwh
        FROM energy_readings r
        JOIN meter_points mp ON mp.id = r.meter_point_id
        JOIN members m ON m.id = mp.member_id
        WHERE m.eeg_id = $1::uuid
          AND r.ts >= NOW() - ($2 * INTERVAL '1 day')
          AND r.quality <> 'L3'
        GROUP BY date_trunc('hour', r.ts AT TIME ZONE 'Europe/Vienna')
        ORDER BY 1
        """,
        eeg_id,
        days_back,
    )
    return [
        {
            "hour": r["hour"],
            "consumption_kwh": float(r["consumption_kwh"]),
            "generation_kwh": float(r["generation_kwh"]),
        }
        for r in rows
    ]
