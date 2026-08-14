# TerraRun — Low-Level Design (LLD)

**Date:** 2025-05-13
**Version:** 1.0
**Depends on:** [HLD](./2025-05-13-terrarun-hld.md) → [Design Spec](./2025-05-13-terrarun-design.md)

---

## 1. Configuration

### 1.1 Environment Variables (backend)

```go
// internal/config/config.go
package config

import "time"

type Config struct {
    // Server
    Port            string        `envconfig:"PORT" default:"8080"`
    ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"10s"`
    WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
    ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"15s"`

    // Database
    DatabaseURL     string        `envconfig:"DATABASE_URL" required:"true"`
    MaxDBConns      int           `envconfig:"MAX_DB_CONNS" default:"25"`
    MinDBConns      int           `envconfig:"MIN_DB_CONNS" default:"5"`

    // Redis
    RedisURL        string        `envconfig:"REDIS_URL" required:"true"`

    // Storage (rustfs S3-compatible)
    StorageEndpoint  string `envconfig:"STORAGE_ENDPOINT" required:"true"`
    StorageAccessKey string `envconfig:"STORAGE_ACCESS_KEY" required:"true"`
    StorageSecretKey string `envconfig:"STORAGE_SECRET_KEY" required:"true"`
    StorageBucket    string `envconfig:"STORAGE_BUCKET" default:"terrarun"`
    StorageUseSSL    bool   `envconfig:"STORAGE_USE_SSL" default:"false"`

    // Auth
    JWTSecret         string        `envconfig:"JWT_SECRET" required:"true"`
    AccessTokenTTL    time.Duration `envconfig:"ACCESS_TOKEN_TTL" default:"15m"`
    RefreshTokenTTL   time.Duration `envconfig:"REFRESH_TOKEN_TTL" default:"720h"` // 30 days
    BcryptCost        int           `envconfig:"BCRYPT_COST" default:"12"`
    PhoneHashPepper   string        `envconfig:"PHONE_HASH_PEPPER" required:"true"`

    // OAuth
    GoogleClientID string `envconfig:"GOOGLE_CLIENT_ID" required:"true"`
    AppleTeamID    string `envconfig:"APPLE_TEAM_ID" required:"true"`
    AppleKeyID     string `envconfig:"APPLE_KEY_ID" required:"true"`
    ApplePrivateKey string `envconfig:"APPLE_PRIVATE_KEY" required:"true"`

    // Mapbox
    MapboxToken string `envconfig:"MAPBOX_TOKEN" required:"true"`

    // Firebase (push notifications)
    FirebaseCredsFile string `envconfig:"FIREBASE_CREDS_FILE"`

    // Territory
    LoopClosureRadiusM     float64 `envconfig:"LOOP_CLOSURE_RADIUS_M" default:"100"`
    PathCaptureEnabled     bool    `envconfig:"PATH_CAPTURE_ENABLED" default:"true"`
    PathCaptureBufferM     float64 `envconfig:"PATH_CAPTURE_BUFFER_M" default:"25"`
    HexHPCap               int     `envconfig:"HEX_HP_CAP" default:"10"`
    HexHPDecay             int     `envconfig:"HEX_HP_DECAY" default:"1"`
    TerritoryBasePoints    int     `envconfig:"TERRITORY_BASE_POINTS" default:"10"`
    TerritoryStealBonusPct float64 `envconfig:"TERRITORY_STEAL_BONUS_PCT" default:"0.5"`

    // Points
    RunnerPointPer100m      float64 `envconfig:"RUNNER_POINT_PER_100M" default:"1.0"`
    RunnerPointPer10mElev   float64 `envconfig:"RUNNER_POINT_PER_10M_ELEV" default:"5.0"`
    TerritoryPointSharePct  float64 `envconfig:"TERRITORY_POINT_SHARE_PCT" default:"0.1"`
    SocialBaseMultiplier    float64 `envconfig:"SOCIAL_BASE_MULTIPLIER" default:"0.25"`
    SocialExtraPerFriend    float64 `envconfig:"SOCIAL_EXTRA_PER_FRIEND" default:"0.10"`
    SocialMultiplierCap     float64 `envconfig:"SOCIAL_MULTIPLIER_CAP" default:"0.50"`
    XPConversionRatio       float64 `envconfig:"XP_CONVERSION_RATIO" default:"1.0"`

    // Cheat Detection
    MaxSegmentSpeedKMH      float64 `envconfig:"MAX_SEGMENT_SPEED_KMH" default:"35"`
    MaxAvgSpeedKMH          float64 `envconfig:"MAX_AVG_SPEED_KMH" default:"20"`
    MinDistanceForAvgCheckM float64 `envconfig:"MIN_DISTANCE_FOR_AVG_CHECK_M" default:"2000"`
    MaxGPSJumpM             float64 `envconfig:"MAX_GPS_JUMP_M" default:"500"`
    FlagAvgSpeedKMH         float64 `envconfig:"FLAG_AVG_SPEED_KMH" default:"15"`
    FlagDistanceM           float64 `envconfig:"FLAG_DISTANCE_M" default:"50000"`
    FlagLowAccuracyM        float64 `envconfig:"FLAG_LOW_ACCURACY_M" default:"20"`
    FlagLowAccuracyRatio    float64 `envconfig:"FLAG_LOW_ACCURACY_RATIO" default:"0.3"`

    // Bots
    BotsPerFaction        int     `envconfig:"BOTS_PER_FACTION" default:"10"`
    BotSeedingRadiusKM    float64 `envconfig:"BOT_SEEDING_RADIUS_KM" default:"100"`
    BotActiveRadiusKM     float64 `envconfig:"BOT_ACTIVE_RADIUS_KM" default:"10"`
    BotDefaultDistanceM   float64 `envconfig:"BOT_DEFAULT_DISTANCE_M" default:"5000"`
    BotCitySpawnIntervalS int     `envconfig:"BOT_CITY_SPAWN_INTERVAL_S" default:"30"`
    BotMaxPerHex          int     `envconfig:"BOT_MAX_PER_HEX" default:"2"`
    BotPhaseOutThreshold  int     `envconfig:"BOT_PHASE_OUT_THRESHOLD" default:"20"`
    BotSeedingMaxClaimed  float64 `envconfig:"BOT_SEEDING_MAX_CLAIMED" default:"0.60"`

    // Season
    SeasonDurationDays int `envconfig:"SEASON_DURATION_DAYS" default:"60"`

    // Rate Limiting
    RateLimitAuthRegister int `envconfig:"RATE_LIMIT_AUTH_REGISTER" default:"5"`
    RateLimitAuthLogin    int `envconfig:"RATE_LIMIT_AUTH_LOGIN" default:"20"`
    RateLimitAuthRefresh  int `envconfig:"RATE_LIMIT_AUTH_REFRESH" default:"10"`
    RateLimitRunsStart    int `envconfig:"RATE_LIMIT_RUNS_START" default:"10"`
    RateLimitRunsEnd      int `envconfig:"RATE_LIMIT_RUNS_END" default:"10"`
    RateLimitTerritory    int `envconfig:"RATE_LIMIT_TERRITORY" default:"30"`
    RateLimitFriendsAdd   int `envconfig:"RATE_LIMIT_FRIENDS_ADD" default:"20"`
}

func Load() (*Config, error) {
    var cfg Config
    err := envconfig.Process("", &cfg)
    return &cfg, err
}
```

### 1.2 Runtime Config Toggles (Redis-backed)

Some config values support hot-reload via Redis without restart:

```go
// internal/config/runtime.go
package config

// Keys in Redis that override env defaults at runtime
const (
    KeyPathCaptureEnabled = "config:path_capture_enabled"  // "true" / "false"
    KeyPathCaptureBuffer  = "config:path_capture_buffer"   // float string
)
```

---

## 2. Go Domain Types (`internal/domain/`)

```go
// internal/domain/user.go
package domain

type Faction string
const (
    FactionNeon  Faction = "neon"
    FactionUmbra Faction = "umbra"
)

type RunnerTier string
const (
    TierFree    RunnerTier = "free"
    TierRunner  RunnerTier = "runner"
    TierCaptain RunnerTier = "captain"
)

type User struct {
    ID           uuid.UUID
    Email        string
    PasswordHash *string     // nil for OAuth-only
    DisplayName  string
    PhoneHash    *string
    AvatarURL    *string
    Faction      *Faction    // nil until chosen
    AccountLevel int
    AccountXP    int64
    GoogleID     *string
    AppleID      *string
    RunnerTier   RunnerTier
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletedAt    *time.Time
}

type Session struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    RefreshToken string
    DeviceInfo   *string
    ExpiresAt    time.Time
    CreatedAt    time.Time
    RevokedAt    *time.Time
}
```

```go
// internal/domain/run.go
package domain

type RunStatus string
const (
    RunStatusActive    RunStatus = "active"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFlagged   RunStatus = "flagged"
    RunStatusRejected  RunStatus = "rejected"
)

type CaptureMode string
const (
    CaptureModePolygon CaptureMode = "polygon"
    CaptureModePath    CaptureMode = "path"
)

type HealthSource string
const (
    HealthSourceKit     HealthSource = "healthkit"
    HealthSourceConnect HealthSource = "healthconnect"
)

type Run struct {
    ID                   uuid.UUID
    UserID               uuid.UUID
    StartTime            time.Time
    EndTime              *time.Time
    Status               RunStatus
    DistanceM            *float64
    ElevationGainM       *float64
    AvgPaceSPerKM        *float64
    MaxSpeedKMH          *float64
    AvgHRBPM             *int
    MaxHRBPM             *int
    Calories             *int
    ClosedLoop           *bool
    CaptureMode          *CaptureMode
    LoopSnapDistanceM    *float64
    PathBufferRadiusM    *float64
    Polyline             *string
    TerritoryPoints      int
    RunnerPoints         int
    SocialRun            bool
    SocialLeader         *uuid.UUID
    SocialParticipants   []uuid.UUID
    HealthSource         *HealthSource
    CreatedAt            time.Time
}

type GPSPoint struct {
    RunID              uuid.UUID
    Timestamp          time.Time
    Lat                float64
    Lng                float64
    Altitude           *float64
    Speed              *float64 // m/s
    HorizontalAccuracy *float64
    HeartRate          *int
}
```

