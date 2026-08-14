package domain

import (
	"time"

	"github.com/google/uuid"
)

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
	ID                 uuid.UUID    `json:"id"`
	UserID             uuid.UUID    `json:"user_id"`
	StartTime          time.Time    `json:"start_time"`
	EndTime            *time.Time   `json:"end_time,omitempty"`
	Status             RunStatus    `json:"status"`
	DistanceM          *float64     `json:"distance_m,omitempty"`
	ElevationGainM     *float64     `json:"elevation_gain_m,omitempty"`
	AvgPaceSPerKM      *float64     `json:"avg_pace_s_per_km,omitempty"`
	MaxSpeedKMH        *float64     `json:"max_speed_kmh,omitempty"`
	AvgHRBPM           *int         `json:"avg_hr_bpm,omitempty"`
	MaxHRBPM           *int         `json:"max_hr_bpm,omitempty"`
	Calories           *int         `json:"calories,omitempty"`
	ClosedLoop         *bool        `json:"closed_loop,omitempty"`
	CaptureMode        *CaptureMode `json:"capture_mode,omitempty"`
	LoopSnapDistanceM  *float64     `json:"loop_snap_distance_m,omitempty"`
	PathBufferRadiusM  *float64     `json:"path_buffer_radius_m,omitempty"`
	Polyline           *string      `json:"polyline,omitempty"`
	TerritoryPoints    int          `json:"territory_points"`
	RunnerPoints       int          `json:"runner_points"`
	SocialRun          bool         `json:"social_run"`
	SocialLeader       *uuid.UUID   `json:"social_leader,omitempty"`
	SocialParticipants []uuid.UUID  `json:"social_participants,omitempty"`
	HealthSource       *HealthSource `json:"health_source,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
}

func (r *Run) Faction() *Faction {
	return nil
}

type GPSPoint struct {
	RunID              uuid.UUID  `json:"-"`
	Timestamp          time.Time  `json:"timestamp"`
	Lat                float64    `json:"lat"`
	Lng                float64    `json:"lng"`
	Altitude           *float64   `json:"altitude,omitempty"`
	Speed              *float64   `json:"speed,omitempty"`
	HorizontalAccuracy *float64   `json:"horizontal_accuracy,omitempty"`
	HeartRate          *int       `json:"heart_rate,omitempty"`
}
