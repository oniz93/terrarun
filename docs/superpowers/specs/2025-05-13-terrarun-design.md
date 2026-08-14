# TerraRun — Design Spec

**Date:** 2025-05-13
**Status:** Draft (pre-implementation)

---

## 1. Elevator Pitch

TerraRun is a location-based running game blending Strava-style run tracking with Ingress-style territory control. Players choose between two factions — **Neon** (red) and **Umbra** (blue) — and compete for control of a real-world hex map. Every run is a territorial move: close a loop, capture hexes, steal from the enemy. Rich run stats, a dual point system, seasons, and friend mechanics drive engagement for individuals, groups, and whole factions.

---

## 2. Core Game Loop

1. Open app → see the map colored in Neon and Umbra hexes
2. Start a run → GPS tracks your path
3. End the run → if the loop is closed (within 100m of start), all hexes inside the polygon are captured. If open and path capture is enabled, hexes along the route are captured as a corridor instead.
4. Points awarded: territory points (team score) and runner points (personal score)
5. HP system modifies hex defense (see Section 5)
6. Repeat to defend turf, expand borders, or steal enemy territory

If start and end are farther than 100m apart AND path capture is disabled, the run is recorded with stats but does **not** capture territory and awards no territory points.

---

## 3. Factions

### Neon (Red)
- **Identity:** The firestarters. Run on impulse, adrenaline, raw speed.
- **Style:** Aggressive expansion — big loops, pushing frontiers, claiming first.
- **Motto:** "Strike first."

### Umbra (Blue)
- **Identity:** The shadow runners. Run with rhythm, precision, endurance.
- **Style:** Infiltration and maintenance — reclaiming, outlasting, methodical.
- **Motto:** "Last longer."

Faction choice is **permanent** and made during onboarding (after profile creation and contacts check).

---

## 4. Registration Flow

1. Email + password, Google SSO, or Apple SSO
2. Grant location permission (triggers onboarding bot seeding — see Section 10)
3. Create profile: display name, phone number (optional, contacts matching only), avatar
4. Contacts scan (opt-in): see friends already on TerraRun with their faction. Shows count per faction ("3 friends in Umbra, 1 in Neon"). Tap faction to see who.
5. Choose faction → permanent
6. Three-screen onboarding tutorial (run, capture, check stats)

Phone numbers are SHA-256 hashed client-side before any server communication. Raw contacts never leave the device.

---

## 5. Hex Grid & Territory Mechanics