```go
// internal/domain/territory.go
package domain

import "github.com/uber/h3-go/v4"

type Hex struct {
    H3Index       h3.H3Index
    OwnedBy       *Faction
    HP            int  // 1-10
    CapturedByID  *uuid.UUID
    CapturedAt    time.Time
    LastDecayedAt time.Time
    IsNatural     bool
}

type TerritoryChange struct {
    ID            int64
    H3Index       h3.H3Index
    RunID         *uuid.UUID
    PreviousOwner *Faction
    NewOwner      *Faction
    HPBefore      int
    HPAfter       int
    ChangedAt     time.Time
}

type TerritoryStats struct {
    TotalHexes      int64   `json:"total_hexes"`
    ClaimedHexes    int64   `json:"claimed_hexes"`
    NeonHexes       int64   `json:"neon_hexes"`
    UmbraHexes      int64   `json:"umbra_hexes"`
    NeonPercentage  float64 `json:"neon_percentage"`
    UmbraPercentage float64 `json:"umbra_percentage"`
    ContestedZones  int64   `json:"contested_zones"`
}
```

```go
// internal/domain/friend.go
package domain

type FriendStatus string
const (
    FriendPending  FriendStatus = "pending"
    FriendAccepted FriendStatus = "accepted"
    FriendBlocked  FriendStatus = "blocked"
)

type Friend struct {
    UserID    uuid.UUID
    FriendID  uuid.UUID
    Status    FriendStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

```go
// internal/domain/season.go
package domain

type Season struct {
    ID          uuid.UUID
    Name        string
    StartDate   time.Time
    EndDate     time.Time
    TiersConfig TiersConfig
    IsActive    bool
}

type SeasonParticipant struct {
    UserID          uuid.UUID
    SeasonID        uuid.UUID
    RunnerPoints    int
    TerritoryPoints int
    CurrentTier     string
    FinalTier       *string
    BadgeAwarded    bool
}

type TiersConfig struct {
    Rookie   int `json:"rookie"`
    Runner   int `json:"runner"`
    Sprinter int `json:"sprinter"`
    Elite    int `json:"elite"`
    Legend   int `json:"legend"`
}
```

```go
// internal/domain/bot.go
package domain

type Bot struct {
    ID             uuid.UUID
    DisplayName    string
    Faction        Faction
    AvatarSeed     *string
    HomeLat        float64
    HomeLng        float64
    ActiveRadiusKM float64
    AvgDistanceM   float64
    TotalRuns      int
    CreatedAt      time.Time
    DeactivatedAt  *time.Time
}
```

---

## 3. Repository Layer

### 3.1 SQL Queries (by module)

All queries use `database/sql` with `pgx` driver or `sqlx`. Each repo is an interface defined alongside the service.

```go
// internal/run/repository.go (interface)
type Repository interface {
    Create(ctx context.Context, run *domain.Run) error
    Update(ctx context.Context, run *domain.Run) error
    GetByID(ctx context.Context, id uuid.UUID) (*domain.Run, error)
    GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]domain.Run, int, error)
    InsertGPSPoints(ctx context.Context, points []domain.GPSPoint) error
    GetGPSPoints(ctx context.Context, runID uuid.UUID, simplify bool) ([]domain.GPSPoint, error)
}
```

```go
// internal/run/repository_postgres.go (implementation)
package run

type postgresRepo struct { db *sqlx.DB }

func NewPostgresRepo(db *sqlx.DB) Repository { return &postgresRepo{db} }

const createRunSQL = `
INSERT INTO runs (id, user_id, start_time, status, social_run, social_leader, social_participants)
VALUES ($1, $2, $3, 'active', $4, $5, $6)
`

const updateRunSQL = `
UPDATE runs SET
    end_time = $2, status = $3, distance_m = $4, elevation_gain_m = $5,
    avg_pace_s_per_km = $6, max_speed_kmh = $7, avg_hr_bpm = $8,
    max_hr_bpm = $9, calories = $10, closed_loop = $11, capture_mode = $12,
    loop_snap_distance_m = $13, path_buffer_radius_m = $14, polyline = $15,
    territory_points = $16, runner_points = $17, health_source = $18
WHERE id = $1
`

const getRunByIDSQL = `
SELECT id, user_id, start_time, end_time, status, distance_m, elevation_gain_m,
       avg_pace_s_per_km, max_speed_kmh, avg_hr_bpm, max_hr_bpm, calories,
       closed_loop, capture_mode, loop_snap_distance_m, path_buffer_radius_m,
       polyline, territory_points, runner_points, social_run, social_leader,
       social_participants, health_source, created_at
FROM runs WHERE id = $1
`

const getRunsByUserSQL = `
SELECT ... FROM runs WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
`

const countRunsByUserSQL = `SELECT COUNT(*) FROM runs WHERE user_id = $1`

const insertGPSPointsSQL = `
INSERT INTO gps_points (run_id, timestamp, lat, lng, altitude, speed, horizontal_accuracy, heart_rate)
SELECT $1, unnest($2::timestamptz[]), unnest($3::double precision[]), unnest($4::double precision[]),
       unnest($5::double precision[]), unnest($6::double precision[]), unnest($7::double precision[]),
       unnest($8::integer[])
`

const getGPSPointsSQL = `
SELECT run_id, timestamp, lat, lng, altitude, speed, horizontal_accuracy, heart_rate
FROM gps_points WHERE run_id = $1 ORDER BY timestamp
`

// Implementation methods
func (r *postgresRepo) Create(ctx context.Context, run *domain.Run) error {
    return r.db.QueryRowContext(ctx, createRunSQL,
        run.ID, run.UserID, run.StartTime, run.SocialRun, run.SocialLeader, pq.Array(run.SocialParticipants),
    ).Err()
}

func (r *postgresRepo) Update(ctx context.Context, run *domain.Run) error {
    return r.db.QueryRowContext(ctx, updateRunSQL,
        run.ID, run.EndTime, run.Status, run.DistanceM, run.ElevationGainM,
        run.AvgPaceSPerKM, run.MaxSpeedKMH, run.AvgHRBPM, run.MaxHRBPM,
        run.Calories, run.ClosedLoop, run.CaptureMode, run.LoopSnapDistanceM,
        run.PathBufferRadiusM, run.Polyline, run.TerritoryPoints,
        run.RunnerPoints, run.HealthSource,
    ).Err()
}

func (r *postgresRepo) InsertGPSPoints(ctx context.Context, points []domain.GPSPoint) error {
    // Batch insert via unnest for performance (single round trip)
    n := len(points)
    tss := make([]time.Time, n)
    lats := make([]float64, n)
    lngs := make([]float64, n)
    alts := make([]*float64, n)
    speeds := make([]*float64, n)
    accs := make([]*float64, n)
    hrs := make([]*int, n)
    for i, p := range points {
        tss[i] = p.Timestamp
        lats[i] = p.Lat
        lngs[i] = p.Lng
        alts[i] = p.Altitude
        speeds[i] = p.Speed
        accs[i] = p.HorizontalAccuracy
        hrs[i] = p.HeartRate
    }
    return r.db.QueryRowContext(ctx, insertGPSPointsSQL,
        points[0].RunID, pq.Array(tss), pq.Array(lats), pq.Array(lngs),
        pq.Array(alts), pq.Array(speeds), pq.Array(accs), pq.Array(hrs),
    ).Err()
}
```

### 3.2 Territory Repository

```go
// internal/territory/repository.go
type Repository interface {
    // Hex CRUD
    GetHex(ctx context.Context, h3Index h3.H3Index) (*domain.Hex, error)
    UpsertHex(ctx context.Context, hex *domain.Hex) error
    GetHexesInBounds(ctx context.Context, swLat, swLng, neLat, neLng float64) ([]domain.Hex, error)
    
    // Batch operations
    BatchUpsertHexes(ctx context.Context, hexes []domain.Hex) error
    BatchDecayHP(ctx context.Context, limit int) (int, error)
    
    // Change history
    RecordChange(ctx context.Context, change *domain.TerritoryChange) error
    GetRecentChanges(ctx context.Context, since time.Time) ([]domain.TerritoryChange, error)
    
    // Stats
    GetRegionStats(ctx context.Context, regionType, regionValue string) (*domain.TerritoryStats, error)
    GetUserCapturedHexes(ctx context.Context, userID uuid.UUID, runID uuid.UUID) ([]domain.Hex, error)
}
```

```go
// internal/territory/repository_postgres.go (key queries)

const upsertHexSQL = `
INSERT INTO hexes (h3_index, owned_by, hp, captured_by, captured_at, last_decayed_at, is_natural)
VALUES ($1, $2, $3, $4, $5, $5, $6)
ON CONFLICT (h3_index) DO UPDATE SET
    owned_by = EXCLUDED.owned_by,
    hp = EXCLUDED.hp,
    captured_by = EXCLUDED.captured_by,
    captured_at = CASE WHEN $2 IS NOT NULL THEN EXCLUDED.captured_at ELSE hexes.captured_at END,
    is_natural = CASE WHEN $6 THEN true ELSE hexes.is_natural END
`

const getHexesInBoundsSQL = `
SELECT h3_index, owned_by, hp, captured_by, captured_at, last_decayed_at, is_natural
FROM hexes
WHERE owned_by IS NOT NULL
  AND h3_cell_to_lat(h3_index) BETWEEN $1 AND $3
  AND h3_cell_to_lng(h3_index) BETWEEN $2 AND $4
LIMIT 10000
`

const decayHPSQL = `
UPDATE hexes
SET hp = hp - 1, last_decayed_at = NOW()
WHERE h3_index IN (
    SELECT h3_index FROM hexes
    WHERE owned_by IS NOT NULL AND hp > 1
    LIMIT $1
)
RETURNING h3_index
`

// batchUpsert uses pgx CopyFrom for maximum throughput
func (r *postgresRepo) BatchUpsertHexes(ctx context.Context, hexes []domain.Hex) error {
    tx, _ := r.pool.Begin(ctx)
    defer tx.Rollback(ctx)
    
    // COPY protocol for bulk insert
    _, err := tx.CopyFrom(
        pgx.Identifier{"hexes"},
        []string{"h3_index", "owned_by", "hp", "captured_by", "captured_at", "last_decayed_at"},
        pgx.CopyFromSlice(len(hexes), func(i int) ([]any, error) {
            h := hexes[i]
            return []any{int64(h.H3Index), string(*h.OwnedBy), h.HP, h.CapturedByID, h.CapturedAt, h.CapturedAt}, nil
        }),
    )
    if err != nil {
        tx.Rollback(ctx)
        // Fall back to regular upsert on conflict
        return r.batchUpsertFallback(ctx, hexes)
    }
    return tx.Commit(ctx)
}
```

### 3.3 Redis Repository (Leaderboards, Presence, Rate Limiting)

```go
// internal/leaderboard/redis_repo.go

