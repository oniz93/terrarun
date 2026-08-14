package mapbox

import (
	"context"
	"testing"
)

func TestClient_GetWater(t *testing.T) {
	c := NewClient("fake_token")
	_, err := c.GetWater(context.Background(), 49.2253, -123.0050, 3000)
	if err == nil {
		t.Error("Expected error for fake token")
	}
}
