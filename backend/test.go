package main
import (
	"fmt"
	"github.com/uber/h3-go/v4"
)
func main() {
	c, _ := h3.LatLngToCell(h3.LatLng{Lat: 49.208, Lng: -122.910}, 6)
	fmt.Printf("%d", int64(c))
}