// Active user presence (set with TTL)
const presenceKey = "presence:user:%s"          // SETEX, TTL 5 min
func SetUserActive(ctx context.Context, rdb *redis.Client, userID uuid.UUID) error {
    return rdb.Set(ctx, fmt.Sprintf(presenceKey, userID), time.Now().Unix(), 5*time.Minute).Err()
}

// Friend activity feed (sorted set)
const friendFeedKey = "feed:friends:%s"
func AddToFriendFeed(ctx context.Context, rdb *redis.Client, userID uuid.UUID, entry string) error {
    return rdb.ZAdd(ctx, fmt.Sprintf(friendFeedKey, userID), redis.Z{
        Score:  float64(time.Now().Unix()),
        Member: entry,
    }).Err()
}
func GetFriendFeed(ctx context.Context, rdb *redis.Client, userID uuid.UUID, page, perPage int64) ([]string, error) {
    start := (page - 1) * perPage
    stop := start + perPage - 1
    return rdb.ZRevRange(ctx, fmt.Sprintf(friendFeedKey, userID), start, stop).Result()
}

// Team leaderboard (sorted set)
const teamLeaderboardKey = "leaderboard:team:%s:%s"  // region_type:region_value
func UpdateTeamScore(ctx context.Context, rdb *redis.Client, regionKey string, faction string, delta int64) error {
    return rdb.ZIncrBy(ctx, fmt.Sprintf(teamLeaderboardKey, "global", regionKey), float64(delta), faction).Err()
}

// Runner leaderboard
const runnerLeaderboardKey = "leaderboard:runners:%s:%s"
func UpdateRunnerScore(ctx context.Context, rdb *redis.Client, key, userID string, points int64) error {
    return rdb.ZIncrBy(ctx, key, float64(points), userID).Err()
}

// Rate limiting (token bucket or sliding window)
func CheckRateLimit(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) (bool, error) {
    now := time.Now().UnixNano()
    windowStart := now - window.Nanoseconds()
    
    pipe := rdb.Pipeline()
    pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprint(windowStart))
    pipe.ZCard(ctx, key)
    pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
    pipe.Expire(ctx, key, window)
    
    cmds, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    count := cmds[1].(*redis.IntCmd).Val()
    return int(count) < limit, nil
}
```

---

## 4. Service Layer

### 4.1 Auth Service

```go
// internal/auth/service.go
type Service struct {
    userRepo user.Repository
    sessRepo Repository // sessions
    cfg      *config.Config
}

// Register creates a user, saves hashed password (or nil for OAuth), creates session
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
    // Validate email uniqueness
    existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
    if existing != nil { return nil, ErrEmailTaken }

    var pwHash *string
    if req.Password != nil {
        h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), s.cfg.BcryptCost)
        if err != nil { return nil, fmt.Errorf("hash password: %w", err) }
        s := string(h)
        pwHash = &s
    }

    user := domain.User{
        ID:          uuid.New(),
        Email:       req.Email,
        PasswordHash: pwHash,
        DisplayName: req.DisplayName,
        PhoneHash:   req.PhoneHash,
        Faction:     req.Faction,
        RunnerTier:  domain.TierFree,
    }
    if err := s.userRepo.Create(ctx, &user); err != nil { return nil, err }

    return s.createAuthResponse(ctx, user)
}

// Login validates credentials and returns tokens
func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
    user, err := s.userRepo.GetByEmail(ctx, req.Email)
    if err != nil { return nil, ErrInvalidCredentials }
    
    if user.PasswordHash == nil { return nil, ErrOAuthOnlyAccount }
    
    if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
        return nil, ErrInvalidCredentials
    }
    return s.createAuthResponse(ctx, *user)
}

// GoogleLogin verifies Google ID token, creates or finds user
func (s *Service) GoogleLogin(ctx context.Context, req GoogleLoginRequest) (*AuthResponse, error) {
    payload, err := verifyGoogleIDToken(ctx, req.IDToken, s.cfg.GoogleClientID)
    if err != nil { return nil, ErrInvalidToken }

    user, err := s.userRepo.GetByGoogleID(ctx, payload.Subject)
    if err == ErrNotFound {
        user = &domain.User{
            ID:         uuid.New(),
            Email:      payload.Email,
            DisplayName: payload.Name,
            GoogleID:   &payload.Subject,
            Faction:    req.Faction,
            RunnerTier: domain.TierFree,
        }
        if err := s.userRepo.Create(ctx, user); err != nil { return nil, err }
    }
    return s.createAuthResponse(ctx, *user)
}

// Refresh rotates the refresh token and issues a new access token
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResponse, error) {
    sess, err := s.sessRepo.GetByRefreshToken(ctx, refreshToken)
    if err != nil { return nil, ErrInvalidToken }
    
    if sess.RevokedAt != nil {
        // Token reuse detected — revoke ALL sessions for this user
        s.sessRepo.RevokeAllForUser(ctx, sess.UserID)
        return nil, ErrTokenReused
    }
    
    // Rotate: revoke old, create new
    s.sessRepo.Revoke(ctx, sess.ID)
    
    user, _ := s.userRepo.GetByID(ctx, sess.UserID)
    return s.createSession(ctx, *user)
}

func (s *Service) createAuthResponse(ctx context.Context, user domain.User) (*AuthResponse, error) {
    accessToken, err := s.generateAccessToken(user)
    if err != nil { return nil, err }
    
    refreshToken := uuid.New().String()
    sess := domain.Session{
        ID:           uuid.New(),
        UserID:       user.ID,
        RefreshToken: refreshToken,
        ExpiresAt:    time.Now().Add(s.cfg.RefreshTokenTTL),
    }
    if err := s.sessRepo.Create(ctx, &sess); err != nil { return nil, err }
    
    return &AuthResponse{
        User:         user,
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}

func (s *Service) generateAccessToken(user domain.User) (string, error) {
    claims := jwt.MapClaims{
        "sub":    user.ID.String(),
        "email":  user.Email,
        "faction": user.Faction,
        "tier":   user.RunnerTier,
        "iat":    time.Now().Unix(),
        "exp":    time.Now().Add(s.cfg.AccessTokenTTL).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(s.cfg.JWTSecret))
}

// Request/Response types
type RegisterRequest struct {
    Email       string          `json:"email" validate:"required,email"`
    Password    *string         `json:"password" validate:"omitempty,min=8"`
    DisplayName string          `json:"display_name" validate:"required,min=2,max=50"`
    PhoneHash   *string         `json:"phone_hash"`
    Faction     *domain.Faction `json:"faction"` // chosen during onboarding, may be nil initially
}
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required"`
}
type GoogleLoginRequest struct {
    IDToken   string          `json:"id_token" validate:"required"`
    Faction   *domain.Faction `json:"faction"`
    PhoneHash *string         `json:"phone_hash"`
}
type AppleLoginRequest struct {
    IdentityToken string          `json:"identity_token" validate:"required"`
    UserIdentifier string         `json:"user_identifier" validate:"required"`
    DisplayName   string          `json:"display_name" validate:"required"`
    Faction       *domain.Faction `json:"faction"`
}
type AuthResponse struct {
    User         domain.User `json:"user"`
    AccessToken  string      `json:"access_token"`
    RefreshToken string      `json:"refresh_token"`
}
```

### 4.2 Run Service

```go
// internal/run/service.go
type Service struct {
    repo       Repository
    territorySvc territory.Service
    cheatDet   *CheatDetector
    statsCalc  *StatsCalculator
    ptsCalc    *PointsCalculator
    wsHub      *websocket.Hub
    cfg        *config.Config
}

// StartRun creates an active run record and validates social participants
func (s *Service) StartRun(ctx context.Context, userID uuid.UUID, req StartRunRequest) (*domain.Run, error) {
    run := domain.Run{
        ID:         uuid.New(),
        UserID:     userID,
        StartTime:  time.Now(),
        Status:     domain.RunStatusActive,
        SocialRun:  req.SocialRun,
    }

    if req.SocialRun {
        // Validate friends are nearby (<20m from start)
        if err := s.validateSocialParticipants(ctx, userID, req); err != nil {
            return nil, err
        }
        run.SocialLeader = &userID
        run.SocialParticipants = req.SocialParticipants
    }

    if err := s.repo.Create(ctx, &run); err != nil {
        return nil, fmt.Errorf("create run: %w", err)
    }
    return &run, nil
}

// EndRun processes the completed run: cheat detection, capture, stats, points
func (s *Service) EndRun(ctx context.Context, userID uuid.UUID, runID uuid.UUID, req EndRunRequest) (*RunSummary, error) {
    run, err := s.repo.GetByID(ctx, runID)
    if err != nil { return nil, err }
    if run.UserID != userID { return nil, ErrNotOwner }
    if run.Status != domain.RunStatusActive { return nil, ErrRunNotActive }

    // 1. Cheat detection
    cheatResult := s.cheatDet.Check(req.GPSPoints, req.DistanceM, req.DurationS)
    
    // 2. Compute stats (always — even flagged runs get personal stats)
    stats := s.statsCalc.Compute(req.GPSPoints)
    
    // 3. Territory capture (if clean)
    var territoryPoints, runnerPoints int
    var hexesCaptured, hexesStolen int
    var captureMode *domain.CaptureMode

    if cheatResult == CheatClean {
        captureResult, err := s.territorySvc.CaptureRun(ctx, run, req.GPSPoints)
        if err != nil { return nil, fmt.Errorf("territory capture: %w", err) }
        territoryPoints = captureResult.TerritoryPoints
        hexesCaptured = captureResult.HexesCaptured
        hexesStolen = captureResult.HexesStolen
        captureMode = &captureResult.Mode
    }

    // 4. Compute points
    runnerPoints = s.ptsCalc.RunnerPoints(stats.DistanceM, stats.ElevationGainM, territoryPoints, run.SocialRun, len(run.SocialParticipants))

    // 5. Update run
    endTime := time.Now()
    run.EndTime = &endTime
    run.Status = cheatResultToRunStatus(cheatResult)
    run.DistanceM = &stats.DistanceM
    run.ElevationGainM = &stats.ElevationGainM
    run.AvgPaceSPerKM = &stats.AvgPace
    run.MaxSpeedKMH = &stats.MaxSpeed
    run.TerritoryPoints = territoryPoints
    run.RunnerPoints = runnerPoints
    run.CaptureMode = captureMode
    run.Polyline = &stats.Polyline

    if err := s.repo.Update(ctx, run); err != nil { return nil, err }

    // 6. Insert GPS points
    if err := s.repo.InsertGPSPoints(ctx, req.GPSPoints); err != nil { return nil, err }

    // 7. Broadcast & notify
    if territoryPoints > 0 {
        s.wsHub.BroadcastTerritoryChange(ctx, s.buildTerritoryEvent(captureResult))
    }
    s.notifyFriends(ctx, run)

    return &RunSummary{
        RunID:          runID,
        Status:         run.Status,
        CaptureMode:    captureMode,
        DistanceM:      *run.DistanceM,
        DurationS:      int(endTime.Sub(run.StartTime).Seconds()),
        AvgPaceSPerKM:  *run.AvgPaceSPerKM,
        ElevationGainM: *run.ElevationGainM,
        TerritoryPoints: territoryPoints,
        RunnerPoints:   runnerPoints,
        HexesCaptured:  hexesCaptured,
        HexesStolen:    hexesStolen,
        CheatStatus:    string(cheatResult),
    }, nil
}

