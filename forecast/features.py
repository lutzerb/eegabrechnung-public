import numpy as np
import pandas as pd
import holidays

_AT_HOLIDAYS = holidays.Austria()

CONSUMPTION_FEATURES = [
    "hour",
    "day_of_week",
    "month",
    "day_of_year",
    "hour_sin",
    "hour_cos",
    "month_sin",
    "month_cos",
    "is_weekend",
    "is_holiday",
    "temperature_2m",
    "cloud_cover",
]

GENERATION_FEATURES = [
    "hour",
    "day_of_week",
    "month",
    "day_of_year",
    "hour_sin",
    "hour_cos",
    "month_sin",
    "month_cos",
    "is_weekend",
    "temperature_2m",
    "shortwave_radiation",
    "cloud_cover",
]


def make_features(df: pd.DataFrame) -> pd.DataFrame:
    """Add time and cyclical features to a DataFrame that has a 'ts' column."""
    df = df.copy()
    df["ts"] = pd.to_datetime(df["ts"])
    df["hour"] = df["ts"].dt.hour
    df["day_of_week"] = df["ts"].dt.dayofweek
    df["month"] = df["ts"].dt.month
    df["day_of_year"] = df["ts"].dt.dayofyear
    df["is_weekend"] = (df["day_of_week"] >= 5).astype(int)
    df["is_holiday"] = df["ts"].dt.date.apply(lambda d: int(d in _AT_HOLIDAYS))
    df["hour_sin"] = np.sin(2 * np.pi * df["hour"] / 24)
    df["hour_cos"] = np.cos(2 * np.pi * df["hour"] / 24)
    df["month_sin"] = np.sin(2 * np.pi * df["month"] / 12)
    df["month_cos"] = np.cos(2 * np.pi * df["month"] / 12)
    return df
