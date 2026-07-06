import os
import logging

import joblib
import pandas as pd
from xgboost import XGBRegressor

from features import CONSUMPTION_FEATURES, GENERATION_FEATURES, make_features

logger = logging.getLogger("forecast.model")

MODELS_DIR = os.getenv("MODELS_DIR", "/app/models")

# In-memory model cache — invalidated after retraining.
_cache: dict[str, XGBRegressor] = {}


def model_path(eeg_id: str, kind: str) -> str:
    return os.path.join(MODELS_DIR, f"{eeg_id}_{kind}.pkl")


def _get(eeg_id: str, kind: str) -> XGBRegressor | None:
    key = f"{eeg_id}_{kind}"
    if key not in _cache:
        path = model_path(eeg_id, kind)
        if os.path.exists(path):
            _cache[key] = joblib.load(path)
        else:
            return None
    return _cache[key]


def invalidate(eeg_id: str) -> None:
    for kind in ("consumption", "generation"):
        _cache.pop(f"{eeg_id}_{kind}", None)


def has_model(eeg_id: str) -> bool:
    for kind in ("consumption", "generation"):
        if os.path.exists(model_path(eeg_id, kind)):
            return True
    return False


def train(eeg_id: str, readings: list[dict], weather_hist: list[dict]) -> str:
    """Train XGBoost models for consumption and generation.

    Returns 'ok' | 'insufficient_data'.
    """
    os.makedirs(MODELS_DIR, exist_ok=True)

    df_r = pd.DataFrame(readings).rename(columns={"hour": "ts"})
    df_w = pd.DataFrame(weather_hist)

    df = df_r.merge(df_w, on="ts", how="left")
    df = make_features(df).fillna(0)

    if len(df) < 200:
        return "insufficient_data"

    _xgb_params = dict(
        n_estimators=300,
        max_depth=6,
        learning_rate=0.05,
        subsample=0.8,
        colsample_bytree=0.8,
        random_state=42,
        tree_method="hist",
        verbosity=0,
    )

    trained_any = False
    for target, features, kind in [
        ("consumption_kwh", CONSUMPTION_FEATURES, "consumption"),
        ("generation_kwh", GENERATION_FEATURES, "generation"),
    ]:
        y = df[target].clip(lower=0)
        if y.sum() < 1.0:
            # No meaningful data for this direction (e.g. EEG without PV).
            continue
        X = df[features].fillna(0)
        mdl = XGBRegressor(**_xgb_params)
        mdl.fit(X, y)
        joblib.dump(mdl, model_path(eeg_id, kind))
        logger.info("trained %s model for eeg %s (%d rows)", kind, eeg_id, len(df))
        trained_any = True

    invalidate(eeg_id)
    return "ok" if trained_any else "insufficient_data"


def predict(eeg_id: str, forecast_weather: list[dict]) -> list[dict]:
    """Generate hourly forecast from weather data. Returns [] if no model exists."""
    df = pd.DataFrame(forecast_weather)
    df = make_features(df).fillna(0)

    consumption_model = _get(eeg_id, "consumption")
    generation_model = _get(eeg_id, "generation")

    if consumption_model is None and generation_model is None:
        return []

    entries = []
    for _, row in df.iterrows():
        entry: dict = {"ts": row["ts"].isoformat()}

        if consumption_model is not None:
            X = row[CONSUMPTION_FEATURES].values.reshape(1, -1)
            entry["consumption_kwh"] = max(0.0, float(consumption_model.predict(X)[0]))
        else:
            entry["consumption_kwh"] = None

        if generation_model is not None:
            X = row[GENERATION_FEATURES].values.reshape(1, -1)
            entry["generation_kwh"] = max(0.0, float(generation_model.predict(X)[0]))
        else:
            entry["generation_kwh"] = None

        entries.append(entry)

    return entries