type StartRunRequest struct {
    Lat                float64     `json:"lat" validate:"required"`
    Lng                float64     `json:"lng" validate:"required"`
    SocialRun          bool        `json:"social_run"`
    SocialParticipants []uuid.UUID `json:"social_participants"`
}

type EndRunRequest struct {
    GPSPoints     []domain.GPSPoint `json:"gps_points" validate:"required,min=2"`
    DistanceM     float64           `json:"distance_m"`
    DurationS     int               `json:"duration_s"`
    HealthSource  *string           `json:"health_source"`
}

type RunSummary struct {
    RunID           uuid.UUID       `json:"run_id"`
    Status          domain.RunStatus `json:"status"`
    CaptureMode     *domain.CaptureMode `json:"capture_mode"`
    DistanceM       float64         `json:"distance_m"`
    DurationS       int             `json:"duration_s"`
    AvgPaceSPerKM   float64         `json:"avg_pace_s_per_km"`
    ElevationGainM  float64         `json:"elevation_gain_m"`
    TerritoryPoints int             `json:"territory_points"`
    RunnerPoints    int             `json:"runner_points"`
    HexesCaptured   int             `json:"hexes_captured"`
    HexesStolen     int             `json:"hexes_stolen"`
    CheatStatus     string          `json:"cheat_status"`
}
```

### 4.3 Territory Capture Service

```go
// internal/territory/capture.go
type CaptureResult struct {
    Mode             domain.CaptureMode
    HexesCaptured    int
    HexesStolen      int
    TerritoryPoints  int
    Changes          []domain.TerritoryChange
    AffectedHexes    []domain.Hex
}

func (s *Service) CaptureRun(ctx context.Context, run *domain.Run, points []domain.GPSPoint) (*CaptureResult, error) {
    // Determine capture mode
    captureMode, geometry, err := s.determineCaptureGeometry(ctx, points)
    if err != nil {
        return nil, fmt.Errorf("determine geometry: %w", err)
    }
    if captureMode == nil {
        return nil, ErrLoopNotClosedAndPathDisabled
    }

    // Extract H3 cells from geometry
    cells, err := s.h3util.PolygonToCells(geometry, 11)
    if err != nil {
        return nil, fmt.Errorf("h3 polygon to cells: %w", err)
    }

    // Apply capture in transaction
    result := CaptureResult{Mode: *captureMode}
    
    err = s.repo.WithTx(ctx, func(tx Repository) error {
        for _, cell := range cells {
            hex, err := tx.GetHex(ctx, cell)
            if err != nil && err != ErrNotFound { return err }
            
            change := domain.TerritoryChange{
                H3Index: cell,
                RunID:   &run.ID,
            }
            
            if hex == nil {
                // Neutral → captured
                h := domain.Hex{
                    H3Index: cell,
                    OwnedBy: run.Faction(),
                    HP: 1,
                    CapturedByID: &run.ID,
                    CapturedAt: time.Now(),
                    LastDecayedAt: time.Now(),
                }
                if err := tx.UpsertHex(ctx, &h); err != nil { return err }
                change.NewOwner = run.Faction()
                change.HPAfter = 1
                result.TerritoryPoints += s.cfg.TerritoryBasePoints
                result.HexesCaptured++
            } else if *hex.OwnedBy == *run.Faction() {
                // Own team → reinforce
                newHP := min(hex.HP + 1, s.cfg.HexHPCap)
                change.PreviousOwner = hex.OwnedBy
                change.NewOwner = hex.OwnedBy
                change.HPBefore = hex.HP
                change.HPAfter = newHP
                hex.HP = newHP
                if err := tx.UpsertHex(ctx, hex); err != nil { return err }
                result.TerritoryPoints += s.cfg.TerritoryBasePoints
                result.HexesCaptured++
            } else {
                // Enemy → attack
                newHP := hex.HP - 1
                change.PreviousOwner = hex.OwnedBy
                change.HPBefore = hex.HP
                if newHP <= 0 {
                    // FLIP
                    hex.OwnedBy = run.Faction()
                    hex.HP = 1
                    hex.CapturedByID = &run.ID
                    hex.CapturedAt = time.Now()
                    change.NewOwner = run.Faction()
                    change.HPAfter = 1
                    bonus := int(float64(s.cfg.TerritoryBasePoints) * s.cfg.TerritoryStealBonusPct)
                    result.TerritoryPoints += s.cfg.TerritoryBasePoints + bonus
                    result.HexesStolen++
                } else {
                    hex.HP = newHP
                    change.NewOwner = hex.OwnedBy
                    change.HPAfter = newHP
                    result.TerritoryPoints += s.cfg.TerritoryBasePoints
                    result.HexesCaptured++
                }
                if err := tx.UpsertHex(ctx, hex); err != nil { return err }
            }
            
            if err := tx.RecordChange(ctx, &change); err != nil { return err }
            result.Changes = append(result.Changes, change)
        }
        return nil
    })
    
    return &result, err
}

func (s *Service) determineCaptureGeometry(ctx context.Context, points []domain.GPSPoint) (*domain.CaptureMode, geom.Geometry, error) {
    start := points[0]
    end := points[len(points)-1]
    dist := haversine(start.Lat, start.Lng, end.Lat, end.Lng)
    
    if dist <= s.cfg.LoopClosureRadiusM {
        // Polygon capture
        simplified := douglasPeucker(points, 10.0)
        polygon := buildClosedRing(simplified)
        
        if !polygon.IsValid() {
            // Attempt to fix self-intersection
            fixed := extractLargestSimplePolygon(polygon)
            if fixed == nil {
                // Fall through to path capture if enabled
                if s.pathCaptureEnabled() {
                    mode := domain.CaptureModePath
                    pathGeo := buildPathCorridor(simplified, s.cfg.PathCaptureBufferM)
                    return &mode, pathGeo, nil
                }
                return nil, nil, ErrInvalidPolygon
            }
            polygon = fixed
        }
        
        mode := domain.CaptureModePolygon
        return &mode, polygon, nil
    }
    
    // Open run — check if path capture is enabled
    if s.pathCaptureEnabled() {
        simplified := douglasPeucker(points, 10.0)
        corridor := buildPathCorridor(simplified, s.cfg.PathCaptureBufferM)
        mode := domain.CaptureModePath
        return &mode, corridor, nil
    }
    
    return nil, nil, ErrLoopNotClosedAndPathDisabled
}

func (s *Service) pathCaptureEnabled() bool {
    // Check Redis for runtime override first
    val, err := s.redis.Get(ctx, config.KeyPathCaptureEnabled).Result()
    if err == nil {
        return val == "true"
    }
    return s.cfg.PathCaptureEnabled
}
```

### 4.4 Cheat Detection Engine

```go
// internal/run/cheat.go
type CheatResult string
const (
    CheatClean    CheatResult = "clean"
    CheatFlagged  CheatResult = "flagged"
    CheatRejected CheatResult = "rejected"
)

type CheatDetector struct {
    cfg *config.Config
}

func (d *CheatDetector) Check(points []domain.GPSPoint, totalDistanceM, totalDurationS float64) CheatResult {
    // 1. Speed checks
    for i := 0; i < len(points)-1; i++ {
        segDist := haversine(points[i].Lat, points[i].Lng, points[i+1].Lat, points[i+1].Lng)
        segDur := points[i+1].Timestamp.Sub(points[i].Timestamp).Seconds()
        if segDur <= 0 { continue }
        segSpeedKMH := (segDist / segDur) * 3.6
        
        if segSpeedKMH > d.cfg.MaxSegmentSpeedKMH {
            return CheatRejected
        }
        
        // Teleportation
        if segDist > d.cfg.MaxGPSJumpM {
            return CheatRejected
        }
    }
    
    // Average speed check (for runs > 2km)
    if totalDistanceM > d.cfg.MinDistanceForAvgCheckM {
        avgSpeedKMH := (totalDistanceM / totalDurationS) * 3.6
        if avgSpeedKMH > d.cfg.MaxAvgSpeedKMH {
            return CheatRejected
        }
    }
    
    // 2. Soft flags
    flagged := false
    
    avgSpeedKMH := (totalDistanceM / totalDurationS) * 3.6
    if avgSpeedKMH > d.cfg.FlagAvgSpeedKMH {
        flagged = true
    }
    if totalDistanceM > d.cfg.FlagDistanceM {
        flagged = true
    }
    
    // Low accuracy
    lowAcc := 0
    for _, p := range points {
        if p.HorizontalAccuracy != nil && *p.HorizontalAccuracy > d.cfg.FlagLowAccuracyM {
            lowAcc++
        }
    }
    if float64(lowAcc)/float64(len(points)) > d.cfg.FlagLowAccuracyRatio {
        flagged = true
    }
    
    // Heart rate corroboration
    if hasHeartRate(points) {
        avgHR := avgHeartRate(points)
        avgSpeedMS := avgSpeedKMH / 3.6
        restHR := 60.0
        maxHR := 200.0
        expectedMin := restHR + (maxHR - restHR) * 0.4
        if avgHR < expectedMin {
            flagged = true
        }
    }
    
    if flagged {
        return CheatFlagged
    }
    return CheatClean
}
```

### 4.5 Points Calculator

```go
// internal/points/calculator.go
type Calculator struct {
    cfg *config.Config
}

