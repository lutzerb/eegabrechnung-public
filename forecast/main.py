import asyncio
import logging
from contextlib import asynccontextmanager
from datetime import datetime, timedelta

from fastapi import BackgroundTasks, FastAPI, HTTPException

import db
import model
import weather

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("forecast")

# eeg_id → "training" | "ok" | "insufficient_data" | "error"
_status: dict[str, str] = {}

# eeg_id → {"data": ..., "at": datetime}
_cache: dict[str, dict] = {}

# Track which EEG IDs have been seen, for the weekly retrain loop.
_known_eegs: set[str] = set()


async def _train(eeg_id: str) -> None:
    _status[eeg_id] = "training"
    try:
        eeg = await db.get_eeg_info(eeg_id)
        if not eeg:
            _status[eeg_id] = "error"
            logger.error("EEG %s not found", eeg_id)
            return

        lat, lon = await weather.geocode_city(eeg.get("ort") or "Wien")
        readings = await db.get_hourly_readings(eeg_id, days_back=365)

        if len(readings) < 200:
            _status[eeg_id] = "insufficient_data"
            logger.info("EEG %s: insufficient data (%d rows)", eeg_id, len(readings))
            return

        # Fetch historical weather for the same window.
        min_ts = min(r["hour"] for r in readings)[:10]  # "YYYY-MM-DD"
        max_ts = datetime.utcnow().strftime("%Y-%m-%d")
        weather_hist = await weather.get_historical_weather(lat, lon, min_ts, max_ts)

        result = model.train(eeg_id, readings, weather_hist)
        _status[eeg_id] = result
        _cache.pop(eeg_id, None)  # invalidate forecast cache after retrain
    except Exception as exc:
        logger.exception("training failed for EEG %s: %s", eeg_id, exc)
        _status[eeg_id] = "error"


async def _weekly_retrain_loop() -> None:
    """Retrain all known EEG models once per week."""
    while True:
        await asyncio.sleep(7 * 24 * 3600)
        for eeg_id in list(_known_eegs):
            logger.info("weekly retrain for EEG %s", eeg_id)
            await _train(eeg_id)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await db.get_pool()
    asyncio.create_task(_weekly_retrain_loop())
    yield


app = FastAPI(lifespan=lifespan)


@app.get("/health")
async def health():
    return {"status": "ok"}


@app.post("/train/{eeg_id}")
async def trigger_training(eeg_id: str, background_tasks: BackgroundTasks):
    """Manually trigger model retraining for an EEG."""
    _known_eegs.add(eeg_id)
    background_tasks.add_task(_train, eeg_id)
    return {"status": "training_started"}


@app.get("/forecast/{eeg_id}")
async def get_forecast(eeg_id: str, background_tasks: BackgroundTasks):
    """Return 7-day hourly forecast for an EEG.

    On first call (no model yet) triggers training in background and returns
    status=training so the caller can poll again later.
    """
    _known_eegs.add(eeg_id)

    # Serve from cache if fresh (5-minute TTL).
    cached = _cache.get(eeg_id)
    if cached and (datetime.utcnow() - cached["at"]).total_seconds() < 300:
        return cached["data"]

    status = _status.get(eeg_id)

    if not model.has_model(eeg_id):
        if status == "training":
            return {"status": "training", "eeg_id": eeg_id, "entries": []}
        if status == "insufficient_data":
            return {"status": "insufficient_data", "eeg_id": eeg_id, "entries": []}
        # First request — kick off training.
        background_tasks.add_task(_train, eeg_id)
        return {"status": "training", "eeg_id": eeg_id, "entries": []}

    eeg = await db.get_eeg_info(eeg_id)
    if not eeg:
        raise HTTPException(status_code=404, detail="EEG not found")

    lat, lon = await weather.geocode_city(eeg.get("ort") or "Wien")
    forecast_weather = await weather.get_forecast_weather(lat, lon)
    entries = model.predict(eeg_id, forecast_weather)

    now = datetime.utcnow()
    result = {
        "status": "ok",
        "eeg_id": eeg_id,
        "generated_at": now.isoformat() + "Z",
        "entries": entries,
    }
    _cache[eeg_id] = {"data": result, "at": now}
    return result
