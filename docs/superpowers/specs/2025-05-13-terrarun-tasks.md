# TerraRun — Implementation Plan

**Date:** 2025-05-13
**Depends on:** [LLD](./2025-05-13-terrarun-lld.md) → [HLD](./2025-05-13-terrarun-hld.md) → [Design Spec](./2025-05-13-terrarun-design.md)

---

## Legend

- **ID** — sequential task identifier
- **Depends** — IDs of tasks that must complete before this one starts
- **Effort** — estimated hours (h) or days (d)
- **Owner** — BE (backend), FE (Flutter), OPS (infra), DEV (both)
- Each task is atomic: completable in a single session with a clear deliverable

---

## Phase 0 — Environment & Scaffolding

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T001 | Create repo directory structure: `backend/`, `client/`, `docs/` | — | 0.5h | DEV |
| T002 | Create `backend/go.mod` with module path `github.com/terrarun/backend` and go version | — | 0.5h | BE |
| T003 | Create `backend/internal/` and `backend/cmd/` subdirectory layout with empty `.go` placeholder files matching LLD structure | T002 | 1h | BE |
| T004 | Create `client/` Flutter project via `flutter create` | — | 0.5h | FE |
| T005 | Create `client/lib/` directory layout matching LLD (features/, core/, shared/) | T004 | 1h | FE |
| T006 | Write `docker-compose.yml` with PostgreSQL+TimescaleDB, Redis, rustfs services | — | 1h | OPS |
| T007 | Write `Dockerfile` for Go API (multi-stage: build + alpine run) | — | 1h | OPS |
| T008 | Write `Makefile` with targets: dev, test, lint, build, deploy, migrate-up, logs | — | 1h | OPS |
| T009 | Write `.env.example` with all config keys and placeholder values | — | 0.5h | BE |
| T010 | Write `.gitignore` for Go (bin/, vendor/) + Flutter (build/, .dart_tool/) + Docker volumes | — | 0.5h | DEV |
| T011 | Verify `docker compose up` brings up all services healthy | T006, T007 | 0.5h | OPS |
| T012 | Add `pre-commit` config: gofmt, goimports, golangci-lint, flutter format | — | 1h | DEV |

---

## Phase 1 — Foundation

### 1.1 Configuration & CLI

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T013 | Implement `internal/config/config.go`: `Config` struct with all fields via `envconfig` | T002 | 2h | BE |
| T014 | Implement `cmd/api/main.go`: load config, connect DB, connect Redis, start chi router, graceful shutdown | T013 | 2h | BE |
| T015 | Implement health check endpoints: `GET /health` (db+redis) and `GET /health/ready` (ready probe) | T014 | 1h | BE |
| T016 | Implement structured logging: zerolog with request ID, level, timestamp, JSON format | — | 1h | BE |
| T017 | Implement `middleware/recovery.go`: panic recovery middleware writing stack trace to logs | — | 0.5h | BE |
| T018 | Implement `middleware/logging.go`: request/response logger (method, path, status, duration) | T016 | 1h | BE |
| T019 | Implement `middleware/requestid.go`: inject/forward X-Request-ID header | — | 0.5h | BE |
| T020 | Wire global middleware chain: RequestID → Recovery → Logger → Timeout(30s) → CORS | T017, T018, T019 | 0.5h | BE |

### 1.2 Database Migrations

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T021 | Write migration M001: `users` table with all columns, constraints, indexes (per LLD schema) | — | 1h | BE |
| T022 | Write migration M002: `sessions` table (refresh tokens) | T021 | 0.5h | BE |
| T023 | Write migration M003: `runs` table + `gps_points` hypertable (TimescaleDB) | T021 | 1.5h | BE |
| T024 | Write migration M004: `hexes` table + `territory_changes` table | T021 | 1h | BE |
| T025 | Write migration M005: `friends` table | T021 | 0.5h | BE |
| T026 | Write migration M006: `seasons` table + `season_participants` table | T021 | 0.5h | BE |
| T027 | Write migration M007: `bots` table | T021 | 0.5h | BE |
| T028 | Write `cmd/migrate/main.go`: run golang-migrate Up() against DATABASE_URL | T021-T027 | 1h | BE |
| T029 | Verify all migrations run successfully against fresh PostgreSQL+TimescaleDB | T028, T006 | 0.5h | BE |

### 1.3 Domain Types

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T030 | Define `internal/domain/user.go`: User, Faction, RunnerTier types + constants | — | 0.5h | BE |
| T031 | Define `internal/domain/run.go`: Run, RunStatus, CaptureMode, GPSPoint, HealthSource | — | 0.5h | BE |
| T032 | Define `internal/domain/territory.go`: Hex, TerritoryChange, TerritoryStats | — | 0.5h | BE |
| T033 | Define `internal/domain/friend.go`: Friend, FriendStatus | — | 0.5h | BE |
| T034 | Define `internal/domain/season.go`: Season, SeasonParticipant, TiersConfig | — | 0.5h | BE |
| T035 | Define `internal/domain/bot.go`: Bot | — | 0.5h | BE |