func (c *Calculator) RunnerPoints(distanceM, elevationGainM float64, territoryPoints int, social bool, friendCount int) int {
    pts := distanceM / 100 * c.cfg.RunnerPointPer100m
    pts += elevationGainM / 10 * c.cfg.RunnerPointPer10mElev
    pts += float64(territoryPoints) * c.cfg.TerritoryPointSharePct
    
    if social && friendCount > 0 {
        extra := c.cfg.SocialBaseMultiplier + (float64(friendCount-1) * c.cfg.SocialExtraPerFriend)
        extra = min(extra, c.cfg.SocialMultiplierCap)
        pts *= 1.0 + extra
    }
    
    return int(math.Round(pts))
}

// XPForLevel returns cumulative XP required to reach a level
func XPForLevel(level int) int64 {
    // Fast early, slower later: xp = 100 * level^1.5
    return int64(100 * math.Pow(float64(level), 1.5))
}

// LevelFromXP returns the account level for a given XP total
func LevelFromXP(totalXP int64) int {
    level := 1
    for totalXP >= XPForLevel(level) {
        level++
    }
    return level - 1
}
```

### 4.6 Bot Service

```go
// internal/bot/service.go
type Service struct {
    botRepo      Repository
    territorySvc territory.Service
    runSvc       run.Service
    cfg          *config.Config
    httpClient   *http.Client // for OSM tile queries
}

// SeedOnboarding generates territory for a new user's region
func (s *Service) SeedOnboarding(ctx context.Context, userLat, userLng float64) error {
    cells := h3.Disk(geoToH3(userLat, userLng, 11), h3DistanceForRadius(s.cfg.BotSeedingRadiusKM))
    
    // Classify and filter cells
    eligible := filterEligibleCells(ctx, cells) // excludes mountains, water, nature
    
    claimed := mustCount(ctx, "SELECT COUNT(*) FROM hexes WHERE h3_index = ANY($1) AND owned_by IS NOT NULL", cells)
    occupancy := float64(claimed) / float64(len(eligible))
    if occupancy >= s.cfg.BotSeedingMaxClaimed {
        return nil // region already seeded enough
    }
    
    target := int(float64(len(eligible)) * s.cfg.BotSeedingMaxClaimed)
    remaining := target - claimed
    
    // Generate islands (connected clusters)
    islands := s.generateIslands(eligible, remaining)
    
    // Assign factions 50/50 and create hex records + bot profiles
    for _, island := range islands {
        faction := randomFaction()
        for _, cell := range island.Cells {
            hex := domain.Hex{
                H3Index:  cell,
                OwnedBy:  &faction,
                HP:       1,
                CapturedAt: time.Now(),
                LastDecayedAt: time.Now(),
            }
            s.territorySvc.CreateHex(ctx, &hex)
        }
        // Create accompanying bot profiles
        s.createBotProfilesForIsland(ctx, island, faction)
    }
    return nil
}

// SpawnActiveBots runs periodically (every 15 min or on-demand after first run)
func (s *Service) SpawnActiveBots(ctx context.Context) error {
    activeRegions := s.getActiveRegions(ctx) // regions with real users active in past week
    
    for _, region := range activeRegions {
        neonPlayers := s.countActivePlayers(ctx, region, domain.FactionNeon)
        umbraPlayers := s.countActivePlayers(ctx, region, domain.FactionUmbra)
        neonBots := max(0, s.cfg.BotsPerFaction - neonPlayers)
        umbraBots := max(0, s.cfg.BotsPerFaction - umbraPlayers)
        
        if neonBots+umbraBots == 0 { continue }
        
        // Check phase-out threshold
        if neonPlayers+umbraPlayers >= s.cfg.BotPhaseOutThreshold {
            s.deactivateAllBots(ctx, region)
            continue
        }
        
        s.spawnBotsForRegion(ctx, region, neonBots, umbraBots)
    }
    return nil
}

// Bot Run Simulator — runs bot routes as if real users
func (s *Service) simulateBotRun(ctx context.Context, bot *domain.Bot) error {
    // 1. Pick start point within active radius, on plausible path, not near real users
    startLat, startLng := s.pickBotStartPoint(ctx, bot)
    
    // 2. Generate closed loop on street/path network
    route := s.generateRoute(ctx, startLat, startLng, bot.AvgDistanceM)
    
    // 3. Generate GPS points with realistic pacing
    points := s.generateGPSPoints(route, bot)
    
    // 4. Create run record
    run := s.runSvc.CreateBotRun(ctx, bot, points)
    
    // 5. Apply territory capture (bots always run opposite faction of nearest real user)
    s.runSvc.CaptureBotRun(ctx, run, points)
    
    return nil
}
```

---

## 5. HTTP Layer

### 5.1 Router Setup

```go
// cmd/api/main.go
func main() {
    cfg := config.MustLoad()
    
    db := mustConnectDB(cfg)
    defer db.Close()
    rdb := mustConnectRedis(cfg)
    defer rdb.Close()
    
    // Repos
    userRepo := user.NewPostgresRepo(db)
    runRepo := run.NewPostgresRepo(db)
    territoryRepo := territory.NewPostgresRepo(db)
    friendRepo := friend.NewPostgresRepo(db)
    seasonRepo := season.NewPostgresRepo(db)
    botRepo := bot.NewPostgresRepo(db)
    
    // Services
    authSvc := auth.NewService(userRepo, sessionRepo, cfg)
    userSvc := user.NewService(userRepo, rdb)
    runSvc := run.NewService(runRepo, territorySvc, cfg)
    territorySvc := territory.NewService(territoryRepo, rdb, cfg)
    friendSvc := friend.NewService(friendRepo, rdb)
    seasonSvc := season.NewService(seasonRepo)
    botSvc := bot.NewService(botRepo, territorySvc, runSvc, cfg)
    
    // WebSocket hub
    wsHub := websocket.NewHub()
    go wsHub.Run()
    
    // Router
    r := chi.NewRouter()
    
    // Global middleware
    r.Use(middleware.Recovery)
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger(zap.NewProduction()))
    r.Use(middleware.Timeout(30 * time.Second))
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Authorization", "Content-Type"},
    }))
    
    // Health
    r.Get("/health", healthHandler(db, rdb))
    
    // Public routes
    r.Route("/auth", func(r chi.Router) {
        r.Post("/register", auth.NewHandler(authSvc).Register)
        r.Post("/login", auth.NewHandler(authSvc).Login)
        r.Post("/google", auth.NewHandler(authSvc).GoogleLogin)
        r.Post("/apple", auth.NewHandler(authSvc).AppleLogin)
        r.Post("/refresh", auth.NewHandler(authSvc).Refresh)
    })
    
    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(middleware.Auth(cfg.JWTSecret))
        r.Use(middleware.RateLimit(rdb))
        
        r.Route("/users", user.NewHandler(userSvc).Routes)
        r.Route("/runs", run.NewHandler(runSvc).Routes)
        r.Route("/territory", territory.NewHandler(territorySvc).Routes)
        r.Route("/friends", friend.NewHandler(friendSvc).Routes)
        r.Route("/leaderboard", leaderboard.NewHandler(leaderboardSvc).Routes)
        r.Route("/seasons", season.NewHandler(seasonSvc).Routes)
    })
    
    // WebSocket (authenticated via query param)
    r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
        token := r.URL.Query().Get("token")
        claims, err := parseJWT(token, cfg.JWTSecret)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        websocket.ServeWs(wsHub, w, r, claims)
    })
    
    // Admin routes (internal, no external exposure)
    r.Route("/admin", func(r chi.Router) {
        r.Use(ipWhitelist(cfg))
        r.Post("/bots/trigger", admin.NewHandler(botSvc).TriggerBots)
        r.Post("/decay/trigger", admin.NewHandler(territorySvc).TriggerDecay)
        r.Post("/season/rollover", admin.NewHandler(seasonSvc).Rollover)
    })
    
    // Start server
    srv := &http.Server{Addr: ":" + cfg.Port, Handler: r}
    // ... graceful shutdown
}
```

### 5.2 Handler Pattern

Every handler follows this pattern:

```go
// internal/run/handler.go
type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc} }

