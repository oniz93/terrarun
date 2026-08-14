package mapbox

import (
	"context"
	"fmt"
	"net/http"
	"io"
)

type Client struct {
	token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{token: token, http: &http.Client{}}
}

func (c *Client) GetWater(ctx context.Context, lat, lng float64, radius int) ([]byte, error) {
	// Mapbox Tilequery API
	// layers: water
	// radius: in meters
	// limit: number of features (max 50)
	url := fmt.Sprintf("https://api.mapbox.com/v4/mapbox.mapbox-streets-v8/tilequery/%f,%f.json?layers=water&radius=%d&limit=50&access_token=%s", lng, lat, radius, c.token)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mapbox error: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
