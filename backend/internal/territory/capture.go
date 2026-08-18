package territory

import (
	"context"
	"fmt"
	"time"

	"github.com/terrarun/backend/internal/domain"
	"github.com/terrarun/backend/pkg/geo"
	"github.com/terrarun/backend/pkg/h3util"
)

type CaptureResult struct {
	Mode             domain.CaptureMode
	HexesCaptured    int
	HexesStolen      int
	TerritoryPoints  int
	Changes          []domain.TerritoryChange
	AffectedHexes    []domain.Hex
}

func (s *Service) CaptureRun(ctx context.Context, run *domain.Run, faction domain.Faction, points []domain.GPSPoint) (*CaptureResult, error) {
	captureMode, geometry, err := s.determineCaptureGeometry(ctx, points)
	if err != nil {
		return nil, fmt.Errorf("determine geometry: %w", err)
	}
	if captureMode == nil {
		return nil, fmt.Errorf("loop not closed and path capture disabled")
	}

	cells, err := h3util.PolygonToCells(geometry, h3util.DefaultResolution)
	if err != nil {
		return nil, fmt.Errorf("h3 polygon to cells: %w", err)
	}

	if len(cells) == 0 {
		return &CaptureResult{Mode: *captureMode}, nil
	}

	result := CaptureResult{Mode: *captureMode}

	capCfg := CaptureConfig{
		BasePoints:    s.cfg.TerritoryBasePoints,
		MaxHP:         s.cfg.HexHPCap,
		StealBonusPct: s.cfg.TerritoryStealBonusPct,
	}

	now := time.Now()
	for _, cell := range cells {
		hex, err := s.repo.GetHex(ctx, int64(cell))
		if err != nil {
			hex = nil
		}

		change := domain.TerritoryChange{
			H3Index:   int64(cell),
			RunID:     &run.ID,
			ChangedAt: now,
		}

		if hex == nil {
			lat, lng, _ := h3util.CellToLatLng(cell)
			h := domain.Hex{
				H3Index:       int64(cell),
				OwnedBy:       &faction,
				HP:            1,
				CapturedByID:  &run.ID,
				CapturedAt:    now,
				LastDecayedAt: now,
				Lat:           lat,
				Lng:           lng,
			}
			if err := s.repo.UpsertHex(ctx, &h); err != nil {
				return nil, fmt.Errorf("upsert hex: %w", err)
			}
			change.NewOwner = &faction
			change.HPAfter = 1
			result.TerritoryPoints += s.cfg.TerritoryBasePoints
			result.HexesCaptured++
			result.AffectedHexes = append(result.AffectedHexes, h)
		} else {
			lat, lng, _ := h3util.CellToLatLng(cell)
			hex.Lat = lat
			hex.Lng = lng
			trans := ComputeCaptureTransition(hex, faction, capCfg)
			change.PreviousOwner = hex.OwnedBy
			change.HPBefore = hex.HP

			if trans.Flipped {
				hex.OwnedBy = &faction
				hex.HP = 1
				hex.CapturedByID = &run.ID
				hex.CapturedAt = now
				change.NewOwner = &faction
				change.HPAfter = 1
				result.HexesStolen++
			} else {
				hex.HP = trans.ToHP
				change.NewOwner = hex.OwnedBy
				change.HPAfter = trans.ToHP
				if *hex.OwnedBy == faction {
					result.HexesCaptured++
				}
			}

			if err := s.repo.UpsertHex(ctx, hex); err != nil {
				return nil, fmt.Errorf("upsert hex: %w", err)
			}
			result.TerritoryPoints += trans.TerritoryPoints
			result.AffectedHexes = append(result.AffectedHexes, *hex)
		}

		if err := s.repo.RecordChange(ctx, &change); err != nil {
			return nil, fmt.Errorf("record change: %w", err)
		}
		result.Changes = append(result.Changes, change)
	}

	return &result, nil
}

func (s *Service) determineCaptureGeometry(ctx context.Context, points []domain.GPSPoint) (*domain.CaptureMode, [][2]float64, error) {
	if len(points) < 2 {
		return nil, nil, fmt.Errorf("need at least 2 points")
	}

	start := points[0]
	end := points[len(points)-1]
	dist := geo.Haversine(start.Lat, start.Lng, end.Lat, end.Lng)

	if dist <= s.cfg.LoopClosureRadiusM {
		simplified := geo.DouglasPeucker(points, 10.0)
		if len(simplified) < 3 {
			return nil, nil, fmt.Errorf("too few points after simplification")
		}
		polygon := geo.PointsToPolygonRing(simplified)

		if isSelfIntersecting(polygon) {
			fixed := geo.ExtractLargestSimplePolygon(polygon)
			if len(fixed) < 3 {
				if s.pathCaptureEnabled() {
					corridor := geo.BuildPathCorridor(simplified, s.cfg.PathCaptureBufferM)
					mode := domain.CaptureModePath
					return &mode, corridor, nil
				}
				return nil, nil, fmt.Errorf("invalid polygon and path capture disabled")
			}
			polygon = fixed
		}

		mode := domain.CaptureModePolygon
		return &mode, polygon, nil
	}

	if s.pathCaptureEnabled() {
		simplified := geo.DouglasPeucker(points, 10.0)
		corridor := geo.BuildPathCorridor(simplified, s.cfg.PathCaptureBufferM)
		mode := domain.CaptureModePath
		return &mode, corridor, nil
	}

	return nil, nil, fmt.Errorf("loop not closed and path capture disabled")
}

func (s *Service) pathCaptureEnabled() bool {
	if s.redis == nil {
		return s.cfg.PathCaptureEnabled
	}
	val, err := s.redis.Get(context.Background(), "config:path_capture_enabled").Result()
	if err == nil {
		return val == "true"
	}
	return s.cfg.PathCaptureEnabled
}

func isSelfIntersecting(vertices [][2]float64) bool {
	if len(vertices) < 4 {
		return false
	}
	n := len(vertices)
	for i := 0; i < n; i++ {
		p1 := vertices[i]
		p2 := vertices[(i+1)%n]
		for j := i + 2; j < n; j++ {
			if (j+1)%n == i {
				continue
			}
			p3 := vertices[j]
			p4 := vertices[(j+1)%n]
			if segmentsIntersect(p1, p2, p3, p4) {
				return true
			}
		}
	}
	return false
}

func segmentsIntersect(p1, p2, p3, p4 [2]float64) bool {
	d1 := cross(p3, p4, p1)
	d2 := cross(p3, p4, p2)
	d3 := cross(p1, p2, p3)
	d4 := cross(p1, p2, p4)

	if ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0)) {
		return true
	}
	return false
}

func cross(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}
