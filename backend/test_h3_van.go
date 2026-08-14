package main
import (
	"fmt"
	"github.com/uber/h3-go/v4"
)
func main() {
	lat := 49.034565721377476
	lng := -123.16559626696728
	cell, _ := h3.LatLngToCell(h3.LatLng{Lat: lat, Lng: lng}, 11)
	fmt.Printf("Hex: %x\n", cell)
	fmt.Printf("Int64: %d\n", int64(cell))
}
