package territory

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.GetViewportHexes)
	r.Get("/stats", h.GetRegionStats)
	return r
}

func (h *Handler) GetViewportHexes(w http.ResponseWriter, r *http.Request) {
	swLat, _ := strconv.ParseFloat(r.URL.Query().Get("sw_lat"), 64)
	swLng, _ := strconv.ParseFloat(r.URL.Query().Get("sw_lng"), 64)
	neLat, _ := strconv.ParseFloat(r.URL.Query().Get("ne_lat"), 64)
	neLng, _ := strconv.ParseFloat(r.URL.Query().Get("ne_lng"), 64)

	if swLat == 0 && swLng == 0 && neLat == 0 && neLng == 0 {
		respondError(w, http.StatusBadRequest, "bbox params required: sw_lat, sw_lng, ne_lat, ne_lng")
		return
	}

	hexes, err := h.svc.GetViewportHexes(r.Context(), swLat, swLng, neLat, neLng)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get hexes")
		return
	}

	type hexResponse struct {
		// Serialized as a string: H3 indexes are unsigned 64-bit values that
		// exceed JavaScript's safe-integer range (2^53), so a JSON number
		// would lose precision on the web client.
		H3Index   string  `json:"h3_index"`
		OwnedBy   *string `json:"owned_by"`
		HP        int     `json:"hp"`
		Lat       float64 `json:"center_lat"`
		Lng       float64 `json:"center_lng"`
	}

	resp := make([]hexResponse, len(hexes))
	for i, h := range hexes {
		var ownedBy *string
		if h.OwnedBy != nil {
			s := string(*h.OwnedBy)
			ownedBy = &s
		}
		resp[i] = hexResponse{
			H3Index: strconv.FormatUint(uint64(h.H3Index), 10),
			OwnedBy: ownedBy,
			HP:      h.HP,
			Lat:     h.Lat,
			Lng:     h.Lng,
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"hexes": resp,
	})
}

func (h *Handler) GetRegionStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetRegionStats(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
