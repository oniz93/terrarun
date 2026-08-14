# TerraRun — High-Level Design (HLD)

**Date:** 2025-05-13
**Version:** 1.0
**Depends on:** [Design Spec](./2025-05-13-terrarun-design.md)

---

## 1. System Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                       CLIENT LAYER                           │
│  ┌─────────────────┐  ┌─────────────────────────────────┐   │
│  │  Flutter App     │  │  Background GPS Service         │   │
│  │  (iOS/Android)   │  │  (flutter_background_geoloc)    │   │
│  └───────┬─────────┘  └──────────────┬──────────────────┘   │
│          │                           │                       │
│          │  REST + WebSocket         │ REST (run end)        │
└──────────┼───────────────────────────┼───────────────────────┘
           │                           │
┌──────────┼───────────────────────────┼───────────────────────┐
│          │       GATEWAY LAYER        │                       │
│  ┌───────┴───────────────────────────┴──────────────────┐    │
│  │                 nginx (reverse proxy)                │    │
│  │    - TLS termination (Let's Encrypt)                 │    │
│  │    - Rate limiting (limit_req_zone)                  │    │
│  │    - WebSocket upgrade passthrough                   │    │
│  │    - Mapbox tile proxy (strips auth, adds token)     │    │
│  │    - Static asset caching (avatars, thumbnails)      │    │
│  └───────────────────────┬──────────────────────────────┘    │
└──────────────────────────┼───────────────────────────────────┘
                           │
┌──────────────────────────┼───────────────────────────────────┐
│                     APPLICATION LAYER                        │
│  ┌───────────────────────┴──────────────────────────────┐    │
│  │              Go API Server (Chi/Echo)                │    │
│  │                                                      │    │
│  │  ┌──────────┐  ┌───────────┐  ┌──────────────────┐   │    │
│  │  │ Auth     │  │ Run       │  │ Territory        │   │    │
│  │  │ Service  │  │ Service   │  │ Service          │   │    │
│  │  └────┬─────┘  └─────┬─────┘  └────────┬─────────┘   │    │
│  │       │              │                  │             │    │
│  │  ┌────┴──────────────┴──────────────────┴─────────┐   │    │
│  │  │              Shared Domain Logic               │   │    │
│  │  │  - H3 capture engine                          │   │    │
│  │  │  - Cheat detection                            │   │    │
│  │  │  - Point calculator                           │   │    │
│  │  │  - Bot scheduler                              │   │    │
│  │  └──────────────────────┬───────────────────────┘   │    │
│  └─────────────────────────┼───────────────────────────┘    │
│                            │                                 │
│  ┌─────────────────────────┼───────────────────────────┐    │
│  │           WebSocket Manager (gorilla/ws)            │    │
│  │  - Run position broadcast                          │    │
│  │  - Friend activity notifications                   │    │
│  │  - Territory change stream                         │    │
│  └─────────────────────────┼───────────────────────────┘    │
│                            │                                 │
│  ┌─────────────────────────┼───────────────────────────┐    │
│  │              Background Job Runner                  │    │
│  │  - Daily HP decay (midnight UTC cron)              │    │
│  │  - Bot spawning scheduler                          │    │
│  │  - Season end processing                           │    │
│  │  - Leaderboard refresh                             │    │
│  └─────────────────────────┼───────────────────────────┘    │
└────────────────────────────┼────────────────────────────────┘
                             │
┌────────────────────────────┼────────────────────────────────┐
│                       DATA LAYER                             │
│  ┌──────────────┐  ┌──────────┐  ┌───────────────┐          │
│  │ PostgreSQL   │  │  Redis   │  │    rustfs     │          │
│  │  + PostGIS   │  │          │  │ (S3-compat)   │          │
│  │  + Timescale │  │          │  │               │          │
│  └──────────────┘  └──────────┘  └───────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### Request Lifecycle

```
Client → nginx (TLS + rate limit) → Go API (auth middleware + handler)
                                        ↓
                ┌───────────────────────┴───────────────────────┐
                │  Handler: validate input, call service layer  │
                │  Service: business logic, coordinate repos    │
                │  Repository: SQL queries, Redis ops           │
                └───────────────────────┬───────────────────────┘
                                        ↓
                              PostgreSQL / Redis / rustfs
```

All vertical slices follow this pattern. Each service package has its own handler/service/repository triad.

---

## 2. Go Project Structure

```
backend/
├── cmd/
│   └── api/
│       └── main.go              # Entry point, wire dependencies
├── internal/
│   ├── config/
│   │   └── config.go            # Env-based config (envconfig)
│   ├── middleware/
│   │   ├── auth.go              # JWT validation
│   │   ├── ratelimit.go         # Per-IP / per-user rate limits
│   │   ├── logging.go           # Request/response logging
│   │   └── recovery.go          # Panic recovery
│   ├── auth/
│   │   ├── handler.go           # POST /auth/*
│   │   ├── service.go           # Register, login, refresh, OAuth
│   │   ├── repository.go        # User queries
│   │   └── jwt.go               # Token generation/validation
│   ├── user/
│   │   ├── handler.go           # GET/PATCH /users/*
│   │   ├── service.go           # Profile, contacts, avatar
│   │   └── repository.go
│   ├── run/
│   │   ├── handler.go           # POST /runs/start, /runs/end, GET /runs/*
│   │   ├── service.go           # Start, track, complete, stats
│   │   ├── repository.go        # Run + GPS point queries
│   │   ├── cheat.go             # Cheat detection engine
│   │   └── stats.go             # Run statistics computation
│   ├── territory/
│   │   ├── handler.go           # GET /territory/*
│   │   ├── service.go           # Capture logic, HP system, regions
│   │   ├── repository.go        # Hex queries, spatial indexes
│   │   ├── capture.go           # H3 polygon → hex set computation
│   │   └── hp.go                # HP addition/removal/decay
│   ├── friend/
│   │   ├── handler.go           # /friends/*
│   │   ├── service.go           # Add, accept, feed, contacts
│   │   └── repository.go
│   ├── leaderboard/
│   │   ├── handler.go           # /leaderboard/*
│   │   ├── service.go           # Rankings computation
│   │   └── repository.go
│   ├── season/
│   │   ├── handler.go           # /seasons/*
│   │   ├── service.go           # Current season, rewards
│   │   └── repository.go
│   ├── bot/
│   │   ├── service.go           # Seeder + active bot logic
│   │   ├── scheduler.go         # Bot run timing
│   │   ├── route.go             # Plausible path generation
│   │   └── repository.go        # Bot profile storage
│   ├── points/
│   │   └── calculator.go        # Point computation shared across services
│   ├── websocket/
│   │   ├── hub.go               # WebSocket connection manager
│   │   └── client.go            # Per-connection handler
│   ├── notification/
│   │   ├── service.go           # Push notification dispatch
│   │   └── firebase.go          # FCM client
│   └── storage/
│       └── rustfs.go            # S3-compatible file storage client
├── pkg/
│   ├── h3util/
│   │   └── h3.go                # H3 utilities (polygon → cells, etc.)
│   ├── geo/
│   │   └── geo.go               # Distance, bounding box, polygon ops
│   └── validator/
│       └── validator.go         # Input validation helpers
├── migrations/
│   ├── 001_users.up.sql
│   ├── 002_runs.up.sql
│   ├── 003_territory.up.sql
│   └── ...
├── Dockerfile
├── docker-compose.yml
├── docker-compose.prod.yml
├── Makefile
├── go.mod
└── go.sum
```

### Flutter Project Structure

```
client/
├── lib/
│   ├── main.dart
│   ├── app.dart                # MaterialApp, routing, theme
│   ├── config/
│   │   ├── env.dart            # Environment config
│   │   └── theme.dart          # Neon/Umbra theme definitions
│   ├── features/
│   │   ├── auth/
│   │   │   ├── screens/
│   │   │   │   ├── login_screen.dart
│   │   │   │   ├── register_screen.dart
│   │   │   │   └── faction_choice_screen.dart
│   │   │   ├── widgets/
│   │   │   └── bloc/            # Auth state (flutter_bloc)
│   │   ├── map/
│   │   │   ├── screens/
│   │   │   │   └── map_screen.dart
│   │   │   ├── widgets/
│   │   │   │   ├── hex_overlay.dart
│   │   │   │   ├── run_controls.dart
│   │   │   │   └── territory_stats.dart
│   │   │   └── bloc/
│   │   ├── run/
│   │   │   ├── screens/
│   │   │   │   ├── run_active_screen.dart
│   │   │   │   ├── run_summary_screen.dart
│   │   │   │   └── run_history_screen.dart
│   │   │   ├── widgets/
│   │   │   │   ├── pace_chart.dart
│   │   │   │   ├── elevation_profile.dart
│   │   │   │   └── splits_table.dart
│   │   │   ├── services/
│   │   │   │   └── gps_service.dart
│   │   │   └── bloc/
│   │   ├── social/
│   │   │   ├── screens/
│   │   │   │   ├── friends_list_screen.dart
│   │   │   │   ├── friend_profile_screen.dart
│   │   │   │   └── friend_feed_screen.dart
│   │   │   └── bloc/
│   │   ├── leaderboard/
│   │   │   ├── screens/
│   │   │   │   └── leaderboard_screen.dart
│   │   │   └── bloc/
│   │   ├── profile/
│   │   │   ├── screens/
│   │   │   │   ├── profile_screen.dart
│   │   │   │   └── edit_profile_screen.dart
│   │   │   └── bloc/
│   │   └── stats/
│   │       ├── screens/
│   │       │   └── stats_screen.dart
│   │       └── bloc/
│   ├── core/
│   │   ├── api/
│   │   │   ├── api_client.dart        # HTTP client (dio)
│   │   │   ├── api_endpoints.dart     # endpoint constants
│   │   │   └── ws_client.dart         # WebSocket client
│   │   ├── storage/
│   │   │   └── secure_storage.dart    # Token storage (flutter_secure_storage)
│   │   ├── models/
│   │   │   ├── user.dart
│   │   │   ├── run.dart
│   │   │   ├── hex.dart
│   │   │   ├── territory.dart
│   │   │   └── friend.dart
│   │   └── extensions/
│   │       └── datetime.dart
│   └── shared/
│       └── widgets/
│           ├── loading_indicator.dart
│           ├── error_view.dart
│           └── avatar_widget.dart
├── test/
├── pubspec.yaml
└── analysis_options.yaml
```

---

## 3. Database Schema

### 3.1 Users

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT,                          -- NULL for OAuth-only users
    display_name    TEXT NOT NULL,
    phone_hash      TEXT,                          -- SHA-256, nullable
    avatar_url      TEXT,
    faction         TEXT CHECK (faction IN ('neon', 'umbra')), -- nullable until chosen
    account_level   INTEGER NOT NULL DEFAULT 1,
    account_xp      BIGINT NOT NULL DEFAULT 0,

    -- OAuth identities
    google_id       TEXT UNIQUE,
    apple_id        TEXT UNIQUE,

    -- Tiers
    runner_tier     TEXT NOT NULL DEFAULT 'free'   -- 'free', 'runner', 'captain'
        CHECK (runner_tier IN ('free', 'runner', 'captain')),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ                    -- soft delete (GDPR grace period)
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_faction ON users(faction) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone_hash ON users(phone_hash) WHERE deleted_at IS NULL AND phone_hash IS NOT NULL;
```

### 3.2 Auth Sessions (Refresh Tokens)

```sql
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token   TEXT UNIQUE NOT NULL,
    device_info     TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at      TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(refresh_token);
```

### 3.3 Runs

```sql
CREATE TABLE runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'completed', 'flagged', 'rejected')),

    -- Stats (populated on completion)
    distance_m      DOUBLE PRECISION,
    elevation_gain_m DOUBLE PRECISION,
    avg_pace_s_per_km DOUBLE PRECISION,
    max_speed_kmh   DOUBLE PRECISION,
    avg_hr_bpm      INTEGER,
    max_hr_bpm      INTEGER,
    calories        INTEGER,

    -- Capture
    closed_loop     BOOLEAN,
    capture_mode    TEXT CHECK (capture_mode IN ('polygon', 'path')),
    loop_snap_distance_m DOUBLE PRECISION,         -- actual snap distance used
    path_buffer_radius_m DOUBLE PRECISION,         -- buffer radius if path capture mode

    -- Encoded track
    polyline        TEXT,                          -- Encoded polyline (Google format)

    -- Points earned
    territory_points INTEGER NOT NULL DEFAULT 0,
    runner_points   INTEGER NOT NULL DEFAULT 0,

    -- Social
    social_run      BOOLEAN NOT NULL DEFAULT FALSE,
    social_leader   UUID REFERENCES users(id),    -- who pressed start
    social_participants UUID[],

    -- Health source
    health_source   TEXT,                          -- 'healthkit', 'healthconnect', null

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runs_user ON runs(user_id, created_at DESC);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_start_time ON runs(start_time);