### 1.4 Auth Module — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T036 | Implement `internal/auth/repository.go`: interface (CreateSession, GetByRefreshToken, Revoke, RevokeAllForUser) | T021, T022 | 1h | BE |
| T037 | Implement `internal/auth/repository_postgres.go`: all SQL queries for sessions table | T036 | 2h | BE |
| T038 | Implement `internal/auth/jwt.go`: generateAccessToken (HS256), parseAndValidateJWT, extractBearerToken | — | 2h | BE |
| T039 | Implement `internal/auth/service.go`: `Register()` method — validate email uniqueness, hash password (bcrypt cost 12), create user via userRepo, create session, return tokens | T037, T038 | 2h | BE |
| T040 | Implement auth service `Login()` — lookup by email, compare bcrypt hash, block OAuth-only accounts | T039 | 1.5h | BE |
| T041 | Implement auth service `GoogleLogin()` — verify Google ID token via Google API, find-or-create user | T039 | 2h | BE |
| T042 | Implement auth service `AppleLogin()` — verify Apple identity token, find-or-create user | T039 | 2h | BE |
| T043 | Implement auth service `Refresh()` — validate refresh token, detect reuse (revoke all on reuse), rotate token | T039 | 2h | BE |
| T044 | Implement `internal/auth/handler.go`: `Register` handler (validate input, call service, respond) | T039 | 1h | BE |
| T045 | Implement auth handler: `Login` handler | T040 | 0.5h | BE |
| T046 | Implement auth handler: `GoogleLogin` handler | T041 | 0.5h | BE |
| T047 | Implement auth handler: `AppleLogin` handler | T042 | 0.5h | BE |
| T048 | Implement auth handler: `Refresh` handler | T043 | 0.5h | BE |
| T049 | Implement `middleware/auth.go`: JWT validation middleware — extract token, parse claims, inject userID/faction/tier into context | T038 | 2h | BE |
| T050 | Register auth routes in router: `/auth/register`, `/auth/login`, `/auth/google`, `/auth/apple`, `/auth/refresh` (all public) | T044-T048 | 0.5h | BE |
| T051 | Implement password validation: min 8 chars, min 1 letter, min 1 number | — | 0.5h | BE |
| T052 | Implement rate limiting: `/auth/register` 5/h, `/auth/login` 20/h, `/auth/refresh` 10/min via Redis sliding window | — | 2h | BE |
| T053 | Implement `middleware/ratelimit.go`: per-route rate limit middleware reading from Redis | T052 | 2h | BE |
| T054 | Wire rate limit middleware on auth routes | T053, T050 | 0.5h | BE |

### 1.5 User Profile Module — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T055 | Implement `internal/user/repository.go`: interface (GetByID, GetByEmail, GetByGoogleID, GetByAppleID, GetByPhoneHash, Create, Update, SoftDelete, Export) | T021 | 1h | BE |
| T056 | Implement `internal/user/repository_postgres.go`: all SQL queries for users table | T055 | 2h | BE |
| T057 | Implement `internal/user/service.go`: `GetMe()` — fetch own profile with computed stats (total runs, distance, points) | T056 | 1.5h | BE |
| T058 | Implement user service: `UpdateMe()` — update display name, phone hash, avatar | T056 | 1h | BE |
| T059 | Implement user service: `GetPublic()` — restricted view (no email, no phone) | T056 | 0.5h | BE |
| T060 | Implement user service: `MatchContacts()` — match hashed phone numbers, return matches + faction counts | T056 | 2h | BE |
| T061 | Implement `internal/user/handler.go`: `GetMe`, `UpdateMe`, `GetPublic`, `MatchContacts` handlers | T057-T060 | 2h | BE |
| T062 | Implement avatar upload: accept multipart file, validate size (<5MB) and type (JPEG/PNG), upload to rustfs, store URL | — | 2h | BE |
| T063 | Implement `internal/storage/rustfs.go`: S3-compatible client for upload/download/delete via AWS SDK v2 with rustfs endpoint | — | 2h | BE |
| T064 | Register user routes under `/users/` with auth middleware | T061 | 0.5h | BE |

### 1.6 Auth Module — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T065 | Implement `core/api/api_client.dart`: Dio HTTP client with base URL, JSON serialization, auth interceptor (auto-attach Bearer token), 401 handler (auto-refresh) | — | 3h | FE |
| T066 | Implement `core/storage/secure_storage.dart`: flutter_secure_storage wrapper for access_token, refresh_token persistence | — | 1.5h | FE |
| T067 | Implement `core/models/user.dart`: User model with fromJson/toJson | — | 1h | FE |
| T068 | Implement `features/auth/bloc/auth_bloc.dart`: auth state (unauthenticated, loading, authenticated, error), events (LoginRequested, RegisterRequested, GoogleLoginRequested, AppleLoginRequested, LogoutRequested) | T065, T066 | 4h | FE |
| T069 | Implement `features/auth/screens/login_screen.dart`: email field, password field, login button, "Register" link, "Sign in with Google" button, "Sign in with Apple" button | T068 | 3h | FE |
| T070 | Implement `features/auth/screens/register_screen.dart`: email, password (with requirements hint), display name, register button | T068 | 2h | FE |
| T071 | Implement Google Sign-In integration: `google_sign_in` plugin, get idToken, send to backend | T068 | 2h | FE |
| T072 | Implement Apple Sign-In integration: `sign_in_with_apple` plugin, get identity token, send to backend | T068 | 2h | FE |
| T073 | Implement `splash_screen.dart`: check stored token, attempt refresh, route to /home or /auth/login | T066, T068 | 2h | FE |
| T074 | Implement auto-logout on 401 with failed refresh (clear secure storage, navigate to login) | T065, T068 | 1h | FE |

### 1.7 Profile & Onboarding — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T075 | Implement `features/auth/screens/faction_choice_screen.dart`: show Neon (red) vs Umbra (blue) cards with lore text, faction description, permanent choice confirmation dialog | T068 | 3h | FE |
| T076 | Implement contacts scan: request permission, hash phone numbers client-side with SHA-256, send to server, display faction counts | T060 | 3h | FE |
| T077 | Implement faction friends detail: tap a faction count → show list of matched friends with names and factions | T076 | 2h | FE |
| T078 | Implement `features/profile/screens/profile_screen.dart`: avatar, display name, faction badge, account level + XP bar, stats (total runs, distance, territory points), tier badge | T068 | 4h | FE |
| T079 | Implement `features/profile/screens/edit_profile_screen.dart`: edit display name, upload avatar (image picker + crop), optional phone number entry with hash-and-send | T062, T078 | 3h | FE |
| T080 | Implement 3-screen onboarding tutorial: (1) "Run to capture", (2) "Stats & history", (3) "Teams & territory" — swipeable pages with illustrations, "Get Started" on last page → navigate to /home | — | 4h | FE |
| T081 | Implement `app.dart`: MaterialApp with theme (Neon red + Umbra blue palette, dark map-friendly background), route generation | T074, T075, T080 | 2h | FE |
| T082 | Implement `config/theme.dart`: light + dark theme, faction color constants, typography, card styles | — | 2h | FE |

