package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port            string        `envconfig:"PORT" default:"8080"`
	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"10s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"15s"`

	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	MaxDBConns  int    `envconfig:"MAX_DB_CONNS" default:"25"`
	MinDBConns  int    `envconfig:"MIN_DB_CONNS" default:"5"`

	RedisURL        string `envconfig:"REDIS_URL" required:"true"`
	RedisMaxRetries int    `envconfig:"REDIS_MAX_RETRIES" default:"3"`

	StorageEndpoint  string `envconfig:"STORAGE_ENDPOINT" required:"true"`
	StorageAccessKey string `envconfig:"STORAGE_ACCESS_KEY" required:"true"`
	StorageSecretKey string `envconfig:"STORAGE_SECRET_KEY" required:"true"`
	StorageBucket    string `envconfig:"STORAGE_BUCKET" default:"terrarun"`
	StorageUseSSL    bool   `envconfig:"STORAGE_USE_SSL" default:"false"`

	JWTSecret         string        `envconfig:"JWT_SECRET" required:"true"`
	AccessTokenTTL    time.Duration `envconfig:"ACCESS_TOKEN_TTL" default:"15m"`
	RefreshTokenTTL   time.Duration `envconfig:"REFRESH_TOKEN_TTL" default:"720h"`
	BcryptCost        int           `envconfig:"BCRYPT_COST" default:"12"`
	PhoneHashPepper   string        `envconfig:"PHONE_HASH_PEPPER" required:"true"`

	GoogleClientID   string `envconfig:"GOOGLE_CLIENT_ID" required:"true"`
	AppleTeamID      string `envconfig:"APPLE_TEAM_ID" required:"true"`
	AppleKeyID       string `envconfig:"APPLE_KEY_ID" required:"true"`
	ApplePrivateKey  string `envconfig:"APPLE_PRIVATE_KEY" required:"true"`

	MapboxToken string `envconfig:"MAPBOX_TOKEN"`

	FirebaseCredsFile string `envconfig:"FIREBASE_CREDS_FILE"`

	LoopClosureRadiusM     float64 `envconfig:"LOOP_CLOSURE_RADIUS_M" default:"100"`
	PathCaptureEnabled     bool    `envconfig:"PATH_CAPTURE_ENABLED" default:"true"`
	PathCaptureBufferM     float64 `envconfig:"PATH_CAPTURE_BUFFER_M" default:"25"`
	HexHPCap               int     `envconfig:"HEX_HP_CAP" default:"10"`
	HexHPDecay             int     `envconfig:"HEX_HP_DECAY" default:"1"`
	TerritoryBasePoints    int     `envconfig:"TERRITORY_BASE_POINTS" default:"10"`
	TerritoryStealBonusPct float64 `envconfig:"TERRITORY_STEAL_BONUS_PCT" default:"0.5"`

	RunnerPointPer100m     float64 `envconfig:"RUNNER_POINT_PER_100M" default:"1.0"`
	RunnerPointPer10mElev  float64 `envconfig:"RUNNER_POINT_PER_10M_ELEV" default:"5.0"`
	TerritoryPointSharePct float64 `envconfig:"TERRITORY_POINT_SHARE_PCT" default:"0.1"`
	SocialBaseMultiplier    float64 `envconfig:"SOCIAL_BASE_MULTIPLIER" default:"0.25"`
	SocialExtraPerFriend    float64 `envconfig:"SOCIAL_EXTRA_PER_FRIEND" default:"0.10"`
	SocialMultiplierCap     float64 `envconfig:"SOCIAL_MULTIPLIER_CAP" default:"0.50"`
	XPConversionRatio       float64 `envconfig:"XP_CONVERSION_RATIO" default:"1.0"`

	MaxSegmentSpeedKMH      float64 `envconfig:"MAX_SEGMENT_SPEED_KMH" default:"35"`
	MaxAvgSpeedKMH          float64 `envconfig:"MAX_AVG_SPEED_KMH" default:"20"`
	MinDistanceForAvgCheckM float64 `envconfig:"MIN_DISTANCE_FOR_AVG_CHECK_M" default:"2000"`
	MaxGPSJumpM             float64 `envconfig:"MAX_GPS_JUMP_M" default:"500"`
	FlagAvgSpeedKMH         float64 `envconfig:"FLAG_AVG_SPEED_KMH" default:"15"`
	FlagDistanceM           float64 `envconfig:"FLAG_DISTANCE_M" default:"50000"`
	FlagLowAccuracyM        float64 `envconfig:"FLAG_LOW_ACCURACY_M" default:"20"`
	FlagLowAccuracyRatio    float64 `envconfig:"FLAG_LOW_ACCURACY_RATIO" default:"0.3"`

	BotsPerFaction        int     `envconfig:"BOTS_PER_FACTION" default:"10"`
	BotSeedingRadiusKM    float64 `envconfig:"BOT_SEEDING_RADIUS_KM" default:"100"`
	BotActiveRadiusKM     float64 `envconfig:"BOT_ACTIVE_RADIUS_KM" default:"10"`
	BotDefaultDistanceM   float64 `envconfig:"BOT_DEFAULT_DISTANCE_M" default:"5000"`
	BotCitySpawnIntervalS int     `envconfig:"BOT_CITY_SPAWN_INTERVAL_S" default:"30"`
	BotMaxPerHex          int     `envconfig:"BOT_MAX_PER_HEX" default:"2"`
	BotPhaseOutThreshold  int     `envconfig:"BOT_PHASE_OUT_THRESHOLD" default:"20"`
	BotSeedingMaxClaimed  float64 `envconfig:"BOT_SEEDING_MAX_CLAIMED" default:"0.60"`

	SeasonDurationDays int `envconfig:"SEASON_DURATION_DAYS" default:"60"`

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