-- Hypertable for GPS points (TimescaleDB)
CREATE TABLE gps_points (
    run_id          UUID NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL,
    lat             DOUBLE PRECISION NOT NULL,
    lng             DOUBLE PRECISION NOT NULL,
    altitude        DOUBLE PRECISION,
    speed           DOUBLE PRECISION,             -- m/s
    horizontal_accuracy DOUBLE PRECISION,
    heart_rate      INTEGER
);

SELECT create_hypertable('gps_points', 'timestamp');
CREATE INDEX idx_gps_points_run ON gps_points(run_id, timestamp);
```

### 3.4 Territory (Hexes)

```sql
CREATE TABLE hexes (
    h3_index        BIGINT PRIMARY KEY,           -- H3 index at resolution 11
    owned_by        TEXT CHECK (owned_by IN ('neon', 'umbra')),
    hp              SMALLINT NOT NULL DEFAULT 1
        CHECK (hp >= 1 AND hp <= 10),
    captured_by     UUID REFERENCES runs(id),
    captured_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_decayed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_natural      BOOLEAN NOT NULL DEFAULT FALSE  -- mountain, water, nature
);

CREATE INDEX idx_hexes_owned ON hexes(owned_by) WHERE owned_by IS NOT NULL;
CREATE INDEX idx_hexes_decay ON hexes(last_decayed_at) WHERE owned_by IS NOT NULL;