---

## Phase 2 — Core Gameplay

### 2.1 Run Module — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T083 | Implement `internal/run/repository.go`: interface (Create, Update, GetByID, GetByUser with pagination, InsertGPSPoints, GetGPSPoints) | T023, T031 | 1h | BE |
| T084 | Implement `internal/run/repository_postgres.go`: all SQL queries for runs + gps_points tables, batch GPS insert via pgx CopyFrom | T083 | 4h | BE |
| T085 | Implement `internal/run/stats.go`: StatsCalculator — compute distance (haversine cumulative), duration, avg pace (s/km), max speed, elevation gain, encode polyline (Google format) | — | 3h | BE |
| T086 | Implement `internal/run/cheat.go`: CheatDetector — all 5 checks (segment speed >35km/h, avg speed >20km/h for >2km, GPS jump >500m, soft flags for >15km/h flat, >50km distance, low accuracy >30%, HR corroboration) | T085 | 4h | BE |
| T087 | Implement `internal/run/service.go`: `StartRun()` — validate inputs, create run record (status=active), validate social participants (friends + within 20m) | T084 | 3h | BE |
| T088 | Implement run service: `EndRun()` — cheat check → stats compute → territory capture (delegated) → points compute → update run → insert GPS points → broadcast → notify | T087 | 4h | BE |
| T089 | Implement run service: `GetRun()` — single run with full detail | T084 | 1h | BE |
| T090 | Implement run service: `ListRuns()` — paginated runs for a user, filterable by status | T084 | 1.5h | BE |
| T091 | Implement run service: `GetGPSPoints()` — return raw or simplified GPS track for a run | T084 | 1h | BE |
| T092 | Implement `internal/run/handler.go`: `StartRun`, `EndRun`, `GetRun`, `ListRuns`, `GetGPSPoints` handlers with validation | T087-T091 | 3h | BE |
| T093 | Register run routes: `POST /runs/start`, `POST /runs/:id/end`, `GET /runs`, `GET /runs/:id`, `GET /runs/:id/gps` | T092 | 0.5h | BE |

### 2.2 Territory Module — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T094 | Add H3 library dependency: `github.com/uber/h3-go/v4` | — | 0.5h | BE |
| T095 | Implement `pkg/h3util/h3.go`: utility functions — LatLngToCell(res=11), CellToLatLng, PolygonToCells, GridDisk, CellsToMultiPolygon, isCellInPolygon | T094 | 3h | BE |
| T096 | Implement `pkg/geo/geo.go`: Haversine distance, bounding box ops, Douglas-Peucker simplification, polygon validation (self-intersection detection, largest simple sub-polygon extraction) | — | 4h | BE |
| T097 | Implement `internal/territory/repository.go`: interface (GetHex, UpsertHex, GetHexesInBounds, BatchUpsertHexes, BatchDecayHP, RecordChange, GetRecentChanges, GetRegionStats) | T024, T032 | 1.5h | BE |
| T098 | Implement `internal/territory/repository_postgres.go`: all SQL queries — single hex CRUD, bounding box query (limit 10k), batch upsert via pgx CopyFrom, decay query | T097 | 4h | BE |
| T099 | Implement `internal/territory/hp.go`: HP arithmetic — AddHP (cap at 10), RemoveHP (flip at 0), DailyDecay (-1, floor at 1) as pure functions | T032 | 1h | BE |
| T100 | Implement `internal/territory/capture.go`: `determineCaptureGeometry()` — branch on loop closure distance vs config, build polygon or path corridor | T095, T096, T013 | 3h | BE |
| T101 | Implement territory capture: `CaptureRun()` — full algorithm per LLD Section 4.1: extract H3 cells from geometry, apply HP changes per hex in transaction, record territory_changes, compute territory points, return CaptureResult | T100, T099, T098 | 4h | BE |
| T102 | Implement `internal/territory/service.go`: `GetViewportHexes()` — query hexes in bbox, return hex list with owner+HP | T098 | 1.5h | BE |
| T103 | Implement territory service: `GetRegionStats()` — aggregate counts per region (country/city/bbox), compute percentages | T098 | 2h | BE |
| T104 | Implement `internal/territory/handler.go`: `GetViewportHexes`, `GetRegionStats` handlers | T102, T103 | 1.5h | BE |
| T105 | Implement `GET /territory` endpoint: accept bbox query params (sw_lat, sw_lng, ne_lat, ne_lng), return hexes with owner+HP, cap at 10k results | T104, T098 | 0.5h | BE |
| T106 | Implement `GET /territory/stats` endpoint: accept region_type + region_value, return TerritoryStats JSON | T104, T103 | 0.5h | BE |
| T107 | Implement `internal/territory/decay.go`: `DecayAllHexes()` — iterate hexes where hp>1, decrement hp, record in territory_changes. Batch size 10k per transaction. | T099, T098 | 2h | BE |
| T108 | Implement runtime config toggle: check Redis key `config:path_capture_enabled` at capture time, fallback to env var | T100, T013 | 1h | BE |
| T109 | Implement runtime config toggle: check Redis key `config:path_capture_buffer` for custom buffer radius | T100 | 0.5h | BE |

### 2.3 Points & Levels — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T110 | Implement `internal/points/calculator.go`: `RunnerPoints()` — distance/100 + elevation/10 + territory*0.1, social multiplier (base 0.25 + 0.10/friend, cap 0.50) | — | 2h | BE |
| T111 | Implement points: `XPForLevel(level)` — exponential curve: 100 * level^1.5 | — | 0.5h | BE |
| T112 | Implement points: `LevelFromXP(totalXP)` — invert XPForLevel to find current level | T111 | 0.5h | BE |
| T113 | Implement `UpdateUserXP()` — called after run completion: add runner points * XP_CONVERSION_RATIO, recompute account level, update user row | T110-T112 | 1.5h | BE |

