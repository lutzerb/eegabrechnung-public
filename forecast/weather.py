import httpx

GEOCODING_URL = "https://geocoding-api.open-meteo.com/v1/search"
FORECAST_URL = "https://api.open-meteo.com/v1/forecast"
HISTORICAL_URL = "https://archive-api.open-meteo.com/v1/archive"

_HOURLY_VARS = [
    "temperature_2m",
    "shortwave_radiation",
    "cloud_cover",
    "wind_speed_10m",
]

# Vienna fallback coordinates
_VIENNA = (48.2085, 16.3721)

_coord_cache: dict[str, tuple[float, float]] = {}


async def geocode_city(city_name: str) -> tuple[float, float]:
    """Returns (lat, lon) for an Austrian city name, falls back to Vienna."""
    if not city_name:
        return _VIENNA
    if city_name in _coord_cache:
        return _coord_cache[city_name]
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            r = await client.get(
                GEOCODING_URL,
                params={
                    "name": city_name,
                    "count": 1,
                    "language": "de",
                    "format": "json",
                    "countryCode": "AT",
                },
            )
            data = r.json()
        if data.get("results"):
            res = data["results"][0]
            coords = (res["latitude"], res["longitude"])
            _coord_cache[city_name] = coords
            return coords
    except Exception:
        pass
    return _VIENNA


async def get_historical_weather(lat: float, lon: float, date_from: str, date_to: str) -> list[dict]:
    """Fetches hourly historical weather data from Open-Meteo archive."""
    async with httpx.AsyncClient(timeout=60) as client:
        r = await client.get(
            HISTORICAL_URL,
            params={
                "latitude": lat,
                "longitude": lon,
                "start_date": date_from,
                "end_date": date_to,
                "hourly": ",".join(_HOURLY_VARS),
                "timezone": "Europe/Vienna",
            },
        )
        data = r.json()
    return _parse_hourly(data)


async def get_forecast_weather(lat: float, lon: float) -> list[dict]:
    """Fetches 7-day hourly weather forecast from Open-Meteo."""
    async with httpx.AsyncClient(timeout=30) as client:
        r = await client.get(
            FORECAST_URL,
            params={
                "latitude": lat,
                "longitude": lon,
                "hourly": ",".join(_HOURLY_VARS),
                "timezone": "Europe/Vienna",
                "forecast_days": 7,
            },
        )
        data = r.json()
    return _parse_hourly(data)


def _parse_hourly(data: dict) -> list[dict]:
    h = data.get("hourly", {})
    times = h.get("time", [])
    n = len(times)

    def col(key: str) -> list:
        vals = h.get(key, [None] * n)
        return [v if v is not None else 0.0 for v in vals]

    temp = col("temperature_2m")
    rad = col("shortwave_radiation")
    cloud = col("cloud_cover")
    wind = col("wind_speed_10m")

    return [
        {
            "ts": times[i],
            "temperature_2m": temp[i],
            "shortwave_radiation": rad[i],
            "cloud_cover": cloud[i],
            "wind_speed_10m": wind[i],
        }
        for i in range(n)
    ]