-- Change history for periodic aggregation
CREATE TABLE territory_changes (
    id              BIGSERIAL PRIMARY KEY,
    h3_index        BIGINT NOT NULL,
    run_id          UUID REFERENCES runs(id),
    previous_owner  TEXT,
    new_owner       TEXT CHECK (new_owner IN ('neon', 'umbra')),
    hp_before       SMALLINT NOT NULL,
    hp_after        SMALLINT NOT NULL,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_territory_changes_time ON territory_changes(changed_at);
CREATE INDEX idx_territory_changes_run ON territory_changes(run_id);
```

### 3.5 Friends

```sql
CREATE TABLE friends (
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'blocked')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, friend_id),
    CHECK (user_id != friend_id)
);

CREATE INDEX idx_friends_friend ON friends(friend_id, status);
```

### 3.6 Seasons

```sql
CREATE TABLE seasons (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    start_date      TIMESTAMPTZ NOT NULL,
    end_date        TIMESTAMPTZ NOT NULL,
    tiers_config    JSONB NOT NULL,               -- tier thresholds
    is_active       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_active UNIQUE (is_active)   -- only one active at a time
        DEFERRABLE INITIALLY DEFERRED
);

-- Per-user season participation
CREATE TABLE season_participants (
    user_id         UUID NOT NULL REFERENCES users(id),
    season_id       UUID NOT NULL REFERENCES seasons(id),
    runner_points   INTEGER NOT NULL DEFAULT 0,
    territory_points INTEGER NOT NULL DEFAULT 0,
    current_tier    TEXT NOT NULL DEFAULT 'rookie',
    final_tier      TEXT,
    badge_awarded   BOOLEAN NOT NULL DEFAULT FALSE,

    PRIMARY KEY (user_id, season_id)
);
```

### 3.7 Bots

```sql
CREATE TABLE bots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name    TEXT NOT NULL,
    faction         TEXT NOT NULL CHECK (faction IN ('neon', 'umbra')),
    avatar_seed     TEXT,                          -- deterministic avatar
    home_lat        DOUBLE PRECISION NOT NULL,
    home_lng        DOUBLE PRECISION NOT NULL,
    active_radius_km DOUBLE PRECISION NOT NULL DEFAULT 5,
    avg_distance_m  DOUBLE PRECISION NOT NULL DEFAULT 5000,
    total_runs      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at  TIMESTAMPTZ                   -- NULL if still active
);

CREATE INDEX idx_bots_home ON bots(home_lat, home_lng);
CREATE INDEX idx_bots_faction ON bots(faction, deactivated_at);
```

### 3.8 Leaderboard (Materialized)

```sql
CREATE MATERIALIZED VIEW leaderboard_regional AS
SELECT
    u.id AS user_id,
    u.display_name,
    u.faction,
    u.account_level,
    COALESCE(SUM(r.runner_points), 0) AS total_runner_points,
    COALESCE(SUM(r.territory_points), 0) AS total_territory_points,
    COUNT(r.id) AS total_runs,
    -- Approximate region via hex ownership centroid
    ST_GeoHash(ST_Centroid(ST_Collect(ST_SetSRID(ST_MakePoint(
        -- Derive approximate center from captured territory
        h3_cell_to_lat(h3_index),
        h3_cell_to_lng(h3_index)
    ), 4326)))) AS region_hash
FROM users u
LEFT JOIN runs r ON r.user_id = u.id AND r.status = 'completed'
WHERE u.deleted_at IS NULL
GROUP BY u.id;

-- Refreshed via cron job
```

---

## 4. Core Algorithms

### 4.1 Run Capture Algorithm

```
Input:  run_id, ordered gps_points[{lat, lng, ts, ...}], social_friends[]
Output: territory_points, runner_points, hex_changes[]

1. VALIDATE
   a. Run anti-cheat checks (see 4.2)
   b. If flagged → mark run.flagged, return NO territory points, compute runner points only
   c. If rejected → mark run.rejected, return ZERO points

2. DETERMINE CAPTURE MODE
   a. start_point = gps_points[0]
   b. end_point = gps_points[-1]
   c. distance = haversine(start_point, end_point)
   d. closure_radius = configurable, default 100m
   e. IF distance <= closure_radius:
        capture_mode = 'polygon'
      ELSE IF config.PATH_CAPTURE_ENABLED == true:
        capture_mode = 'path'
      ELSE:
        mark run.closed_loop = false
        compute stats, award 0 territory points, compute runner points without territory component
        RETURN