### 2.4 Map Screen — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T114 | Install Mapbox GL Flutter plugin, configure iOS and Android with proxy URL pattern | T004 | 2h | FE |
| T115 | Implement `core/models/hex.dart`: Hex model with h3Index, centerLat, centerLng, ownedBy, hp | T105 | 1h | FE |
| T116 | Implement `features/map/bloc/map_bloc.dart`: MapState (hexes map, bbox, active run, status), events (MapLoaded, MapMoved, RunStarted, RunEnded, TerritoryUpdated) | T115 | 3h | FE |
| T117 | Implement map bloc: `_onMapLoaded` — fetch hexes for initial viewport from API | T116 | 1h | FE |
| T118 | Implement map bloc: `_onMapMoved` — debounce 300ms, fetch hexes for new viewport, resubscribe WebSocket to new bbox | T117 | 2h | FE |
| T119 | Implement `features/map/screens/map_screen.dart`: Mapbox map widget, hex overlay, run control button (bottom center), FAB column (leaderboard + friends), current location button | T116, T114 | 5h | FE |
| T120 | Implement `features/map/widgets/hex_overlay.dart`: CustomPainter that renders hexes as colored polygons projected to screen, opacity based on HP/maxHP | T115 | 4h | FE |
| T121 | Implement hex overlay: color mapping — Neon = red (#FF3B30), Umbra = blue (#007AFF), opacity = 0.1 + (hp/10)*0.6 (range 0.17 to 0.70) | T120 | 1h | FE |
| T122 | Implement hex overlay: stroke/border for each hex (0.5px, white at 12% opacity) for visual separation | T121 | 0.5h | FE |
| T123 | Implement hex overlay performance: only render hexes visible in current viewport (calculate screen bounds), clip to viewport | T120 | 2h | FE |
| T124 | Implement `features/map/widgets/run_controls.dart`: large circular "Start Run" button (pulsing animation when inactive), swipe-up-to-confirm gesture to prevent accidental starts | — | 2h | FE |
| T125 | Implement run controls: "Start Social Run" sub-button — opens friend selector to pick participants (max 10), shows which friends are nearby | T124 | 2h | FE |
| T126 | Implement run controls: during run — show elapsed time, distance, current pace, "End Run" button with swipe-to-confirm | T124 | 3h | FE |
| T127 | Implement territory stats overlay: collapsible bottom sheet showing faction percentages in current region (Neon X% / Umbra Y%) | T106 | 2h | FE |
| T128 | Integrate WebSocket connection on map screen: connect on enter, disconnect on leave, reconnect on failure with exponential backoff | T116 | 3h | FE |
| T129 | Implement WebSocket territory subscription: send subscribe message with current bbox, handle territory:changed events to update hex map in real-time | T128, T118 | 2h | FE |

### 2.5 Run Tracking — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T130 | Implement `features/run/services/gps_service.dart`: GPS point collection via geolocator, start/stop, distance filter (5m), speed/accuracy tracking | — | 3h | FE |
| T131 | Implement background GPS: integrate `flutter_background_geolocation`, config (stopOnTerminate=false, startOnBoot=false, desiredAccuracy=high) | T130 | 3h | FE |
| T132 | Implement battery optimization: reduce GPS polling to every 30s when stationary (>2 min no movement), warn user if battery saver active | T131 | 2h | FE |
| T133 | Implement `features/run/bloc/run_bloc.dart`: RunState (idle, active, ending, completed), events (StartRun, UpdatePosition, EndRun) | T130 | 3h | FE |
| T134 | Implement run bloc: `_onStartRun` — generate run ID, POST /runs/start, begin GPS tracking, start timer | T133, T092 | 2h | FE |
| T135 | Implement run bloc: `_onUpdatePosition` — collect GPS points, compute distance so far, emit updated state with live stats | T133 | 2h | FE |
| T136 | Implement run bloc: `_onEndRun` — stop tracking, POST /runs/:id/end with GPS points array, receive summary, navigate to summary screen | T133, T092 | 2h | FE |
| T137 | Implement `features/run/screens/run_active_screen.dart`: full-screen run view — large pace display, distance, elapsed time, map with live trace, territory capture estimation ("you are capturing ~X hexes") | T133, T120 | 5h | FE |
| T138 | Implement run active screen: visual feedback for territory capture — hexes along route highlight in team color as run progresses (client-side estimation) | T137 | 2h | FE |
| T139 | Implement live position broadcast via WebSocket: send `run:position` every 5 seconds to server (relayed to social participants) | T128, T137 | 1.5h | FE |

### 2.6 Run Summary & History — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T140 | Implement `features/run/screens/run_summary_screen.dart`: post-run stats display — distance, duration, avg pace, elevation, calories, route map (static), territory captured count (own + stolen), points earned breakdown, cheat status badge | T136 | 5h | FE |
| T141 | Implement pace chart: `fl_chart` line chart showing pace (s/km) over distance, with avg pace reference line | T140 | 3h | FE |
| T142 | Implement elevation profile: area chart showing elevation (m) over distance, net gain label | T140 | 2h | FE |
| T143 | Implement split times table: per-km splits with pace for each km, fastest km highlighted | T140 | 2h | FE |
| T144 | Implement territory capture summary: hex count (captured + stolen), team points earned, animated counter from 0 to final value | T140 | 2h | FE |
| T145 | Implement share functionality: generate run card image (map + stats) for sharing to social media / messaging | T140 | 3h | FE |
| T146 | Implement `features/run/screens/run_history_screen.dart`: scrollable list of past runs — each card shows date, distance, pace, territory hexes, closed-loop badge or path-capture badge | T140 | 3h | FE |
| T147 | Implement `features/run/screens/run_detail_screen.dart`: full details of a past run — all stats, interactive map trace, pace chart, elevation profile, territory changes (hexes affected) | T146 | 3h | FE |
| T148 | Implement run history filtering: by status (completed/flagged), by date range, by distance range | T146 | 2h | FE |
| T149 | Implement GPX export button on run detail (Runner tier only): download GPX file via share sheet | T147 | 2h | FE |

---

## Phase 3 — Social & Engagement

### 3.1 Friend System — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T150 | Implement `internal/friend/repository.go`: interface (AddRequest, Accept, Reject, Block, GetFriends, GetPending, GetByUserPair, GetFriendIDs) | T025, T033 | 1h | BE |
| T151 | Implement `internal/friend/repository_postgres.go`: all SQL queries for friends table | T150 | 2h | BE |
| T152 | Implement `internal/friend/service.go`: `AddFriend()` — validate not already friends/blocked, not self, insert pending request | T151 | 2h | BE |
| T153 | Implement friend service: `AcceptFriend()` — validate pending request exists, update status to accepted | T152 | 1h | BE |
| T154 | Implement friend service: `RejectFriend()` — delete pending request | T152 | 0.5h | BE |
| T155 | Implement friend service: `GetFriends()` — list accepted friends with last run summary (distance, date, territory points) | T151 | 2h | BE |
| T156 | Implement friend service: `GetPending()` — list incoming friend requests | T151 | 1h | BE |
| T157 | Implement friend service: `GetFeed()` — paginated list of friends' recent completed runs, ordered by date desc | T151 | 2h | BE |
| T158 | Implement `internal/friend/handler.go`: `AddFriend`, `AcceptFriend`, `RejectFriend`, `GetFriends`, `GetPending`, `GetFeed` handlers | T152-T157 | 2h | BE |
| T159 | Register friend routes: `POST /friends/add`, `POST /friends/:id/accept`, `DELETE /friends/:id`, `GET /friends`, `GET /friends/pending`, `GET /friends/feed` | T158 | 0.5h | BE |

### 3.2 Notifications — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T160 | Implement `internal/notification/firebase.go`: Firebase Admin SDK init from credentials file, send push notification helper (Android + iOS), handle token registration | — | 3h | BE |
| T161 | Implement `internal/notification/service.go`: dispatch notification by userID — resolve FCM token from Redis, call Firebase send | T160 | 2h | BE |
| T162 | Implement notification triggers: friend request received → notify target | T161, T152 | 1h | BE |
| T163 | Implement notification triggers: friend request accepted → notify requester | T161, T153 | 0.5h | BE |
| T164 | Implement notification triggers: friend starts run → notify all friends ("X started a run — join them!") | T161, T087 | 1h | BE |
| T165 | Implement notification triggers: friend completes run → notify all friends ("X ran Y km, captured N hexes") | T161, T088 | 1h | BE |
| T166 | Implement notification triggers: territory under attack → notify users whose hexes were stolen | T161, T101 | 1.5h | BE |
| T167 | Implement notification triggers: season ending soon (3 days before) → notify all active users | T161 | 1h | BE |
| T168 | Implement notification triggers: season ended → notify each user with their final tier and badge | T161 | 1h | BE |
| T169 | Implement FCM token registration endpoint: `POST /users/me/device-token` stores token in Redis (keyed by userID) | T161 | 1h | BE |

### 3.3 Leaderboard — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T170 | Implement `internal/leaderboard/repository.go`: Redis sorted set operations — update team score (ZIncrBy), update runner score (ZIncrBy), get top N (ZRevRangeWithScores), get user rank (ZRevRank) | — | 2h | BE |
| T171 | Implement `internal/leaderboard/service.go`: `GetTeamLeaderboard()` — per region (global/country/city), return top 100 factions by territory points | T170 | 2h | BE |
| T172 | Implement leaderboard service: `GetRunnerLeaderboard()` — per season, return top 100 runners by runner points, include current user rank | T170 | 2h | BE |
| T173 | Implement leaderboard service: `GetFriendsLeaderboard()` — friends-only ranking by runner points | T170, T155 | 2h | BE |
| T174 | Implement leaderboard service: `UpdateScores()` — called after run completion, update Redis sorted sets for runner and team | T170 | 1.5h | BE |
| T175 | Implement `internal/leaderboard/handler.go`: `GetTeamLeaderboard`, `GetRunnerLeaderboard`, `GetFriendsLeaderboard` handlers | T171-T173 | 1.5h | BE |
| T176 | Register leaderboard routes: `GET /leaderboard/team`, `GET /leaderboard/runners`, `GET /leaderboard/friends` | T175 | 0.5h | BE |
| T177 | Implement leaderboard cache refresh: periodic job (every 15 min) recalculates from DB to correct any drift | T174 | 1.5h | BE |

### 3.4 WebSocket — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T178 | Implement `internal/websocket/client.go`: WebSocket client — readPump (parse messages, handle subscribe/unsubscribe/run:position), writePump (send buffered messages to connection with timeout) | — | 3h | BE |
| T179 | Implement `internal/websocket/hub.go`: Hub — client registry (userID→client), territory subscription registry (bboxKey→userIDs), friend subscription registry (userID→followerIDs), broadcast channel | T178 | 4h | BE |
| T180 | Implement hub: `handleBroadcast()` — route messages by type: territory:changed → filter by bbox overlap, friend:* → filter by friend subscriptions | T179 | 2h | BE |
| T181 | Implement hub: `BroadcastTerritoryChange()` — public method called by territory service after capture, publishes to broadcast channel | T179 | 1h | BE |
| T182 | Implement `GET /ws` endpoint: authenticate via `?token=` query param (JWT), upgrade to WebSocket, create Client, register in Hub | T179, T049 | 2h | BE |
| T183 | Implement friend activity live broadcast: when friend starts or completes a run, push event to subscribed followers | T179, T087, T088 | 2h | BE |

### 3.5 Social Module — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T184 | Implement `core/models/friend.dart`: Friend model with status, lastRun, profile fields | T155 | 1h | FE |
| T185 | Implement `features/social/bloc/friends_bloc.dart`: FriendsState (friends list, pending list, feed, status), events (LoadFriends, AddFriend, AcceptFriend, RejectFriend, LoadFeed) | T184 | 3h | FE |
| T186 | Implement `features/social/screens/friends_list_screen.dart`: tab bar (Friends | Pending), list of friends with avatar, name, faction, account level, last run. Swipe-to-unfriend. "Add Friend" FAB | T185 | 4h | FE |
| T187 | Implement add friend dialog: search by username or phone number → send friend request | T186, T159 | 2h | FE |
| T188 | Implement `features/social/screens/friend_profile_screen.dart`: friend's public profile — avatar, name, faction, level, total stats, recent runs | T186 | 3h | FE |
| T189 | Implement `features/social/screens/friend_feed_screen.dart`: chronological feed of friends' runs — each item shows name, date, distance, pace, territory hexes captured, map thumbnail | T185, T157 | 4h | FE |
| T190 | Implement friend activity push notification handling: on tap → navigate to friend profile or run detail | T161-T168, T074 | 2h | FE |
| T191 | Implement `features/leaderboard/screens/leaderboard_screen.dart`: three tabs (Team | Runners | Friends), ranked list with rank number, avatar, name, faction badge, score, current user highlighted | T176, T185 | 5h | FE |
| T192 | Implement leaderboard filters: region selector (global/country/city dropdown), season selector | T191 | 2h | FE |
| T193 | Implement leaderboard: pull-to-refresh, pagination (load more on scroll to bottom) | T191 | 2h | FE |

---

## Phase 4 — Bots & Territory Seeding

### 4.1 Bot Seeding — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T194 | Implement `pkg/geo/landuse.go`: cell classification — determine if H3 cell is urban/suburban/rural/natural (mountain/water/protected) using population density data or OSM land-use tile lookup | — | 4h | BE |
| T195 | Implement bot name generator: randomized plausible names — first name + optional last name (from curated lists), avoid obvious patterns. Ensure unique display names. | — | 2h | BE |
| T196 | Implement `internal/bot/repository.go`: interface (CreateBot, GetByRegion, GetActiveBots, Deactivate, DeactivateAll, CountActivePerFaction) | T027, T035 | 1h | BE |
| T197 | Implement `internal/bot/repository_postgres.go`: all SQL queries for bots table | T196 | 2h | BE |
| T198 | Implement `internal/bot/service.go`: `SeedOnboarding()` — per LLD Section 4.4: compute 100km disk, classify cells, check occupancy, generate islands, assign 50/50 factions, insert hexes, create bot profiles | T197, T194, T195, T095 | 5h | BE |
| T199 | Implement bot seeding: island generation — pick random seed cells, cluster by H3 adjacency, enforce min 5 and max 200 cells per island, shape organic boundaries | T198, T095 | 3h | BE |
| T200 | Implement bot seeding: 60/40 split — enforce max 60% of eligible cells claimed, skip if already >60%. Exclude natural cells from both numerator and denominator. | T198 | 2h | BE |
| T201 | Implement bot seeding: trigger on user registration — after user created with location permission, fire goroutine to seed asynchronously, notify client via WebSocket when done | T198 | 2h | BE |
| T202 | Implement seeding WebSocket notification: `type: "seeding:complete"` event with seeded region summary (islands count, hexes created) | T201, T183 | 1h | BE |

### 4.2 Active Bot Spawning — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T203 | Implement `internal/bot/service.go`: `CountActivePlayers()` — distinct users per faction who completed a run in past 7 days within region | T097, T084 | 2h | BE |
| T204 | Implement bot service: `SpawnActiveBots()` — per LLD Section 4.5: iterate active regions, compute bots_needed per faction (10 - real_users), check density caps, spawn bot runs | T203, T197 | 5h | BE |
| T205 | Implement bot service: pick start point — random point within active radius (10km), must be on plausible path (OSM street data), must be >1km from real users, max 2 bots per 1km hex | T204 | 3h | BE |
| T206 | Implement bot service: determine faction — always opposite of nearest real user's faction to create opposition | T205 | 1h | BE |
| T207 | Implement bot service: calculate loop distance — proportional to average run distance of real users in area, fallback to 5km | T205 | 1h | BE |
| T208 | Implement `internal/bot/route.go`: plausible route generation — use cached OSM street network to create closed loop of target distance, snake through available streets, avoid dead ends | — | 6h | BE |
| T209 | Implement bot route: generate realistic GPS points — apply pacing variance (±15% of user avg), GPS noise (±3m), smooth speed transitions between segments | T208 | 3h | BE |
| T210 | Implement bot route: schedule run timing — between 6 AM and 10 PM local time, 100% probability at 7 AM and 6 PM peaks, linearly scaled down elsewhere | T209 | 2h | BE |
| T211 | Implement bot service: spawn timing caps — max 1 bot per 30 seconds in same city area, max 2 bots active in same 1km hex | T210 | 2h | BE |
| T212 | Implement bot service: simulate bot run — create run record, insert GPS points, apply territory capture (same as real user), update bot total_runs | T209, T088 | 3h | BE |
| T213 | Implement bot service: `PhaseOutCheck()` — if real users in region >= 20, deactivate all bots. If >= 10, reduce to 5-faction. If >= 5, reduce to 7-faction. | T203 | 2h | BE |
| T214 | Implement bot spawning trigger: after real user's first completed run, start 10-hour timer → begin spawning bots | T204 | 2h | BE |
| T215 | Wire bot spawner into scheduler: run every 15 minutes, also triggered on first user run event | T214, T028 | 1h | BE |

### 4.3 Bot Seeding — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T216 | Implement onboarding seeding UX: after location permission granted, show loading screen with animated map pulse + "Mapping your territory..." text | T201 | 2h | FE |
| T217 | Implement seeding complete screen: show region summary ("X hexes seeded across Y islands"), "Your region is ready. Choose your faction!" button | T216 | 2h | FE |
| T218 | Implement map screen: distinguish bot-runner hex captures from real-user captures via subtle visual cue (optional, post-MVP) | — | 1h | FE |

---

## Phase 5 — Polish & Launch

### 5.1 Season System — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T219 | Implement `internal/season/repository.go`: interface (GetCurrent, GetByID, Create, CreateParticipant, UpdateParticipant, FinalizeTiers) | T026, T034 | 1h | BE |
| T220 | Implement `internal/season/repository_postgres.go`: all SQL queries for seasons + season_participants | T219 | 2h | BE |
| T221 | Implement `internal/season/service.go`: `GetCurrentSeason()` — return active season with tier thresholds and user progress | T220 | 2h | BE |
| T222 | Implement season service: `RegisterParticipant()` — create season_participant row when user first runs in a season | T220 | 1h | BE |
| T223 | Implement season service: `UpdateProgress()` — increment runner_points and territory_points after each run, recalculate current tier | T222 | 1.5h | BE |
| T224 | Implement season service: `Rollover()` — calculate final tiers for all participants, award badges, create new season, archive old | T220 | 3h | BE |
| T225 | Implement `internal/season/handler.go`: `GetCurrentSeason` handler | T221 | 1h | BE |
| T226 | Register season route: `GET /seasons/current` | T225 | 0.5h | BE |
| T227 | Implement `POST /admin/season/rollover` (admin endpoint, IP-whitelisted) — manually trigger season rollover | T224 | 1h | BE |

### 5.2 Health Platform Integration — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T228 | Implement HealthKit integration (iOS): request authorization to read/write workouts and heart rate, fetch HR data for past run duration, attach to run end request | — | 4h | FE |
| T229 | Implement Health Connect integration (Android): request permissions, read heart rate data, attach to run end request | — | 4h | FE |
| T230 | Implement health sync toggle in profile: auto-sync runs to HealthKit/Health Connect after completion, gated by Runner tier subscription | T228, T229 | 2h | FE |
| T231 | Implement health data merge: if user records HR via watch, merge with GPS timestamps to produce HR chart on run summary | T228, T229, T140 | 3h | FE |

### 5.3 Stats & History — Flutter

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T232 | Implement `features/stats/screens/stats_screen.dart`: comprehensive stats dashboard — weekly/monthly/yearly charts for distance, runs count, territory captured, points earned | T147 | 5h | FE |
| T233 | Implement stats: personal records — fastest km, longest run, most elevation, most territory in single run | T232 | 2h | FE |
| T234 | Implement stats: territory contribution — % of team's total territory that your runs have captured, ranking within team | T232 | 2h | FE |
| T235 | Implement run detail map: interactive map showing trace, hexes captured highlighted in team color, hexes stolen highlighted in steal color (darker shade) | T147 | 3h | FE |

### 5.4 Mapbox Proxy — Backend

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T236 | Implement Mapbox tile proxy in Go: `GET /proxy/mapbox/*` handler — forward requests to api.mapbox.com, append access token, cache tiles in Redis (TTL 24h) | T013 | 3h | BE |
| T237 | Implement nginx Mapbox proxy alternative: `location /mapbox/` with proxy_pass + sub_filter to inject token (lighter weight, fallback option) | — | 2h | OPS |
| T238 | Update Flutter map configuration: use proxy URL instead of direct Mapbox endpoint | T236 or T237, T114 | 1h | FE |

### 5.5 Infrastructure & Deployment

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T239 | Write `docker-compose.prod.yml`: production overrides — resource limits, restart policies, volume mounts, read-only filesystem for API, secrets via env_file | T006, T007 | 2h | OPS |
| T240 | Write `nginx.conf` (prod): TLS termination, HTTP→HTTPS redirect, API reverse proxy, WebSocket upgrade, static asset caching, rate limit zones | — | 3h | OPS |
| T241 | Implement certbot integration: auto-obtain Let's Encrypt TLS cert, auto-renew cron | T240 | 2h | OPS |
| T242 | Write `deploy.sh`: rsync files to server, docker compose pull, docker compose up -d, run migrations, health check verify | T239, T240 | 2h | OPS |
| T243 | Implement backup script: `pg_dump` → gzip → upload to rustfs, delete backups older than 30 days, cron daily at 3 AM | T063 | 2h | OPS |
| T244 | Implement `make deploy` target calling deploy.sh | T242 | 0.5h | OPS |
| T245 | Write `systemd` service file: run docker compose on boot, restart on failure | T239 | 1h | OPS |
| T246 | Configure firewall (ufw/iptables): allow 80, 443 only. Block all other inbound. | — | 1h | OPS |
| T247 | Set up monitoring: Prometheus metrics endpoint `/metrics` with counters/gauges per LLD Section 8.2 | — | 3h | BE |
| T248 | Set up health check alerts: if `/health` fails 3 times in a row, trigger notification (can be simple: log + alert hook) | T247 | 1h | OPS |

### 5.6 Testing

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T249 | Write unit tests for `internal/auth/service.go`: register (success, duplicate email, weak password), login (success, wrong password, OAuth-only account), refresh (success, expired, token reuse) | T039-T043 | 6h | BE |
| T250 | Write unit tests for `internal/run/cheat.go`: all cheat detection cases per LLD Section 13.3 (clean run, motorcycle, car, teleportation, bike, ultrarunner, bad GPS, HR mismatch) | T086 | 4h | BE |
| T251 | Write unit tests for `internal/territory/hp.go`: HP state machine — all 9 transitions per LLD Section 13.3 | T099 | 2h | BE |
| T252 | Write unit tests for `internal/territory/capture.go`: closed loop polygon capture, open path capture (enabled/disabled), self-intersecting polygon extraction, hex count verification | T101, T108 | 4h | BE |
| T253 | Write unit tests for `internal/points/calculator.go`: distance points, elevation points, territory share, social multiplier (1 friend, 5 friends, capped), streak bonus | T110-T112 | 3h | BE |
| T254 | Write unit tests for `internal/bot/service.go`: seeding island generation, occupancy check, faction balancing, bot count calculation, phase-out thresholds | T198-T213 | 5h | BE |
| T255 | Write integration tests: HTTP handler tests using test database (testcontainers-go) — register → login → start run → end run with synthetic GPS → verify hexes created → verify points → verify leaderboard update | T014, T092, T104, T176 | 6h | BE |
| T256 | Write integration tests: friend flow — add → accept → feed → social run → multiplier verification | T158, T092 | 3h | BE |
| T257 | Write integration tests: WebSocket lifecycle — connect → subscribe territory → capture changes broadcast → receive event | T182, T101 | 4h | BE |
| T258 | Write integration tests: bot seeding → verify hex islands created → verify 60/40 split | T198 | 3h | BE |
| T259 | Write integration tests: HP decay cron → verify -1 per day → verify floor at 1 | T107 | 2h | BE |
| T260 | Write Flutter widget tests: login screen (form validation, Google/Apple buttons), register screen (password requirements), faction choice screen (cards, confirmation dialog) | T069-T075 | 4h | FE |
| T261 | Write Flutter widget tests: map screen (hex overlay rendering, run controls), run summary (stats display, charts rendering), leaderboard (ranked list, current user highlight) | T119, T140, T191 | 5h | FE |
| T262 | Write Flutter bloc tests: auth bloc (login success/error, refresh, logout), map bloc (loaded, moved, territory updated), run bloc (start, position updates, end) | T068, T116, T133 | 5h | FE |
| T263 | Write Flutter GPS service tests: point collection, distance filter, start/stop lifecycle | T130 | 2h | FE |
| T264 | Write Flutter integration test: full app flow — splash → login → see map with hexes → start run → simulate GPS → end run → view summary → check history | T081, T092, T140, T147 | 4h | FE |

### 5.7 Load Testing & Optimization

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T265 | Write k6 load test script: simulate 10k concurrent users, focus on territory queries (GET /territory?bbox=), run end (POST /runs/:id/end with GPS array), WebSocket connections | — | 4h | BE |
| T266 | Run load test, profile bottlenecks, optimize slow queries (add missing indexes, materialized views) | T265 | 8h | BE |
| T267 | Optimize territory bbox query: add spatial index on hexes, test performance with 1M+ hex rows | T098, T266 | 4h | BE |
| T268 | Optimize GPS point insert: benchmark pgx CopyFrom vs unnest, tune TimescaleDB chunk size | T084, T266 | 3h | BE |
| T269 | Optimize WebSocket: benchmark 1k concurrent connections, tune gorilla/ws buffer sizes, add connection limits | T179, T266 | 4h | BE |
| T270 | Profile Go API memory/CPU, fix leaks, add pprof endpoints for production debugging | T266 | 4h | BE |

### 5.8 Pre-Launch Checklist

| ID | Task | Depends | Effort | Owner |
|---|---|---|---|---|
| T271 | Final security review: JWT rotation tested, rate limits verified, Mapbox token not exposed, phone hashes properly salted, SQL injection scan, HTTPS enforced | T248 | 4h | BE |
| T272 | Environment hardening: disable debug logging, rotate all dev secrets to production secrets, restrict CORS to production domain | T246 | 2h | OPS |
| T273 | GDPR compliance: implement data export endpoint (`GET /users/me/export`), implement account deletion with 30-day soft-delete, test deletion cron | T055, T064 | 3h | BE |
| T274 | Write `README.md`: project overview, setup instructions (docker compose up), architecture summary, API documentation link, contributing guide | — | 3h | DEV |
| T275 | Configure beta launch: all monetization gates disabled (all users get Runner tier features free), set `PATH_CAPTURE_ENABLED=true`, seed initial territory for test users | T013 | 2h | BE |
| T276 | Create seed territory for major NA cities (NYC, LA, Chicago, Toronto, SF) via admin endpoint before public launch | T198 | 2h | BE |
| T277 | App store prep: iOS App Store screenshots, privacy policy, HealthKit usage description, location usage description. Google Play: privacy policy, permissions declaration. | T081 | 8h | FE |
| T278 | Beta distribution: TestFlight for iOS, internal testing track for Android. Invite initial testers. | T277 | 3h | DEV |

---

## Dependency Graph Summary

```
Phase 0 (T001-T012)
    └── Phase 1 (T013-T082)
            ├── Phase 2.1-2.3 (T083-T113) ──→ Phase 2.4-2.6 (T114-T149)
            ├── Phase 3.1-3.4 (T150-T183) ──→ Phase 3.5 (T184-T193)
            ├── T083-T113 ──→ Phase 4.2 (T203-T215)
            └── T105 ──→ Phase 4.1 (T194-T202)
                    └── Phase 4.3 (T216-T218)
                            └── Phase 5 (T219-T278)
```

## Effort Summary

| Phase | Backend Hours | Flutter Hours | OPS Hours | Total |
|---|---|---|---|---|
| 0 — Scaffolding | 3 | 1.5 | 3 | 7.5 |
| 1 — Foundation | 42 | 33.5 | 0 | 75.5 |
| 2 — Core Gameplay | 49 | 67.5 | 0 | 116.5 |
| 3 — Social & Engagement | 49 | 36 | 0 | 85 |
| 4 — Bots & Seeding | 55 | 5 | 0 | 60 |
| 5 — Polish & Launch | 31 | 34 | 26 | 91 |
| **Total** | **229** | **177.5** | **29** | **435.5** |

Estimated with 1 full-time backend + 1 full-time Flutter developer + shared OPS: **~12 weeks** to production beta.