func (h *Handler) EndRun(w http.ResponseWriter, r *http.Request) {
    userID := middleware.UserIDFromContext(r.Context())
    runID, err := uuid.Parse(chi.URLParam(r, "id"))
    if err != nil {
        respondError(w, http.StatusBadRequest, "invalid run id")
        return
    }
    
    var req EndRunRequest
    if err := decodeJSON(r, &req); err != nil {
        respondError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    
    if err := validate.Struct(req); err != nil {
        respondValidationError(w, err)
        return
    }
    
    summary, err := h.svc.EndRun(r.Context(), userID, runID, req)
    if err != nil {
        if errors.Is(err, ErrNotOwner) {
            respondError(w, http.StatusForbidden, "not your run")
            return
        }
        if errors.Is(err, ErrRunNotActive) {
            respondError(w, http.StatusConflict, "run already ended")
            return
        }
        respondError(w, http.StatusInternalServerError, "failed to end run")
        return
    }
    
    respondJSON(w, http.StatusOK, summary)
}
```

### 5.3 Middleware

```go
// internal/middleware/auth.go
func Auth(jwtSecret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                respondError(w, http.StatusUnauthorized, "missing authorization")
                return
            }
            claims, err := parseAndValidateJWT(token, jwtSecret)
            if err != nil {
                respondError(w, http.StatusUnauthorized, "invalid token")
                return
            }
            userID, _ := uuid.Parse(claims["sub"].(string))
            ctx := context.WithValue(r.Context(), CtxKeyUserID, userID)
            ctx = context.WithValue(ctx, CtxKeyUserFaction, claims["faction"])
            ctx = context.WithValue(ctx, CtxKeyUserTier, claims["tier"])
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// internal/middleware/ratelimit.go
func RateLimit(rdb *redis.Client) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := rateLimitKey(r)
            limit, window := rateLimitForRoute(r.URL.Path)
            
            allowed, err := rl.CheckRateLimit(r.Context(), rdb, key, limit, window)
            if err != nil {
                next.ServeHTTP(w, r) // fail open
                return
            }
            if !allowed {
                respondError(w, http.StatusTooManyRequests, "rate limit exceeded")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 6. WebSocket Hub

```go
// internal/websocket/hub.go
type Hub struct {
    clients    map[uuid.UUID]*Client  // userID → client
    territorySubs map[string]map[uuid.UUID]bool // bbox key → userIDs
    friendSubs map[uuid.UUID]map[uuid.UUID]bool // userID → friendIDs watching
    register   chan *Client
    unregister chan *Client
    broadcast  chan *Message
}

type Client struct {
    hub    *Hub
    conn   *gorilla.Conn
    userID uuid.UUID
    send   chan []byte
}

type Message struct {
    Type    string   `json:"type"`
    Payload any      `json:"payload,omitempty"`
    BBox    *[4]float64 `json:"bbox,omitempty"` // [swLat, swLng, neLat, neLng]
}

func NewHub() *Hub {
    return &Hub{
        clients:       make(map[uuid.UUID]*Client),
        territorySubs: make(map[string]map[uuid.UUID]bool),
        friendSubs:    make(map[uuid.UUID]map[uuid.UUID]bool),
        register:      make(chan *Client),
        unregister:    make(chan *Client),
        broadcast:     make(chan *Message),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client.userID] = client
        case client := <-h.unregister:
            if _, ok := h.clients[client.userID]; ok {
                delete(h.clients, client.userID)
                h.unsubscribeAll(client.userID)
                close(client.send)
            }
        case msg := <-h.broadcast:
            h.handleBroadcast(msg)
        }
    }
}

func (h *Hub) handleBroadcast(msg *Message) {
    switch msg.Type {
    case "territory:changed":
        // Find all users whose subscribed bbox overlaps changed area
        for bboxKey, users := range h.territorySubs {
            if bboxOverlaps(bboxKey, msg.Payload) {
                for userID := range users {
                    if client, ok := h.clients[userID]; ok {
                        select {
                        case client.send <- marshal(msg):
                        default:
                            // client buffer full, skip
                        }
                    }
                }
            }
        }
    case "friend:run_started", "friend:run_completed":
        for userID := range h.friendSubs[msg.Payload.UserID] {
            if client, ok := h.clients[userID]; ok {
                client.send <- marshal(msg)
            }
        }
    }
}

// BroadcastTerritoryChange is called by run service after capture
func (h *Hub) BroadcastTerritoryChange(changedHexes []domain.Hex) {
    h.broadcast <- &Message{
        Type:    "territory:changed",
        Payload: changedHexes,
    }
}

// Client message handling
func (c *Client) readPump() {
    defer c.conn.Close()
    for {
        var msg Message
        if err := c.conn.ReadJSON(&msg); err != nil { break }
        
        switch msg.Type {
        case "subscribe:territory":
            key := bboxKey(msg.BBox)
            if c.hub.territorySubs[key] == nil {
                c.hub.territorySubs[key] = make(map[uuid.UUID]bool)
            }
            c.hub.territorySubs[key][c.userID] = true
        case "unsubscribe:territory":
            key := bboxKey(msg.BBox)
            delete(c.hub.territorySubs[key], c.userID)
        case "subscribe:friends":
            // Audit actual friend list from DB on subscribe
            friends := loadFriendIDs(c.userID)
            for _, fid := range friends {
                if c.hub.friendSubs[fid] == nil {
                    c.hub.friendSubs[fid] = make(map[uuid.UUID]bool)
                }
                c.hub.friendSubs[fid][c.userID] = true
            }
        case "run:position":
            // Relayed only to social participants on the same run
            c.relayPosition(msg)
        }
    }
}
```

---

## 7. Background Jobs (Schedulers)

```go
// internal/scheduler/scheduler.go
type Scheduler struct {
    cfg           *config.Config
    territorySvc  territory.Service
    botSvc        bot.Service
    seasonSvc     season.Service
    leaderboardSvc leaderboard.Service
    db            *sqlx.DB
}

func (s *Scheduler) Start(ctx context.Context) {
    c := cron.New()
    
    // HP Decay — daily at midnight UTC
    c.AddFunc("0 0 * * *", func() {
        s.territorySvc.DecayAllHexes(ctx)
    })
    
    // Leaderboard refresh — every 15 min
    c.AddFunc("*/15 * * * *", func() {
        s.leaderboardSvc.RefreshMaterializedView(ctx)
    })
    
    // Bot spawner — every 15 min
    c.AddFunc("*/15 * * * *", func() {
        s.botSvc.SpawnActiveBots(ctx)
    })
    
    // Backup — daily at 3 AM UTC
    c.AddFunc("0 3 * * *", func() {
        s.runBackup(ctx)
    })
    
    // Data cleanup — daily at 2 AM UTC
    c.AddFunc("0 2 * * *", func() {
        s.cleanupDeletedUsers(ctx)
    })
    
    c.Start()
    <-ctx.Done()
    c.Stop()
}
```

---

## 8. Database Migrations

### 8.1 Migration Tool: golang-migrate

```go
// cmd/migrate/main.go
func main() {
    m, _ := migrate.New("file://migrations", os.Getenv("DATABASE_URL"))
    m.Up()
}
```

### 8.2 Migration Files

```sql
-- migrations/001_users.up.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT UNIQUE NOT NULL,
    password_hash   TEXT,
    display_name    TEXT NOT NULL,
    phone_hash      TEXT,
    avatar_url      TEXT,
    faction         TEXT CHECK (faction IN ('neon', 'umbra')),
    account_level   INTEGER NOT NULL DEFAULT 1,
    account_xp      BIGINT NOT NULL DEFAULT 0,
    google_id       TEXT UNIQUE,
    apple_id        TEXT UNIQUE,
    runner_tier     TEXT NOT NULL DEFAULT 'free' CHECK (runner_tier IN ('free', 'runner', 'captain')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_phone_hash ON users(phone_hash) WHERE deleted_at IS NULL AND phone_hash IS NOT NULL;
```

```sql
-- migrations/002_sessions.up.sql
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

```sql
-- migrations/003_runs.up.sql
CREATE TABLE runs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id),
    start_time          TIMESTAMPTZ NOT NULL,
    end_time            TIMESTAMPTZ,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','completed','flagged','rejected')),
    distance_m          DOUBLE PRECISION,
    elevation_gain_m    DOUBLE PRECISION,
    avg_pace_s_per_km   DOUBLE PRECISION,
    max_speed_kmh       DOUBLE PRECISION,
    avg_hr_bpm          INTEGER,
    max_hr_bpm          INTEGER,
    calories            INTEGER,
    closed_loop         BOOLEAN,
    capture_mode        TEXT CHECK (capture_mode IN ('polygon', 'path')),
    loop_snap_distance_m DOUBLE PRECISION,
    path_buffer_radius_m DOUBLE PRECISION,
    polyline            TEXT,
    territory_points    INTEGER NOT NULL DEFAULT 0,
    runner_points       INTEGER NOT NULL DEFAULT 0,
    social_run          BOOLEAN NOT NULL DEFAULT FALSE,
    social_leader       UUID REFERENCES users(id),
    social_participants UUID[],
    health_source       TEXT CHECK (health_source IN ('healthkit', 'healthconnect')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_runs_user ON runs(user_id, created_at DESC);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_start_time ON runs(start_time);

-- GPS Points hypertable
CREATE TABLE gps_points (
    run_id              UUID NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL,
    lat                 DOUBLE PRECISION NOT NULL,
    lng                 DOUBLE PRECISION NOT NULL,
    altitude            DOUBLE PRECISION,
    speed               DOUBLE PRECISION,
    horizontal_accuracy DOUBLE PRECISION,
    heart_rate          INTEGER
);
SELECT create_hypertable('gps_points', 'timestamp');
CREATE INDEX idx_gps_points_run ON gps_points(run_id, timestamp);
```

```sql
-- migrations/004_territory.up.sql
CREATE TABLE hexes (
    h3_index        BIGINT PRIMARY KEY,
    owned_by        TEXT CHECK (owned_by IN ('neon', 'umbra')),
    hp              SMALLINT NOT NULL DEFAULT 1 CHECK (hp >= 1 AND hp <= 10),
    captured_by     UUID REFERENCES runs(id),
    captured_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_decayed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_natural      BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_hexes_owned ON hexes(owned_by) WHERE owned_by IS NOT NULL;

CREATE TABLE territory_changes (
    id              BIGSERIAL PRIMARY KEY,
    h3_index        BIGINT NOT NULL,
    run_id          UUID REFERENCES runs(id),
    previous_owner  TEXT CHECK (previous_owner IN ('neon', 'umbra')),
    new_owner       TEXT CHECK (new_owner IN ('neon', 'umbra')),
    hp_before       SMALLINT NOT NULL DEFAULT 0,
    hp_after        SMALLINT NOT NULL DEFAULT 0,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_territory_changes_time ON territory_changes(changed_at);
```

```sql
-- migrations/005_friends.up.sql
CREATE TABLE friends (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'blocked')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, friend_id),
    CHECK (user_id != friend_id)
);
CREATE INDEX idx_friends_friend ON friends(friend_id, status);
```

```sql
-- migrations/006_seasons.up.sql
CREATE TABLE seasons (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    start_date  TIMESTAMPTZ NOT NULL,
    end_date    TIMESTAMPTZ NOT NULL,
    tiers_config JSONB NOT NULL DEFAULT '{"rookie":0,"runner":1000,"sprinter":5000,"elite":15000,"legend":30000}',
    is_active   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE season_participants (
    user_id          UUID NOT NULL REFERENCES users(id),
    season_id        UUID NOT NULL REFERENCES seasons(id),
    runner_points    INTEGER NOT NULL DEFAULT 0,
    territory_points INTEGER NOT NULL DEFAULT 0,
    current_tier     TEXT NOT NULL DEFAULT 'rookie',
    final_tier       TEXT,
    badge_awarded    BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, season_id)
);
```

```sql
-- migrations/007_bots.up.sql
CREATE TABLE bots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name    TEXT NOT NULL,
    faction         TEXT NOT NULL CHECK (faction IN ('neon', 'umbra')),
    avatar_seed     TEXT,
    home_lat        DOUBLE PRECISION NOT NULL,
    home_lng        DOUBLE PRECISION NOT NULL,
    active_radius_km DOUBLE PRECISION NOT NULL DEFAULT 5,
    avg_distance_m  DOUBLE PRECISION NOT NULL DEFAULT 5000,
    total_runs      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deactivated_at  TIMESTAMPTZ
);
CREATE INDEX idx_bots_home ON bots(home_lat, home_lng);
CREATE INDEX idx_bots_faction ON bots(faction, deactivated_at);
```

---

## 9. nginx Configuration

```nginx
# nginx.conf
upstream api {
    server api:8080;
}

server {
    listen 80;
    server_name terrarun.app;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name terrarun.app;

    ssl_certificate     /etc/letsencrypt/live/terrarun.app/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/terrarun.app/privkey.pem;

    # API
    location /api/ {
        proxy_pass http://api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }

    # WebSocket
    location /ws {
        proxy_pass http://api/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 3600s;
    }

    # Mapbox tile proxy
    location /mapbox/ {
        proxy_pass https://api.mapbox.com/;
        proxy_set_header Host api.mapbox.com;
        # Token appended by njs or lua; alternatively Go proxy handles this
    }

    # Static assets (avatars via rustfs)
    location /assets/ {
        proxy_pass http://rustfs:9000/terrarun/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

---

## 10. Makefile

```makefile
# Makefile
.PHONY: dev test lint migrate build deploy

# ---- Development ----
dev:
	docker compose up -d
	cd backend && air                        # live reload
	cd client && flutter run

dev-down:
	docker compose down

# ---- Database ----
migrate-up:
	docker compose exec api ./migrate up

migrate-down:
	docker compose exec api ./migrate down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir backend/migrations -seq $$name

# ---- Testing ----
test:
	cd backend && go test ./internal/... -count=1 -coverprofile=coverage.out
	cd client && flutter test

test-integration:
	cd backend && go test ./internal/... -tags=integration -count=1

test-e2e:
	cd backend && go test ./test/e2e/... -count=1

# ---- Linting ----
lint:
	cd backend && golangci-lint run ./...
	cd backend && go vet ./...
	cd client && flutter analyze

# ---- Build ----
build:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api
	cd backend && CGO_ENABLED=0 go build -o bin/migrate ./cmd/migrate

build-client-ios:
	cd client && flutter build ios --release

build-client-android:
	cd client && flutter build appbundle --release

# ---- Deploy ----
deploy:
	@test -n "$(HOST)" || (echo "HOST required: make deploy HOST=user@server"; exit 1)
	rsync -avz docker-compose.prod.yml nginx.conf backend/bin/ $(HOST):/opt/terrarun/
	ssh $(HOST) "cd /opt/terrarun && docker compose -f docker-compose.prod.yml up -d --build && docker compose exec api ./migrate up"

# ---- Utilities ----
seed-bots:
	docker compose exec api curl -X POST http://localhost:8080/admin/bots/trigger

trigger-decay:
	docker compose exec api curl -X POST http://localhost:8080/admin/decay/trigger

db-psql:
	docker compose exec postgres psql -U terrarun

redis-cli:
	docker compose exec redis redis-cli

logs:
	docker compose logs -f api
```

---

## 11. Error Codes & Response Format

### 11.1 API Response Envelope

```json
// Success
{ "data": { ... } }

// Error
{
  "error": {
    "code": "ERR_VALIDATION_FAILED",
    "message": "Human-readable description",
    "details": [
      { "field": "email", "reason": "invalid email format" }
    ]
  }
}
```

### 11.2 Error Code Table

| Code | HTTP | Description |
|---|---|---|
| `ERR_EMAIL_TAKEN` | 409 | Email already registered |
| `ERR_INVALID_CREDENTIALS` | 401 | Wrong email or password |
| `ERR_OAUTH_ONLY` | 409 | Account uses OAuth, no password |
| `ERR_INVALID_TOKEN` | 401 | JWT or refresh token invalid/expired |
| `ERR_TOKEN_REUSED` | 401 | Refresh token reuse detected (force re-login) |
| `ERR_RATE_LIMITED` | 429 | Too many requests |
| `ERR_VALIDATION_FAILED` | 400 | Request body validation failed |
| `ERR_NOT_FOUND` | 404 | Resource not found |
| `ERR_NOT_OWNER` | 403 | Trying to modify another user's resource |
| `ERR_RUN_NOT_ACTIVE` | 409 | Run already completed/flagged |
| `ERR_LOOP_NOT_CLOSED` | 422 | Run loop not closed, path capture disabled |
| `ERR_INVALID_POLYGON` | 422 | GPS track forms invalid polygon |
| `ERR_FRIEND_ALREADY_EXISTS` | 409 | Friend relationship already exists |
| `ERR_FACTION_ALREADY_CHOSEN` | 409 | Faction already set (permanent) |

---

## 12. Flutter Architecture

### 12.1 Route Definitions

```dart
// lib/app.dart
class TerraRunApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'TerraRun',
      theme: AppTheme.light,
      initialRoute: '/splash',
      onGenerateRoute: AppRouter.onGenerateRoute,
    );
  }
}

// lib/config/router.dart
class AppRouter {
  static Route<dynamic> onGenerateRoute(RouteSettings settings) {
    switch (settings.name) {
      case '/splash':
        return MaterialPageRoute(builder: (_) => const SplashScreen());
      case '/auth/login':
        return MaterialPageRoute(builder: (_) => const LoginScreen());
      case '/auth/register':
        return MaterialPageRoute(builder: (_) => RegisterScreen());
      case '/auth/faction':
        return MaterialPageRoute(builder: (_) => FactionChoiceScreen());
      case '/onboarding':
        return MaterialPageRoute(builder: (_) => const OnboardingScreen());
      case '/home':
        return MaterialPageRoute(builder: (_) => const MapScreen());
      case '/run/active':
        return MaterialPageRoute(builder: (_) => const RunActiveScreen());
      case '/run/summary':
        final args = settings.arguments as RunSummaryArgs;
        return MaterialPageRoute(builder: (_) => RunSummaryScreen(runID: args.runID));
      case '/run/history':
        return MaterialPageRoute(builder: (_) => const RunHistoryScreen());
      case '/run/detail':
        final runID = settings.arguments as String;
        return MaterialPageRoute(builder: (_) => RunDetailScreen(runID: runID));
      case '/friends':
        return MaterialPageRoute(builder: (_) => const FriendsListScreen());
      case '/friends/feed':
        return MaterialPageRoute(builder: (_) => const FriendFeedScreen());
      case '/friends/profile':
        final userID = settings.arguments as String;
        return MaterialPageRoute(builder: (_) => FriendProfileScreen(userID: userID));
      case '/leaderboard':
        return MaterialPageRoute(builder: (_) => const LeaderboardScreen());
      case '/profile':
        return MaterialPageRoute(builder: (_) => const ProfileScreen());
      case '/profile/edit':
        return MaterialPageRoute(builder: (_) => const EditProfileScreen());
      case '/stats':
        return MaterialPageRoute(builder: (_) => const StatsScreen());
      default:
        return MaterialPageRoute(builder: (_) => const NotFoundScreen());
    }
  }
}
```

### 12.2 State Management — flutter_bloc (map screen example)

```dart
// lib/features/map/bloc/map_bloc.dart
class MapBloc extends Bloc<MapEvent, MapState> {
  final TerritoryRepository territoryRepo;
  final RunService runService;
  late StreamSubscription _wsSubscription;

  MapBloc({required this.territoryRepo, required this.runService})
      : super(MapState.initial()) {
    on<MapLoaded>(_onMapLoaded);
    on<MapMoved>(_onMapMoved);
    on<RunStarted>(_onRunStarted);
    on<RunEnded>(_onRunEnded);
    on<TerritoryUpdated>(_onTerritoryUpdated);
    on<FriendActivityReceived>(_onFriendActivity);
  }

  Future<void> _onMapLoaded(MapLoaded event, Emitter<MapState> emit) async {
    emit(state.copyWith(status: MapStatus.loading));
    try {
      final hexes = await territoryRepo.getHexes(
        state.currentBBox ?? event.initialBBox,
      );
      emit(state.copyWith(
        status: MapStatus.loaded,
        hexes: hexes,
      ));
    } catch (e) {
      emit(state.copyWith(status: MapStatus.error, error: e.toString()));
    }
  }

  Future<void> _onMapMoved(MapMoved event, Emitter<MapState> emit) async {
    emit(state.copyWith(currentBBox: event.bbox));
    // Debounce: fetch new hexes for new viewport
    final hexes = await territoryRepo.getHexes(event.bbox);
    emit(state.copyWith(hexes: hexes));
    // Resubscribe WS to new bbox
    _wsSubscription.cancel();
    _wsSubscription = runService.subscribeTerritory(event.bbox, _onTerritoryUpdate);
  }

  void _onTerritoryUpdated(TerritoryUpdated event, Emitter<MapState> emit) {
    final updatedHexes = Map<int64, Hex>.from(state.hexes);
    for (final hex in event.hexes) {
      updatedHexes[hex.h3Index] = hex;
    }
    emit(state.copyWith(hexes: updatedHexes));
  }
}

// MapState
class MapState extends Equatable {
  final MapStatus status;
  final Map<int64, Hex> hexes;
  final BBox? currentBBox;
  final RunState? activeRun;
  final String? error;

  const MapState({...});
  factory MapState.initial() => MapState(status: MapStatus.initial, hexes: {});
  MapState copyWith({...}) => MapState(...);
}
```

### 12.3 Map Screen Widget

```dart
// lib/features/map/screens/map_screen.dart
class MapScreen extends StatefulWidget { ... }

class _MapScreenState extends State<MapScreen> {
  late MapboxMapController mapController;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Stack(
        children: [
          // Map
          MapboxMap(
            accessToken: '', // proxy URL used
            onMapCreated: (controller) {
              mapController = controller;
              context.read<MapBloc>().add(MapLoaded(
                initialBBox: BBox.fromMapBounds(await controller.getVisibleRegion()),
              ));
            },
            onCameraIdle: () async {
              final bounds = await mapController.getVisibleRegion();
              context.read<MapBloc>().add(MapMoved(bbox: BBox.fromMapBounds(bounds)));
            },
          ),
          // Hex Overlay (CustomPainter or tile layer)
          const HexOverlay(),
          // Run Controls
          Positioned(
            bottom: 32,
            left: 0,
            right: 0,
            child: RunControls(
              onStart: () => _startRun(),
            ),
          ),
          // FABs
          Positioned(
            top: 48, right: 16,
            child: Column(
              children: [
                IconButton(icon: Icon(Icons.leaderboard), onPressed: () => Navigator.pushNamed(context, '/leaderboard')),
                IconButton(icon: Icon(Icons.people), onPressed: () => Navigator.pushNamed(context, '/friends')),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
```

### 12.4 Hex Overlay Widget

```dart
// lib/features/map/widgets/hex_overlay.dart
class HexOverlay extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return BlocBuilder<MapBloc, MapState>(
      buildWhen: (prev, curr) => prev.hexes != curr.hexes,
      builder: (context, state) {
        return IgnorePointer(
          child: CustomPaint(
            painter: HexPainter(
              hexes: state.hexes.values.toList(),
              cameraPosition: mapController.cameraPosition,
            ),
            size: Size.infinite,
          ),
        );
      },
    );
  }
}

class HexPainter extends CustomPainter {
  final List<Hex> hexes;

  @override
  void paint(Canvas canvas, Size size) {
    for (final hex in hexes) {
      final screenPoint = latLngToScreen(hex.centerLat, hex.centerLng);
      final color = hex.ownedBy == 'neon' 
          ? AppTheme.neonColor.withOpacity(hex.hp / 10 * 0.7)
          : AppTheme.umbraColor.withOpacity(hex.hp / 10 * 0.7);
      final paint = Paint()
        ..color = color
        ..style = PaintingStyle.fill;
      
      // Draw hex polygon projected to screen
      final path = hexToScreenPath(hex.h3Index);
      canvas.drawPath(path, paint);
      canvas.drawPath(path, Paint()..color = Colors.white24..style = PaintingStyle.stroke..strokeWidth = 0.5);
    }
  }
}
```

### 12.5 GPS Service (Background)

```dart
// lib/features/run/services/gps_service.dart
class GPSService {
  final List<GPSPoint> _points = [];
  StreamSubscription<Position>? _subscription;
  DateTime? _startTime;

  Stream<GPSPoint> startTracking() {
    _startTime = DateTime.now();
    _points.clear();

    return Geolocator.getPositionStream(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 5, // meters
      ),
    ).map((position) {
      final point = GPSPoint(
        timestamp: position.timestamp ?? DateTime.now(),
        lat: position.latitude,
        lng: position.longitude,
        altitude: position.altitude,
        speed: position.speed,
        horizontalAccuracy: position.accuracy,
      );
      _points.add(point);
      return point;
    });
  }

  void stopTracking() {
    _subscription?.cancel();
  }

  List<GPSPoint> get points => List.unmodifiable(_points);
  double get totalDistance => computeTotalDistance(_points);
  int get durationSeconds => DateTime.now().difference(_startTime!).inSeconds;
  double get avgPace => durationSeconds / (totalDistance / 1000);
}
```

### 12.6 pubspec.yaml Dependencies

```yaml
dependencies:
  flutter:
    sdk: flutter
  flutter_bloc: ^8.1.0
  equatable: ^2.0.5
  dio: ^5.4.0
  flutter_secure_storage: ^9.0.0
  go_router: ^13.0.0
  mapbox_gl: ^0.16.0
  geolocator: ^11.0.0
  flutter_background_geolocation: ^4.15.0
  google_sign_in: ^6.2.0
  sign_in_with_apple: ^5.0.0
  contacts_service: ^0.6.3
  crypto: ^3.0.3       # SHA-256 for phone hashing
  health: ^12.0.0      # HealthKit
  health_connect: ^1.0.0  # Google Health Connect
  fl_chart: ^0.68.0    # Pace/elevation charts
  intl: ^0.19.0
  json_annotation: ^4.8.0
  path_provider: ^2.1.0
  flutter_local_notifications: ^17.0.0
  firebase_core: ^2.27.0
  firebase_messaging: ^14.7.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  bloc_test: ^9.1.0
  mocktail: ^1.0.0
  build_runner: ^2.4.0
  json_serializable: ^6.7.0
  flutter_lints: ^3.0.0
```

---

## 13. Test Structure

### 13.1 Go Tests

```
backend/internal/
├── auth/
│   ├── service_test.go         # 30+ test cases
│   └── handler_test.go         # 15+ HTTP test cases
├── run/
│   ├── service_test.go
│   ├── cheat_test.go           # Exhaustive: 50+ GPS scenarios
│   ├── stats_test.go
│   └── handler_test.go
├── territory/
│   ├── capture_test.go         # Polygon + path capture scenarios
│   ├── hp_test.go              # HP state machine transitions
│   └── handler_test.go
├── friend/
│   ├── service_test.go
│   └── handler_test.go
├── bot/
│   ├── seeder_test.go
│   └── spawner_test.go
├── points/
│   └── calculator_test.go
└── websocket/
    └── hub_test.go
```

### 13.2 Flutter Tests

```
client/test/
├── features/
│   ├── auth/
│   │   └── bloc/auth_bloc_test.dart
│   ├── map/
│   │   ├── bloc/map_bloc_test.dart
│   │   └── widgets/hex_overlay_test.dart
│   ├── run/
│   │   ├── bloc/run_bloc_test.dart
│   │   └── services/gps_service_test.dart
│   └── social/
│       └── bloc/friends_bloc_test.dart
├── integration/
│   └── app_test.dart           # Full flow: register → run → capture
└── helpers/
    ├── test_utils.dart
    └── mocks.dart
```

### 13.3 Key Test Cases

**Cheat Detection:**
- Clean run: 5km @ 12 km/h, good accuracy → clean
- Motorcycle: consecutive segments at 45 km/h → rejected
- Car: 10km run avg 30 km/h → rejected
- Teleportation: 1km GPS jump → rejected
- Bike: 20km run avg 18 km/h flat → flagged
- Ultrarunner: 60km run → flagged (manual review)
- Bad GPS: 50% points with >20m accuracy → flagged

**HP State Machine:**
- Neutral(0) → capture → HP=1, owned=N
- Own(N, HP=1) → same-team capture → HP=2
- Own(N, HP=5) → same-team capture → HP=6
- Own(N, HP=10) → same-team capture → HP=10 (cap)
- Own(N, HP=3) → enemy capture → HP=2
- Own(N, HP=1) → enemy capture → HP=0 → FLIP → HP=1, owned=U
- Own(N, HP=any) → daily decay → HP-1, floor at 1
- Own(N, HP=1) → daily decay → HP=1 (floor)
- Never returns to neutral (HP never goes to 0 without a flip)

**Territory Capture:**
- Closed loop 5km → polygon → captures interior hexes
- Open run 3km, path enabled → corridor buffer 25m → captures stripe hexes
- Open run 3km, path disabled → no capture, stats only
- Self-intersecting loop → extracts largest simple sub-polygon
- Captured hex count verified against known values for test polygons

---

## 14. Implementation Order (Task Groups)

### Phase 1 — Foundation (Week 1-2)
1. Docker Compose setup (PostgreSQL+TimescaleDB, Redis, rustfs, Go API)
2. Go project scaffolding, config, middleware
3. Database migrations (all tables)
4. Auth module (register, login, refresh + Google/Apple OAuth)
5. User profile CRUD
6. Flutter: auth screens, secure storage, API client

### Phase 2 — Core Gameplay (Week 3-5)
7. Run start/end endpoints
8. GPS point storage via TimescaleDB
9. Cheat detection engine
10. Territory capture — polygon mode (closed loop)
11. Territory capture — path mode (config toggle)
12. HP system + daily decay cron
13. Points calculator + XP/level system
14. Flutter: map screen, hex overlay, run controls, GPS tracking
15. Flutter: run summary screen, stats charts

### Phase 3 — Social & Engagement (Week 6-7)
16. Friend system (add, accept, feed)
17. Social run multiplier
18. Contact matching service
19. WebSocket hub (territory, friend activity)
20. Push notifications (FCM + APNs)
21. Leaderboards (Redis sorted sets)
22. Flutter: friends screens, feed, leaderboard

### Phase 4 — Bots & Territory Seeding (Week 8-9)
23. Bot profile generation (names, stats)
24. Onboarding seeder (100km radius, islands, faction split)
25. Active bot spawner (15-min cron, faction balancing)
26. Bot routing engine (plausible path generation)
27. Bot scaling/phase-out logic
28. Flutter: onboarding flow with bot seeding UX (loading + completion)

### Phase 5 — Polish & Launch (Week 10-12)
29. Season system (creation, participation, rollover)
30. Run history with pagination/filtering
31. HealthKit / Health Connect integration (Runner tier)
32. Mapbox token proxy
33. nginx config, TLS, deploy scripts
34. Performance optimization (bulk hex ops, H3 caching, DB indexing)
35. Load testing + profiling
36. Beta launch configs (all monetization gates disabled)

---

## 15. Key Implementation Decisions & Edge Cases

### 15.1 H3 Index Storage
- Store as `BIGINT` (int64) for index performance and PG compatibility
- Convert to/from hex string representation in Go service layer
- Use H3 v4 Go library (`github.com/uber/h3-go/v4`)

### 15.2 GPS Point Storage Optimization
- TimescaleDB hypertable with 1-day chunks for efficient time-range queries
- Batch insert via `pgx.CopyFrom` for bulk GPS point ingestion
- Simplify GPS arrays before storage: store only every Nth point for display (N based on accuracy)

### 15.3 Territory Query Optimization
- Hex query by bounding box uses `h3_cell_to_lat`/`h3_cell_to_lng` reference functions for spatial filtering before PostGIS
- Result set capped at 10,000 hexes per request
- Redis cache hex sets per zoom level for common viewports

### 15.4 Bot Route Generation
- Use OpenStreetMap Overpass API (self-hostable instance) for street network data
- Pre-cache street network tiles per region
- Bot routes are generated offline, stored, then "played back" as GPS point streams at scheduled times

### 15.5 Social Run Validation
- When starting a social run, server validates all participants are:
  1. Friends (accepted)
  2. Currently within 20m of the run leader (GPS proximity)
- Reject if any participant fails these checks

### 15.6 Incremental Territory Queries (Map)
- Client sends `?since=timestamp` to get only hex changes since last poll
- WebSocket provides real-time updates for active viewport
- Fallback: full viewport query on map idle if WS disconnected

### 15.7 Faction Permanence
- Faction column nullable initially (NULL = not yet chosen)
- Once set, server rejects any attempt to change it
- Admin override available for edge cases (not user-facing)

### 15.8 Account Deletion (GDPR)
- Soft delete: set `deleted_at`, retain data for 30 days
- After 30 days: hard-delete user row, anonymize territory changes (set `user_id` to NULL in relational reference), delete GPS points
- Export: GET /users/me/export returns all user data as JSON + GPX

### 15.9 Background GPS on Flutter
- `flutter_background_geolocation` plugin keeps tracking active when app is backgrounded
- Config: `stopOnTerminate: false`, `startOnBoot: false`, `desiredAccuracy: high`
- Battery optimization: decrease GPS polling when user is stationary (no movement > 2 min)
- Warn user if battery saver mode is active before starting a run