3. BUILD GEOMETRY (depends on capture_mode)

   a. POLYGON MODE (loop closed):
      i.   Simplify GPS track (Douglas-Peucker, epsilon = 10m) to reduce point count
      ii.  Build polygon ring: simplified_points → closed ring
      iii. Validate: polygon must not self-intersect catastrophically
           - If self-intersection detected, attempt to extract largest simple sub-polygon
           - If no valid simple polygon → fall back to capture_mode = 'path' if enabled, else RETURN

   b. PATH MODE (open run, path capture enabled):
      i.   Simplify GPS track (Douglas-Peucker, epsilon = 10m)
      ii.  Build line from simplified points
      iii. Buffer the line by PATH_CAPTURE_BUFFER_RADIUS (configurable, default 25m) to form a corridor polygon
      iv.  If the corridor self-intersects (e.g., run doubles back), split into simple sub-corridors

4. HEX EXTRACTION
   a. Use h3_polygon_to_cells(capture_polygon, resolution=11)
   b. Get list of H3 indexes whose centers are inside the polygon
   c. Deduplicate

5. APPLY CAPTURE (in transaction)
   a. FOR each hex_index in extracted_hexes:
        current_hex = SELECT * FROM hexes WHERE h3_index = hex_index FOR UPDATE
        IF current_hex is NULL:
            INSERT INTO hexes (h3_index, owned_by='{runner.faction}', hp=1, captured_by=run_id)
            territory_points += 10
        ELSE IF current_hex.owned_by == runner.faction:
            new_hp = min(current_hex.hp + 1, 10)
            UPDATE hexes SET hp = new_hp
            territory_points += 10
        ELSE IF current_hex.owned_by != runner.faction:
            new_hp = current_hex.hp - 1
            IF new_hp <= 0:
                -- FLIP
                UPDATE hexes SET owned_by='{runner.faction}', hp=1, captured_by=run_id, captured_at=NOW()
                territory_points += 15  -- +50% bonus for stealing
            ELSE:
                UPDATE hexes SET hp = new_hp
                territory_points += 10
        INSERT INTO territory_changes (h3_index, run_id, previous_owner, new_owner, hp_before, hp_after)

6. COMPUTE POINTS
   a. runner_points = distance_m / 100 + elevation_gain_m / 2 + territory_points * 0.1
   b. Apply social multiplier if social_run:
        base = 1.0
        bonus = 0.25 + 0.1 * (len(social_participants) - 1)  # capped at 0.50 below
        multiplier = min(1.0 + bonus, 1.50)
        runner_points *= multiplier
   c. Apply streak multiplier if applicable
   d. Update run record with computed points
   e. Update user XP: user.account_xp += runner_points * XP_CONVERSION_RATIO
   f. Recompute user.account_level from XP
   g. Update season_participants for current season

7. BROADCAST
   a. Publish territory change event via WebSocket to all connected clients in bbox
   b. Send push notification to friends
   c. Update Redis leaderboard cache
```

### 4.2 Cheat Detection Algorithm

```
Input:  gps_points[{lat, lng, ts, speed, horizontal_accuracy, altitude}], run_distance_m, run_duration_s
Output: status IN ('clean', 'flagged', 'rejected')

1. SPEED CHECKS
   a. FOR each consecutive pair of gps_points:
        segment_distance = haversine(p[i], p[i+1])
        segment_duration = ts[i+1] - ts[i]
        segment_speed_kmh = (segment_distance / segment_duration) * 3.6
        IF segment_speed_kmh > 35:
            RETURN 'rejected' (motorized: any segment > 35 km/h)
   b. avg_speed_kmh = (total_distance / total_duration) * 3.6
        IF avg_speed_kmh > 20 AND run_distance > 2000:
            RETURN 'rejected' (motorized: sustained > 20 km/h for non-sprint distance)

2. TELEPORTATION CHECK
   a. FOR each consecutive pair:
        IF segment_distance > 500:  # impossible GPS jump between points
            RETURN 'rejected'

3. HUMAN FEASIBILITY CHECKS
   a. IF avg_speed_kmh > 15 AND flat_terrain:
        SET status = 'flagged'  (suspicious: bike-like speed on flat)
   b. IF total_distance > 50000:  # 50km
        SET status = 'flagged'  (ultra-runner possible, but rare)

4. ACCURACY CHECK
   a. low_accuracy_points = count(points where horizontal_accuracy > 20)
   b. IF low_accuracy_points / total_points > 0.3:
        SET status = 'flagged'  (poor GPS quality)

5. HEART RATE CORROBORATION (if HR data available)
   a. avg_speed_m_s = avg_speed_kmh / 3.6
   b. expected_min_hr = resting_hr + (max_hr - resting_hr) * 0.4  # moderate effort
   c. IF avg_hr < expected_min_hr:
        SET status = 'flagged'  (HR too low for pace — likely vehicle)

RETURN status
```

### 4.3 HP Daily Decay (Cron Job — Midnight UTC)

```
FOR hex IN (SELECT h3_index, hp, owned_by FROM hexes WHERE owned_by IS NOT NULL AND hp > 1):
    new_hp = hex.hp - 1
    UPDATE hexes SET hp = new_hp, last_decayed_at = NOW() WHERE h3_index = hex.h3_index
    INSERT INTO territory_changes (h3_index, NULL, owned_by, owned_by, hex.hp, new_hp, decay=true)
```

### 4.4 Bot Onboarding Seeder Algorithm

```
Input: user_lat, user_lng
Output: seeded hexes

1. BOUNDING REGION
   region = circle(center=(user_lat, user_lng), radius_km=100)
   h3_cells = h3_disk(user_h3_index, k=...covering ~100km)

2. CLASSIFY CELLS
   FOR cell IN h3_cells:
        cell_type = classify(cell) using population_density_tiles / land_use_data
        IF cell_type IN ['mountain', 'water', 'nature_reserve', 'protected']:
            cell.is_natural = TRUE
            CONTINUE (exclude from seeding)