### Grid System
- **Library:** H3 (Uber's hexagonal hierarchical geospatial indexing)
- **Resolution:** 11 (~50m edge, ~0.00215 km² per hex)
- Only hexes with an owner are stored in the database. Neutral hexes are implied by absence of a row.

### Hex HP System
Each owned hex has a Hit Point (HP) value, representing defense strength.

| Action | Effect |
|---|---|
| First capture of neutral hex | Flips to capturing team at 1 HP |
| Same-team run recaptures hex | +1 HP (cap: 10) |
| Enemy run captures hex | -1 HP |
| HP reaches 0 | Hex flips to capturing enemy at 1 HP |
| Daily decay (midnight UTC) | -1 HP, floor at 1 (never returns to neutral) |

### Capture Rules — Closed Loop

When a run forms a valid closed loop (start and end within configurable radius, default 100m):
- Compute the polygon from the GPS track
- Identify all H3 indexes with centroids inside the polygon
- Apply HP changes to each hex (see HP System table above)
- Territory points: +10 base per hex captured, +50% bonus (15pts) for stealing enemy hexes
- If hex was already runner's faction: still awards base points, HP increment applies

### Capture Rules — Open Run (Path Capture)

When a run does NOT close a loop AND path capture is enabled (config toggle):
- Buffer the GPS track line by a configurable radius (default 25m) to form a corridor polygon
- Identify all H3 indexes with centroids inside that corridor
- Apply same HP changes and point rules as closed-loop capture
- Path capture captures fewer hexes than an equivalent closed loop (stripe vs. full polygon interior), providing a gentler territory impact for casual runs
- **Configuration:** Path capture is a global runtime toggle. It can be enabled/disabled at any time without a restart. When disabled, non-closed runs award no territory points (personal stats only).

### Cheat Detection
**Hard gates (auto-reject territory capture):**
- Any GPS segment exceeding 35 km/h
- Average speed > 20 km/h for runs over 2 km
- GPS jumps > 500m between consecutive points (teleportation)
- Runs over 50 km in a single session (flagged for manual review)

**Soft flags (no territory capture, stats preserved):**
- Average speed > 15 km/h on flat terrain
- Low heart rate relative to pace (if HR data available)
- GPS accuracy consistently low (< 20m precision)

Flagged runs record personal stats but award **no territory points and no hex changes**. User can appeal for manual review.

Transport allowed: running and walking only in MVP. Bicycles, rollerblades, and motorized transport are auto-flagged.

---

## 6. Point System & Gamification

### Dual Point Tracks

**Territory Points (Team Score)**
- Awarded per hex flipped: +10 pts base, +15 pts (50% bonus) for enemy hexes
- Feeds team global ranking and per-region leaderboards
- Core metric: % territory control per country / region / city radius

**Runner Points (Personal)**

| Source | Amount |
|---|---|
| Distance | +1 pt per 100m |
| Elevation gain | +5 pts per 10m |
| Territory capture | +10% of territory points earned |
| Consistency streak | +X% bonus per consecutive day |
| Social multiplier | +25% with 1+ friends (+10% per extra, capped at 50%) |

### Account Level (Permanent)
- XP from all Runner Points (tuned conversion ratio)
- Fast early levels, slowing progressively
- Unlocks: cosmetic badges, profile flair, faction veteran title tiers
- Never resets

### Seasonal System (Resets every 2 months)
**Tiers:** Rookie → Runner → Sprinter → Elite → Neon/Umbra Legend
- Based on Runner Points earned within the season
- End-of-season rewards: exclusive seasonal badge (permanent), profile cosmetics, +10% territory multiplier for next season's first week
- Top X per region: "Regional Champion" badge

---

## 7. Social & Friends (MVP)

- **Friend list:** add by phone number (hashed matching), username, or contacts sync
- **Friend feed:** see friends' recent runs (distance, territory captured, date)
- **Run-together bonus:** if 2+ friends are within 20m when starting a run, all get the social multiplier (+25% base, +10% per additional friend, capped at 50%). Territory captured uses the run leader's faction.
- **Profile visibility:** see friends' stats, level, seasonal tier, run history
- No crew/clan system in v1

---

## 8. Run Stats

### During Run
- Distance, duration, current pace, average pace
- Territory overlay: "you are capturing X hexes," nearby team/enemy territory indicators

### End of Run
- Distance, duration, average pace, elevation gain, pace chart
- Heart rate zones (via Apple HealthKit / Google Health Connect)
- Split times per km (auto-lap)
- Elevation profile, cadence
- Map trace overlay
- Territory summary: hexes captured, team points earned

### Health Platform Integration
- Apple HealthKit: read/write runs, heart rate
- Google Health Connect: read/write runs, heart rate
- Gated behind Premium subscription (health sync is a Runner-tier feature)

### Run History
- Free tier: last 30 days
- Runner tier: unlimited + GPX/TCX export

---

## 9. Bot System

### Onboarding Seeding (new user grants location)
- Generate territory within 100 km radius of user
- Random islands (connected hex clusters), not uniform fill
- 50/50 chance per island for Neon/Umbra
- 40% of hexes in the 100 km radius left unclaimed
- Skip: mountains, rural areas, nature reserves (use population density / land-use tiles)
- Urban → dense; suburban → sparse; rural → none
- Skip if 100 km radius has already less than 40% hexes unclaimed
- In the 40% neutral hexes calculation exclude the nature / mountain / water hexes

### Active Bots (post first-run, 10-hour timer starts)
- Spawn bots within 10 km of real users
- Number: 10 bots per faction **minus** distinct real users of that faction who completed a run in the area in the past week (minimum 0)
- Bot loop distance: proportional to average run distance of real users in the area (default 5 km)
- Bots start at random points, different from real users
- Run schedule: 6 AM–10 PM local time, peak activity at 7 AM and 6 PM (100% active), off-peak reduced
- Density caps: max 1 bot per 30 seconds in the same city, max 2 bots active per 1 km hex
- Rural areas: neutral (no bots)
- Concentration increases after 1 km from user center
- Bots have to run on plausible paths (streets, parks, etc)

### Bot Profiles
- Plausible randomized names (not obviously bot-like)
- Visible run history and stats
- No direct social interaction

### Bot Scaling
- Bot density inversely proportional to real user count per region
- As real user base grows, bots phase out automatically

---

## 10. Regional Statistics

- Territory control statistics at multiple levels: global, country, region, city, custom radius
- % hex control per faction at each level
- Leaderboards: team dominance, individual runner ranking, friends ranking
- Regional Champion badges at season end

---

## 11. Technical Architecture

### Client
- **Framework:** Flutter (iOS + Android, single codebase)
- **Map:** Mapbox GL (free tier: 50k MAU)
- **Key plugins:** flutter_mapbox_gl, flutter_background_geolocation, health (iOS), health_connect (Android), google_sign_in, sign_in_with_apple, contacts_service

### Backend
- **Language:** Go
- **Web framework:** Chi or Echo router
- **WebSocket:** gorilla/websocket (live run tracking)
- **Geo processing:** custom Go + H3 library (h3-go), optional GEOS bindings for polygon ops
- **Auth:** JWT with refresh token rotation, OAuth2 for Google/Apple, bcrypt for email/password
- **Mapbox token proxy:** backend proxies all Mapbox tile requests — API key never exposed client-side

### Data Layer
| Component | Purpose |
|---|---|
| PostgreSQL + PostGIS | Users, runs, territories, friends, seasons |
| TimescaleDB (PG extension) | GPS point time-series hypertable |
| Redis | Session cache, leaderboards, real-time presence, rate limiting |
| rustfs (S3-compatible) | Avatar storage, run route thumbnails, GPX exports |

### Infrastructure
- All Dockerized (`docker-compose` for dev, `docker-compose.prod.yml` for production)
- Single VPS deployable (Hetzner, DigitalOcean, or self-hosted)
- nginx reverse proxy + Let's Encrypt TLS
- Systemd timers: backup rotation (pg_dump → rustfs daily), cert renewal
- Deploy: `rsync` docker-compose + env → `docker compose up -d`
- CI: local `make test`, `make lint`, pre-commit hooks

### External Services
| Service | Purpose | Self-hostable? |
|---|---|---|
| Mapbox | Map tiles, geocoding | No (free tier sufficient) |
| FCM + APNs | Push notifications | No (free, no data lock-in) |
| Google / Apple OAuth | Social sign-in | No (required) |
| Apple HealthKit / Google Health Connect | Run data sync | N/A (on-device APIs) |

### Scaling Path (post-MVP)
- PostgreSQL → managed instance or read replicas
- Go API → horizontal scale behind nginx/HAProxy
- Redis Sentinel for high availability

---

## 12. API Endpoints (Summary)

| Group | Key Endpoints |
|---|---|
| Auth | POST /auth/register, /auth/login, /auth/google, /auth/apple, /auth/refresh |
| Users | GET /users/me, PATCH /users/me, GET /users/:id |
| Runs | POST /runs/start, POST /runs/end (GPS array), GET /runs, GET /runs/:id, GET /runs/:id/gps |
| Territory | GET /territory?bbox=... (hexes for map viewport), GET /territory/stats?region=... |
| Friends | POST /friends/add, GET /friends, GET /friends/pending, POST /friends/:id/accept |
| Leaderboard | GET /leaderboard/team?region=..., GET /leaderboard/runners?season=..., GET /leaderboard/friends |
| Seasons | GET /seasons/current, GET /seasons/:id/rewards |
| Bots | (internal) POST /bots/seeder, GET /bots/status |
| WebSocket | ws:// /ws/run/:run_id (live position broadcast) |

---

## 13. Monetization (Post-Beta)

### Free Tier (permanent)
- Basic run stats (distance, duration, pace, map)
- Territory capture, friend list, seasonal participation
- Run history (last 30 days)

### TerraRun Runner ($4.99/mo or $39.99/yr)
- Full advanced stats: splits, pace chart, elevation profile, heart rate zones, training load, cadence
- Unlimited run history + GPX/TCX export
- Apple Health / Google Health Connect sync
- Customizable profile badge

### TerraRun Captain ($9.99/mo or $79.99/yr)
- Everything in Runner
- Crew creation (persistent group of 10) — post-MVP feature
- Crew analytics (territory heatmaps, combined stats)
- Monthly XP booster (+20%)
- Early access to new features

### Future Revenue Streams
- **Territory sponsorships:** local businesses sponsor hex areas, branded pin on map, XP bonus for capturing near sponsor location
- **Event partnerships:** branded running events, marathon tie-ins, team challenge sponsorships

---

## 14. Security & Privacy

- GPS tracks stored encrypted at rest
- Location data never shared with third parties
- Phone numbers SHA-256 hashed client-side; raw contacts stay on device
- Contacts matching: app sends hashes → server matches → returns faction counts only
- Mapbox API key proxied through backend
- JWT with refresh token rotation, refresh token reuse detection
- Rate limiting on auth endpoints
- GDPR/CCPA compliance: data export, account deletion with 30-day grace period

---

## 15. MVP Scope

### In Scope
- Auth (email + Google + Apple)
- Profile management
- Run tracking with cheat detection
- Hex territory capture with HP system
- Run stats (all fields) and history
- Map with hex overlay
- Friend system (add, feed, run-together bonus)
- Bot system (onboarding seeding + active bots)
- Dual point system (territory + runner)
- Account levels
- Regional leaderboards
- HealthKit / Health Connect (Runner tier)
- Push notifications for friend activity

### Out of Scope (Post-MVP)
- Crew/clan system
- Seasonal tiers
- Crew challenges
- Territory sponsorships
- Event partnerships
- Cross-team rivalries (auto-pairing)

### Beta
- Completely free during beta period
- Full feature set available
- Monetization gates disabled
