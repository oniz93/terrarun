# TerraRun — Project Map

> Auto-generated analysis of the repository at the time of writing.

## 1. What this is

**TerraRun** is a location-based running game blending **Strava-style run tracking** with **Ingress-style territory control**. Runners join one of two permanent factions — **Neon (red)** or **Umbra (blue)** — and compete for control of a real-world map tiled with **H3 hexagons**. Every run is a "territorial move": close a loop (or run a corridor) to capture hexes, flip enemy turf, earn points, and climb leaderboards.

| Part | Tech | Purpose | Size |
|---|---|---|---|
| `backend/` | Go 1.26 (chi, pgx, H3, gorilla/websocket) | REST API, game engine, bot scheduler, data loaders | ~9,500 lines / 82 files |
| `client/` | Flutter (Dart, flutter_bloc, flutter_map, H3) | iOS/Android/**web** app | ~1,500 lines / 24 files |
| `docs/` | Markdown | Design spec, HLD, LLD, task plan, seeding designs | 6 docs |

---

## 2. Repository layout

```
terrarun/
├── backend/               # Go monolith (API + CLI tools)
│   ├── cmd/
│   │   ├── api/           # live server (main.go wires everything)
│   │   ├── migrate/       # golang-migrate up/down
│   │   ├── osmloader/     # downloads & loads OSM/Natural Earth → PostGIS
│   │   ├── load_land/     # one-off JSON landmask loader
│   │   ├── load_water/    # one-off Overpass water loader
│   │   ├── migrate_latlng/# backfills lat/lng on hexes
│   │   └── seed_* / seed_burnaby/  # one-off territory seeders (v6 = latest)
│   ├── internal/          # domain packages (handler → service → repository)
│   ├── pkg/               # h3util, geo, mapbox, validator
│   ├── migrations/        # 14 up/down SQL migrations (001–014)
│   ├── osm_data/          # downloaded OSM/Natural Earth shapefiles
│   └── *.go at root       # ⚠️ leftover scratch/test files
├── client/                # Flutter app (lib/, ios/, android/, web/, test/)
├── docs/                  # design/session docs
├── docker-compose.yml        # dev stack
├── docker-compose.prod.yml   # prod stack
├── nginx.conf                # TLS + reverse proxy + WS passthrough
├── deploy.sh                 # single-VPS deploy script
├── Makefile                  # dev/test/lint/build/deploy targets
└── .env / .env.prod.example  # secrets (gitignored)
```

---

## 3. System architecture (4 layers)

```
CLIENT:  Flutter app (REST + WebSocket)
   │
GATEWAY: nginx → TLS (Let's Encrypt), rate-limit, WebSocket upgrade, /mapbox token proxy
   │
APP:     Go API (chi router)
         ├── middleware: RequestID → Recovery → Logger → CORS → Auth → RateLimit
         ├── 10 vertical-slice domain packages (handler/service/repo)
         ├── shared: H3 capture engine, cheat detection, points calc, bot scheduler
         └── WebSocket hub (gorilla/websocket) + in-process bot cron
   │
DATA:    PostgreSQL 16 (PostGIS + TimescaleDB) · Redis · rustfs (S3-compatible)
```

Request lifecycle: `client → nginx → chi middleware → handler (validate) → service (business logic) → repository (SQL/Redis) → Postgres/Redis/rustfs`.

---

## 4. Backend domain map

Every package follows the **handler → service → repository** triad:

| Package | Responsibilities |
|---|---|
| `auth` | Email/password (bcrypt), Google OAuth, Apple OAuth, JWT access + refresh-token rotation with reuse detection |
| `user` | Profile CRUD, avatar upload (rustfs), hashed contacts matching, GDPR export/soft-delete |
| `run` | Start/end runs, GPS ingestion, cheat detection, stats, orchestration of capture |
| `territory` | Capture engine (polygon/path → H3 cells → HP transitions), viewport queries, decay, seeding |
| `friend` | Add/accept/reject, feed |
| `leaderboard` | Team / runner / friends rankings (Redis-backed) |
| `season` | Current season, join, per-season leaderboard |
| `bot` | Onboarding seeding + active bot spawn/simulation (15-min scheduler) |
| `points` | Runner-point and XP/level calculator |
| `notification` | In-app notifications (DB + WebSocket push), Firebase FCM stub |
| `websocket` | Hub + client, JWT-authenticated, ping/pong, per-user send |
| `storage` | rustfs (S3-compatible) client via AWS SDK |
| `admin` | Stats, ban, cache-clear, migration trigger (admin-gated) |
| `config` | ~60 settings via envconfig |
| `middleware` | Auth (JWT), CORS, logging, recovery, request-ID, Redis rate limiting |
| `domain` | `User`, `Run`, `Hex`, `Bot`, `Friend`, `Season`, `Faction` types |

**`pkg/` shared libs:** `h3util` (lat/lng↔H3, polygon→cells, grid disks), `geo` (Haversine, Douglas-Peucker, polyline encoding, polygon/corridor ops, self-intersection repair), `mapbox` (Tilequery client), `validator`.

### API endpoints (wired in `cmd/api/main.go`)

- **Public:** `GET /health`, `GET /health/ready`, `GET /mapbox/{z}/{x}/{y}.mvt|.png`, `GET /ws?token=`
- **Auth (rate-limited):** `POST /auth/register|login|google|apple|refresh`
- **Authed:** `/users` (me, avatar, export, delete, contacts/match, `/{id}`), `/runs` (start, `{id}/end`, list, `{id}`, `{id}/gps`), `/territory` (`?sw_lat...` + `/stats`), `/friends` (add, `{id}/accept`, delete, list, pending, feed), `/leaderboard` (team/runners/friends), `/seasons` (current, list, join, `{id}/leaderboard`), `/admin/*`, `/notifications`

---

## 5. Database schema (14 migrations)

| Migration | Tables |
|---|---|
| `001_users` | `users` (email, faction, XP/level, tier, OAuth ids, soft-delete) + `pgcrypto` |
| `002_sessions` | `sessions` (refresh tokens) |
| `003_runs` | `runs` + `gps_points` (TimescaleDB hypertable) |
| `004_territory` | `hexes` (H3 index PK, owner, HP 1–10) + `territory_changes` |
| `005_friends` | `friends` (pending/accepted/blocked) |
| `006_seasons` | `seasons` + `season_participants` |
| `007_bots` | `bots` |
| `008_land_cache` | PostGIS `land_cache` (Mapbox-derived landmask, per H3 res-6 parent) |
| `009_water_cache` | `water_cache` |
| `010_osm_tables` | `global_landmask`, `osm_water`, `osm_forest`, `osm_protected`, `osm_mountain` |
| `011_osm_waterways` | `osm_waterways` (river centerlines) |
| `012_landmask_subdivided` | `global_landmask_subdivided` (ST_Subdivide) |
| `013_osm_ocean` | `osm_ocean` |
| `014_osm_urban` | `osm_urban` |

---

## 6. Core game mechanics (implemented)

- **Hex grid:** H3 resolution 11 (~50m edge). Only owned hexes are stored; neutral = absence of a row.
- **HP system:** neutral capture → 1 HP; same-team recapture → +1 (cap 10); enemy capture → −1; flip at 0; daily decay −1 (floor 1).
- **Capture geometry:** closed loop → polygon (Douglas-Peucker simplify → self-intersection repair); open run → 25m corridor buffer. Extracted via `h3.PolygonToCells`.
- **Points:** territory (10 base, +50% steal bonus), runner (1/100m, 5/10m elevation, 10% of territory, social multiplier capped at +50%), XP→level.
- **Cheat detection:** hard-reject (>35 km/h segment, >20 km/h avg, >500m GPS jump); soft-flag (>15 km/h avg, >50km, >30% low-accuracy, HR/pacing mismatch).
- **Bots:** onboarding seeder + 15-minute scheduler that balances bot count vs. real users and simulates runs that flip territory.

---

## 7. Client map (Flutter)

**Implemented:**
- `main.dart` — auth-gated app root (Bloc + MaterialApp)
- `core/api` — dio client with queued interceptor token-refresh
- `core/storage` — `flutter_secure_storage` token persistence
- `features/auth` — login only (email/password) + auth Bloc
- `features/map` — map + live H3 hex overlay, viewport fetching with 500ms debounce
- `features/run` — GPS tracking, start/end run, run Bloc, active/summary screens

**Not implemented (empty scaffold dirs):** leaderboard, social/friends, profile, stats; registration / faction-choice / onboarding; WebSocket client; health sync; social runs (`TODO` in code).

**Dependencies:** `dio`, `flutter_bloc`, `flutter_map` (map engine), `h3_flutter_plus`, `geolocator`, `flutter_background_geolocation`, `flutter_secure_storage`, `rxdart`, `latlong2`.

**Map engine:** the client uses `flutter_map` (pure Dart, works on iOS/Android/**web**) with OSM raster tiles by default. The original `mapbox_maps_flutter` SDK was Android/iOS-only and has been replaced so the whole app can run in a browser. (It is still in `pubspec.yaml` but no longer imported.)

---

## 8. Data / seeding pipeline

1. **`osmloader`** downloads Geofabrik British Columbia + Natural Earth 10m land shapefiles and loads them into PostGIS tables.
2. **`seed_vancouver_v6`** (latest) generates a 30km-radius territory field around Vancouver: res-9 parent classification, simplex-noise territory field, water/ocean exclusion, urban-only seeding, faction assignment with HP gradients.
3. **`migrate_latlng`** backfills `lat`/`lng` on existing hexes.
4. Older seeders (`v4`, `v5`, `seed_burnaby`, `seed_vancouver_full`) are historical iterations.

The world was seeded with ~1,000,000 hexes at 40% density, with land-aware exclusion of the Fraser River, Burrard Inlet, and False Creek.

---

## 9. Infrastructure / deploy

- **Dev:** `docker compose up` → Postgres (TimescaleDB image), Redis, rustfs, API.
- **Prod:** `docker-compose.prod.yml` adds nginx (TLS + WS + `/mapbox` proxy), certbot renewal daemon, one-shot `migrate` service; `deploy.sh` orchestrates the release.
- **Makefile:** `dev`, `test`, `lint`, `build`, `migrate-up/down`, `deploy`, `logs`, `clean`.

---

## 10. Current-state assessment

✅ **Working:**
- Backend API builds (`go build ./cmd/api`) and tests pass (`go test ./internal/... ./pkg/...`).
- Full auth flow, friend/leaderboard/season endpoints, capture geometry, cheat detection, bot scheduler, land-aware seeding implemented.

⚠️ **Gaps, drift, and bugs:**

1. **Critical — real-run territory capture is effectively broken.** `domain.Run.Faction()` is a stub returning `nil`; the `Run` struct has no faction field, and the run service never resolves the runner's faction before calling `CaptureRun`. `userFaction` therefore stays `""`, violating `hexes.owned_by CHECK (IN ('neon','umbra'))`; capture errors and is silently swallowed in `EndRun` ("territory capture failed"), so runs complete but capture 0 hexes. Bots/seeding write factions directly and are unaffected.
2. **Schema drift from migrations:** code uses `notifications` (notification service), `users.is_admin` + `users.is_banned` (admin handler), and `hexes.lat`/`hexes.lng` (repository) — none exist in any migration. Added ad-hoc to the live DB.
3. **`go build ./...` fails** because of leftover scratch files at `backend/` root: `seed_burnaby.go` (syntax errors), plus `test.go`, `test_h3_valid.go`, `test_h3_van.go`.
4. **Apple OAuth is not actually verified** — `verifyAppleIdentityToken` parses the JWT unverified (no signature/key check). Google uses the `tokeninfo` endpoint (not full JWKS verification).
5. **Bot logic is simplified/hardcoded vs. the HLD:** spawns around hardcoded London coordinates (51.5074, −0.1278); `SimulateBotRun` writes a bogus `hp_after = 90`; no plausible street routing or peak-hour weighting.
6. **Client is ~40% of the intended app:** only login + map + run are built.

---

## 11. Running it

```bash
# 1. Backend (dev stack: postgres, redis, rustfs, api on :8080)
docker compose up --build

# 2. Run migrations (in a second terminal)
make migrate-up

# Backend tests
cd backend && go test ./internal/... ./pkg/...

# 3a. Flutter on WEB (browser, targets http://localhost:8080 by default)
cd client && flutter run -d chrome

# 3b. Or build web and serve it manually
cd client && flutter build web
cd build/web && python3 -m http.server 8000   # open http://localhost:8000

# 3c. Flutter on a physical device (override the API URL to your Mac's LAN IP)
cd client && flutter run --dart-define=API_BASE_URL=http://<mac-lan-ip>:8080
```

**Web notes:**
- The web client defaults `API_BASE_URL` to `http://localhost:8080`.
- The backend already sends `Access-Control-Allow-Origin: *`, so the browser client can reach it.
- Browser geolocation works on `localhost` (a secure context); the browser will ask for location permission.
- A seeded database is required to see territory hexes (`osmloader` + `seed_vancouver_v6`); an empty DB will still let you log in and view the map.