3. CHECK EXISTING COVERAGE
   claimed = COUNT(hexes WHERE h3_index IN h3_cells AND owned_by IS NOT NULL)
   total_eligible = COUNT(h3_cells) - natural_count
   occupancy = claimed / total_eligible
   IF occupancy >= 0.60:
        RETURN (skip — already > 60% claimed, don't seed)

4. GENERATE ISLANDS
   target_claimed = total_eligible * 0.60  # 60% max claimed, 40% free
   remaining = target_claimed - claimed
   # Select random seed cells among eligible unclaimed cells
   seed_cells = random_sample(remaining_eligible_cells, count=remaining * 1.3)  # oversample
   # Cluster seed cells into islands (neighboring H3 cells)
   islands = h3_cluster(seed_cells, min_island_size=5, max_island_size=200)
   # Drop islands until we reach target

5. ASSIGN FACTION
   FOR island IN islands:
        faction = RANDOM('neon', 'umbra')  # 50/50
        FOR cell IN island:
            INSERT INTO hexes (h3_index, owned_by=faction, hp=1)

6. CREATE BOT PROFILES FOR SEEDED TERRITORY
   FOR each seeded region, create bot profiles:
        bot_count = 3 + (island_size / 50)  # more bots for larger islands
        Create bots with random plausible names, matching faction
```

### 4.5 Active Bot Spawning Algorithm

```
Trigger: 10 hours after real user's first completed run
Runs: Every 15 minutes (or on-demand check)

Input: real_users_in_region (completed run in past 7 days)
Output: bot runs created

1. FOR each active region (has at least 1 real user active in past week):
    2. Calculate distinct real users per faction:
        neon_players = COUNT users WHERE faction='neon' AND recent_run_in_10km
        umbra_players = COUNT users WHERE faction='umbra' AND recent_run_in_10km

    3. Calculate bot allocation:
        neon_bots_needed = MAX(0, 10 - neon_players)
        umbra_bots_needed = MAX(0, 10 - umbra_players)

    4. IF total_bots_needed == 0: CONTINUE

    5. For each bot to spawn:
        a. Select random start point within 10km of user centers
           - Must be on a plausible path (use OSM road/park data or route via road network)
           - Must be > 1km from any real user's current position
           - Must be in an area with < 2 active bots in same 1km hex

        b. Determine bot faction:
           - Find nearest real user → use OPPOSITE faction
           - Creates more dynamic territorial tension

        c. Calculate loop distance:
           avg_user_distance = AVG run distance of real users in region
           IF avg_user_distance > 0: loop_distance = avg_user_distance
           ELSE: loop_distance = 5000  # default 5km

        d. Check density constraints:
           - Max 1 bot spawning per 30 seconds in same city
           - Max 2 bots active simultaneously in same 1km hex
           - Skip if violation

        e. Determine run time:
           - Only between 6:00 AM and 10:00 PM local time
           - Peak hours (7 AM, 6 PM): 100% probability of spawning
           - Off-peak: scaled down linearly (e.g., 50% at 12 PM, 20% at 9 PM)
           - Random delay within current time window

        f. Generate plausible route:
           - Use road/path network to create closed loop of loop_distance
           - Snake through available streets
           - Create GPS points with realistic pacing (variance ±15% of user avg pace)
           - Add small GPS noise (±3m) for realism

        g. Record bot run:
           - INSERT INTO runs (user_id=NULL, bot_id=bot.id, ...)
           - INSERT gps_points for the generated route
           - Queue territory capture to be processed same as real runs
           - Bot runs are processed offline (no real-time WebSocket needed)

        h. After bot run completes, run capture algorithm to flip territory
           - Bot runs always run OPPOSITE faction territories to create opposition
```

### 4.6 Bot Scaling / Phase-Out

```
Daily check per km-hex cell (midnight):

FOR each 10km region cell with bots:
    real_neon = COUNT real users in region (faction=neon, active in past week)
    real_umbra = COUNT real users in region (faction=umbra, active in past week)
    total_real = real_neon + real_umbra

    IF total_real >= 20:  # region is healthy
        Deactivate all bots in region immediately
    ELSE IF total_real >= 10:
        Reduce bots to MAX(0, 5 - per_faction_real)
    ELSE IF total_real >= 5:
        Reduce bots to MAX(0, 7 - per_faction_real)
    ELSE:
        Keep current allocation (10 - per_faction_real)
```

---

## 5. API Specifications (Detailed)

### 5.1 Auth

**POST /auth/register**
```json
// Request
{
  "email": "runner@example.com",
  "password": "min8chars",
  "display_name": "SpeedDemon",
  "phone_hash": "sha256...",  // optional
  "faction": "neon"           // chosen during onboarding
}
// Response 201
{
  "user": { "id": "...", "display_name": "SpeedDemon", "faction": "neon", ... },
  "access_token": "jwt...",
  "refresh_token": "uuid..."
}
```

**POST /auth/login**
```json
// Request
{ "email": "...", "password": "..." }
// Response 200 — same shape as register
```

**POST /auth/google**
```json
// Request
{ "id_token": "google_id_token", "phone_hash": "sha256...", "faction": "umbra" }
// Response 200 — same shape; auto-creates user if first login
```

**POST /auth/apple**
```json
// Request
{ "identity_token": "apple_jwt", "user_identifier": "apple_user_id", ... }
// Response 200 — same shape
```

**POST /auth/refresh**
```json
// Request — HTTP-only cookie or body
{ "refresh_token": "uuid..." }
// Response 200
{ "access_token": "new_jwt..." }
// If refresh token reuse detected → revoke all sessions for user (force re-login)
```

### 5.2 Users

**GET /users/me**
```
Authorization: Bearer <jwt>
Response 200: full user object including stats
{
  "id": "...",
  "display_name": "SpeedDemon",
  "faction": "neon",
  "avatar_url": "...",
  "account_level": 12,
  "account_xp": 45000,
  "runner_tier": "free",
  "stats": {
    "total_runs": 87,
    "total_distance_m": 423000,
    "total_territory_points": 15200,
    "total_runner_points": 38400,
    "season_tier": "sprinter",
    "season_runner_points": 8200
  }
}
```

**PATCH /users/me**
```json
// Request (all fields optional)
{
  "display_name": "NewName",
  "avatar_upload": "multipart file or base64",
  "phone_hash": "sha256..."
}
```

**GET /users/:id**
```
// Public profile view
Response 200: { id, display_name, faction, avatar_url, account_level, stats (subset) }
```

**POST /users/contacts/match**
```json
// Request — client sends hashed contacts
{ "hashes": ["sha256_hash1", "sha256_hash2", ...] }
// Response 200
{
  "matches": [
    { "hash": "sha256_hash1", "user_id": "...", "display_name": "FriendOne", "faction": "neon" }
  ],
  "faction_counts": { "neon": 3, "umbra": 1 }
}
```

### 5.3 Runs

**POST /runs/start**
```json
// Request
{
  "lat": 40.7128,
  "lng": -74.0060,
  "social_run": true,
  "social_participants": ["friend_user_id_1", "friend_user_id_2"]
}
// Response 201
{
  "run_id": "...",
  "start_time": "2025-05-13T07:00:00Z",
  "social_run": true,
  "social_participants": [...]
}
// Validates: friend IDs are in friend list and within 20m (GPS proximity check)
```

**POST /runs/:id/end**
```json
// Request — GPS points array sent
{
  "gps_points": [
    { "timestamp": "2025-05-13T07:00:00Z", "lat": 40.7128, "lng": -74.0060, "altitude": 10,
      "speed": 3.2, "horizontal_accuracy": 5.0, "heart_rate": 145 },
    ...
  ],
  "health_source": "healthkit"  // if heart rate came from HealthKit
}
// Response 200
{
  "run_id": "...",
  "status": "completed",
  "closed_loop": true,
  "capture_mode": "polygon",
  "distance_m": 5230,
  "duration_s": 1620,
  "avg_pace_s_per_km": 310,
  "elevation_gain_m": 45,
  "territory_points": 340,
  "runner_points": 86,
  "hexes_captured": 34,
  "hexes_stolen": 8,
  "cheat_status": "clean"
}
```

**GET /runs**
```
Query: ?page=1&per_page=20&status=completed
Response 200: paginated list of run summaries
```

**GET /runs/:id**
```
Response 200: full run object including stats, point breakdown, territory hex list
```

**GET /runs/:id/gps**
```
Query: ?resolution=full (all points) or ?resolution=simplified (Douglas-Peucker, 10m epsilon)
Response 200:
{
  "polyline": "encoded_polyline...",
  "gps_points": [...]  // only for full resolution
}
```

### 5.4 Territory

**GET /territory**
```
Query:
  ?sw_lat=40.70&sw_lng=-74.02&ne_lat=40.75&ne_lng=-73.97  // bounding box
  ?include_hp=true                                           // include HP values (for hex detail view)

Response 200:
{
  "hexes": [
    {
      "h3_index": 608692844030885887,  // stored as int64, decoded client-side
      "center_lat": 40.7128,
      "center_lng": -74.0060,
      "owned_by": "neon",
      "hp": 3
    },
    ...
  ]
}
// Only returns hexes with an owner. Client renders neutral hexes as empty.
// Maximum 10,000 hexes per request (enforced). Client should request only visible viewport.
```

**GET /territory/stats**
```
Query: ?region=city|country|bbox|h3_resolution=&region_value=...
Response 200:
{
  "total_hexes": 150000,
  "claimed_hexes": 90000,
  "neon_hexes": 48000,
  "umbra_hexes": 42000,
  "neon_percentage": 53.3,
  "umbra_percentage": 46.7,
  "contested_zones": 120  // hexes flipped in last 24h
}
```

### 5.5 Friends

**POST /friends/add**
```json
// Request — one of: phone_hash, username, or user_id from contacts/scan
{ "user_id": "target-user-id" }
// Response 201: { status: "pending" }
// Side effect: push notification to target user
```

**GET /friends**
```
Response 200:
{
  "friends": [
    { "user_id": "...", "display_name": "RunnerX", "faction": "neon", "account_level": 8,
      "last_run_at": "...", "last_run_distance_m": 4200 }
  ]
}
```

**GET /friends/pending**
```
Response 200: list of incoming pending friend requests
```

**POST /friends/:id/accept**
```
Response 200: { status: "accepted" }
```

**GET /friends/feed**
```
Query: ?page=1&per_page=20
Response 200: chronologically ordered list of friends' recent completed runs
```

### 5.6 Leaderboards

**GET /leaderboard/team**
```
Query: ?region=country&region_value=US&sort=territory_points
Response 200: top 100 teams or regional factions ranked by territory control
```

**GET /leaderboard/runners**
```
Query: ?season=current|c20d29a1-...&region=city&region_value=NYC&sort=runner_points&page=1&per_page=20
Response 200: paginated runner leaderboard
{
  "entries": [
    { "rank": 1, "user_id": "...", "display_name": "TopRunner", "faction": "neon",
      "runner_points": 15200, "territory_points": 8400, "account_level": 23 }
  ],
  "current_user_rank": 42
}
```

**GET /leaderboard/friends**
```
Response 200: friends-only leaderboard, sorted by runner_points
```

### 5.7 Seasons

**GET /seasons/current**
```
Response 200:
{
  "id": "...",
  "name": "Season 1: Ignition",
  "start_date": "2025-06-01T00:00:00Z",
  "end_date": "2025-07-31T23:59:59Z",
  "tiers": {
    "rookie": 0, "runner": 1000, "sprinter": 5000, "elite": 15000, "legend": 30000
  },
  "my_progress": {
    "current_tier": "sprinter",
    "runner_points": 7200,
    "next_tier": "elite",
    "points_to_next": 7800
  }
}
```

**GET /seasons/:id/rewards**
```
Response 200: rewards breakdown for the completed season
```

### 5.8 WebSocket Protocol

**Connection:** `ws://host/ws?token=<jwt>`

**Client → Server messages:**
```json
// Subscribe to territory updates for current viewport
{ "type": "subscribe:territory", "bbox": { "sw": [40.70, -74.02], "ne": [40.75, -73.97] } }

// Subscribe to friend activity
{ "type": "subscribe:friends" }

// Broadcast live position during active run
{ "type": "run:position", "run_id": "...", "lat": 40.7128, "lng": -74.0060, "ts": "..." }

// Unsubscribe
{ "type": "unsubscribe:territory" }
```

**Server → Client messages:**
```json
// Territory hex changed
{
  "type": "territory:changed",
  "hexes": [{ "h3_index": ..., "owned_by": "neon", "hp": 4 }, ...]
}

// Friend started a run
{ "type": "friend:run_started", "user_id": "...", "display_name": "RunnerX" }

// Friend completed a run
{
  "type": "friend:run_completed",
  "user_id": "...", "display_name": "RunnerX",
  "distance_m": 5200, "territory_points": 340
}
```

---

## 6. Cameras & Schedulers

### 6.1 Daily HP Decay
- **Schedule:** Every day at 00:00 UTC
- **Logic:** Decrement `hp - 1` for all hexes where `hp > 1` AND `owned_by IS NOT NULL`
- **Batch size:** 10,000 hexes per transaction to avoid long-running locks
- **Implementation:** In-process cron (robfig/cron) or external process

### 6.2 Leaderboard Refresh
- **Schedule:** Every 15 minutes
- **Logic:** `REFRESH MATERIALIZED VIEW CONCURRENTLY leaderboard_regional`
- **Implementation:** In-process cron

### 6.3 Season Rollover
- **Manual trigger:** Admin endpoint or database migration
- **Logic:**
  1. Calculate final tiers for all season_participants
  2. Award badges, record rewards
  3. Create new season record
  4. Clear/reset season_participants for new season
- **Downstream:** Push notification to all users with their final tier and badge

### 6.4 Bot Scheduler
- **Schedule:** Every 15 minutes
- **Logic:** Run Bot Spawning Algorithm (Section 4.5)
- **Also triggered:** Immediately after a new user completes their first run

### 6.5 Backup Scheduler
- **Schedule:** Daily at 03:00 UTC
- **Logic:** `pg_dump` → GZIP → upload to rustfs bucket → delete backups older than 30 days

### 6.6 User Data Cleanup
- **Schedule:** Daily at 02:00 UTC
- **Logic:** Hard-delete users where `deleted_at < NOW() - INTERVAL '30 days'` and anonymize associated data

---

## 7. Security Implementation

### 7.1 Authentication Flow
```
1. User registers/logs in → server returns access_token (JWT, 15 min TTL) + refresh_token (opaque UUID, 30 day TTL)
2. Client stores access_token in memory, refresh_token in flutter_secure_storage
3. Every API request: Authorization: Bearer <access_token>
4. On 401: client calls POST /auth/refresh with refresh_token
5. Server issues new access_token + rotates refresh_token
6. If a refresh token is reused after its rotation → all user sessions revoked (token theft detection)
```

### 7.2 Password Requirements
- Minimum 8 characters
- Require at least one letter and one number
- bcrypt cost factor: 12

### 7.3 Rate Limiting (Per IP + Per User)
```
/auth/register:    5 per hour
/auth/login:       20 per hour
/auth/refresh:     10 per minute
/runs/start:       10 per hour
/runs/end:         10 per hour
/territory:        30 per minute
/friends/add:      20 per hour
```

### 7.4 Data Encryption
- GPS tracks: application-level AES-256-GCM encryption before DB insert. Per-run encryption key derived from user_id + run_id.
- Phone hashes: SHA-256 with per-server pepper (stored in env). Only used for matching, never reversible.
- All secrets (DB password, JWT secret, Mapbox token, etc.) via environment variables or Docker secrets.

### 7.5 Mapbox Token Security
- Mapbox access token stored server-side only
- nginx location `/mapbox/` proxies to `https://api.mapbox.com/` appending the token
- Flutter app uses `https://host/mapbox/styles/v1/...` instead of direct Mapbox URLs
- Rate limits enforced at nginx level (per user or global)

---

## 8. Monitoring & Observability

### 8.1 Health Checks
```
GET /health        → 200 { "status": "ok", "db": "connected", "redis": "connected" }
GET /health/ready  → 200 when all dependencies ready
```

### 8.2 Metrics (Exposed via /metrics — Prometheus format)
- `terrarun_runs_total{status}` — counter
- `terrarun_territory_changes_total{faction}` — counter
- `terrarun_active_users_gauge` — gauge, active in last 24h
- `terrarun_bots_active_gauge{region}` — gauge
- `terrarun_ws_connections_gauge` — gauge
- `terrarun_api_request_duration_seconds` — histogram
- `terrarun_hexes_total{faction}` — gauge

### 8.3 Logging
- Structured JSON logging (zerolog)
- Log levels: debug, info, warn, error
- Key events: registration, login, run start/end, territory capture, bot spawn, cheat flag, error

---

## 9. Testing Strategy

### 9.1 Unit Tests (Go)
- All service-layer logic tested with mocked repositories
- Cheat detection algorithm: exhaustive test cases (clean runs, speed violations, teleportation, HR mismatch)
- H3 capture engine: known polygons → known hex sets
- HP arithmetic: full state machine transitions
- Point calculator: combinatorial scenarios

### 9.2 Integration Tests (Go)
- HTTP handler tests with test database (testcontainers-go)
- WebSocket connection lifecycle tests
- Bot scheduler full-flow test
- Rate limiter tests

### 9.3 End-to-End Tests
- `POST /auth/register` → get token → `POST /runs/start` → `POST /runs/end` with synthetic GPS data → verify territory hexes created → verify points → verify leaderboard
- Bot seeding → verify hex islands created with correct 60/40 split
- HP decay → verify -1 per day, floor at 1
- Friend flow: add → accept → feed → social run → multiplier

### 9.4 Flutter Tests
- Widget tests for key screens (map, run summary, leaderboard)
- Bloc/Cubit unit tests for state management
- Integration tests for critical user flows (registration through first run)

### 9.5 Load Test (pre-launch)
- Simulate 10,000 concurrent users using k6 or vegeta
- Focus: territory queries (most expensive), WebSocket scalability, HP decay performance on large hex table

---

## 10. Data Flow Diagrams

### 10.1 Run Lifecycle (Sequential)
```
User taps "Start"
  → Flutter: /runs/start (lat, lng, social participants)
  → Go: validate friends within 20m, create run record (status=active)
  → Flutter: GPS tracking starts, live position to WebSocket

User taps "End"
  → Flutter: /runs/end (gps_points array)
  → Go:
      1. Cheat detection → status
      2. If clean: build polygon → extract H3 cells → apply capture → compute points
      3. Update run record
      4. Update user XP/level
      5. Broadcast territory changes to subscribers
      6. Push notify friends
  → Flutter: receive run summary, navigate to summary screen
```

### 10.2 Map View Load
```
User opens map
  → Flutter: GET /territory?bbox=current_viewport
  → Go: query hexes WHERE h3_index intersects bbox AND owned_by IS NOT NULL
  → Flutter: render hex overlay on Mapbox map
  → Flutter: WebSocket subscribe to territory changes for this bbox
  → Live updates: server pushes hex changes as they happen within viewport
```

### 10.3 Bot Seeding Flow
```
User registration → /users POST (includes lat, lng)
  → Go: after user creation, trigger bot_seeder in background goroutine
  → Check existing coverage in 100km radius
  → If occupancy < 60%:
      - Classify cells (urban/suburban/rural/natural)
      - Generate islands
      - Insert hexes with 50/50 faction split
      - Create bot profiles
  → Notify client: seeding_complete event via WebSocket
```

### 10.4 Friend Contact Matching
```
Client requests contacts permission → device returns contacts
Client SHA-256 hashes each phone number locally
Client sends hashes → POST /users/contacts/match
Server queries: users WHERE phone_hash IN (hashes) AND deleted_at IS NULL
Server returns: matched user IDs, display names, factions + aggregated counts
Client shows: "3 friends are Umbra, 1 is Neon"
User taps a faction → client shows friend list filtered by faction
```

---

## 11. Push Notification Flows

| Trigger | Recipient | Payload |
|---|---|---|
| Friend request received | Target user | "{Name} wants to be your running buddy" |
| Friend request accepted | Requester | "{Name} accepted your friend request" |
| Friend started a run | All friends | "{Name} just started a run — join them?" |
| Friend completed run | All friends | "{Name} ran {distance}km and captured {n} hexes" |
| Season ending soon | All users | "Season ends in 3 days! Climb the leaderboard!" |
| Season ended | All users | "Season over! You finished as {tier}. Claim your badge!" |
| Territory under attack | Users whose territory was stolen | "{Name} is taking your turf! Run to defend!" |

Implementation: Firebase Cloud Messaging for Android, Apple Push Notification service for iOS, dispatched from Go backend via firebase-admin-go SDK.

---

## 12. Infrastructure Details

### 12.1 docker-compose.yml (Development)

```yaml
services:
  postgres:
    image: timescale/timescaledb:latest-pg16
    environment:
      POSTGRES_DB: terrarun
      POSTGRES_USER: terrarun
      POSTGRES_PASSWORD: terrarun_dev
    ports: ["5432:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  rustfs:
    image: rustfs/rustfs:latest
    ports: ["9000:9000", "9001:9001"]
    volumes: [rustfs_data:/data]

  api:
    build: ./backend
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://terrarun:terrarun_dev@postgres:5432/terrarun?sslmode=disable
      REDIS_URL: redis://redis:6379
      RUSTFS_ENDPOINT: http://rustfs:9000
      JWT_SECRET: dev_secret_change_in_prod
    depends_on: [postgres, redis, rustfs]
    volumes: [./backend:/app]  # hot reload

volumes:
  pgdata:
  rustfs_data:
```

### 12.2 docker-compose.prod.yml

```yaml
services:
  nginx:
    image: nginx:alpine
    ports: ["80:80", "443:443"]
    volumes: [./nginx.conf:/etc/nginx/nginx.conf, certs:/etc/letsencrypt]
    depends_on: [api]

  postgres:
    # same as dev but with resource limits
    deploy:
      resources:
        limits: { cpus: '2', memory: 4G }
        reservations: { cpus: '1', memory: 2G }

  api:
    restart: unless-stopped
    # read-only filesystem, no source mount
    environment: [...secrets...]
```

### 12.3 Deploy Script (deploy.sh)

```bash
#!/bin/bash
set -e
HOST=$1
rsync -avz docker-compose.prod.yml nginx.conf backend/Dockerfile $HOST:/opt/terrarun/
ssh $HOST "cd /opt/terrarun && docker compose -f docker-compose.prod.yml pull && docker compose -f docker-compose.prod.yml up -d && docker compose -f docker-compose.prod.yml exec api ./migrate up"
```

---

## 13. Open / Tuning Parameters

| Parameter | Default | Notes |
|---|---|---|
| Loop closure radius | 100 m | Configurable per-region SQL setting |
| Path capture enabled | true | Global runtime toggle (no restart needed) |
| Path capture buffer radius | 25 m | Buffer distance from GPS track for hex capture in open runs |
| Hex HP cap | 10 | Prevents unassailable fortresses |
| Hex HP daily decay | 1 | Floor at 1 |
| Base territory points (own hex) | 10 | Tune based on beta analytics |
| Territory point bonus (steal) | +50% | 15 pts |
| Runner point per 100m | 1 | Linear accumulation |
| Runner point per 10m elevation | 5 | Rewards hilly runs |
| XP conversion ratio | TBD | Runner points → XP multiplier |
| Social multiplier base | +25% | With 1+ friends |
| Social multiplier per extra friend | +10% | Capped at +50% total |
| Bots per faction (per region) | 10 minus real users | Minimum 0 |
| Bot loop distance default | 5 km | Falls back when no real user data |
| Bot active hours | 6 AM – 10 PM | Local time |
| Bot peak hours | 7 AM, 6 PM | 100% spawn probability |
| Bot city spawn cap | 1 per 30 sec | Prevents bot flood |
| Bot hex density cap | 2 per 1 km hex | Prevents bot clumping |
| Onboarding seeding radius | 100 km | Centered on new user |
| Seeding max claimed | 60% | 40% hexes left neutral |
| Season duration | 2 months | Configurable |
| Access token TTL | 15 min | JWT |
| Refresh token TTL | 30 days | Opaque UUID |
| Run soft-flag distance threshold | 50 km | Manual review |
| Speed hard reject | 35 km/h (segment), 20 km/h (avg > 2km) | Motorized |
| GPS teleportation reject | 500 m between consecutive points | Cheater pausing |
| Low GPS accuracy threshold | 20 m (30% of points) | Soft flag |
| Bot phase-out threshold | 20 real users in 10km | Full deactivation |
