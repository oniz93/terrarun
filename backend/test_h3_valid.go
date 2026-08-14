package main
import (
	"fmt"
	"github.com/uber/h3-go/v4"
)
func main() {
	idx := h3.Cell(626719328019967999)
	latLng, _ := h3.CellToLatLng(idx)
	fmt.Printf("Center: %v, %v\n", latLng.Lat, latLng.Lng)
}
