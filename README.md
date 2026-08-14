# TerraRun

A location-based running game that blends **Strava-style run tracking** with
**Ingress-style territory control**. Runners join one of two factions — **Neon**
(red) or **Umbra** (blue) — and compete for control of a real-world map tiled
with **H3 hexagons**.

## Repository layout

| Path | Description |
|---|---|
| `backend/` | Go API + game engine + data-loading CLIs (chi, pgx, PostGIS/Timescale, Redis, H3) |
| `client/` | Flutter app (iOS / Android / **web**) using flutter_map, H3, flutter_bloc |
| `docs/` | Design spec, HLD, LLD, task plan, and hybrid seeding design |

For a detailed map of everything in this repo, see
[`PROJECT_MAP.md`](./PROJECT_MAP.md).

## Quick start

```bash
# 1. Start the backend stack (Postgres+PostGIS/Timescale, Redis, rustfs, API)
docker compose up --build

# 2. Run database migrations (second terminal)
make migrate-up

# 3a. Run the Flutter client in a browser (targets http://localhost:8080)
cd client && flutter run -d chrome

# 3b. Or build the web client and serve it
cd client && flutter build web
cd build/web && python3 -m http.server 8000   # open http://localhost:8000

# 3c. Run on a physical device (point at your Mac's LAN IP)
cd client && flutter run --dart-define=API_BASE_URL=http://<mac-lan-ip>:8080
```

A seeded database is required to see territory hexes. Load OSM data and seed
the Vancouver area with:

```bash
cd backend
go run ./cmd/osmloader landmask
go run ./cmd/osmloader british-columbia
go run ./cmd/seed_vancouver_v6
```

## Testing

```bash
make test       # backend tests (go test ./...)
cd client && flutter analyze
```

## Map tiles

The client uses **OpenStreetMap raster tiles by default** (free/unmetered) to
keep Mapbox usage to a minimum. Mapbox can be opted into with:

```bash
cd client && flutter run -d chrome   --dart-define=TILE_URL_TEMPLATE='https://api.mapbox.com/styles/v1/mapbox/dark-v11/tiles/{z}/{x}/{y}?access_token=YOUR_TOKEN'
```
