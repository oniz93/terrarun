package territory

import "testing"

func TestSegmentsIntersect(t *testing.T) {
	tests := []struct {
		name string
		p1   [2]float64
		p2   [2]float64
		p3   [2]float64
		p4   [2]float64
		want bool
	}{
		{
			name: "crossing segments",
			p1:   [2]float64{0, 0},
			p2:   [2]float64{10, 10},
			p3:   [2]float64{0, 10},
			p4:   [2]float64{10, 0},
			want: true,
		},
		{
			name: "parallel, no intersect",
			p1:   [2]float64{0, 0},
			p2:   [2]float64{10, 0},
			p3:   [2]float64{0, 10},
			p4:   [2]float64{10, 10},
			want: false,
		},
		{
			name: "non-adjacent gap",
			p1:   [2]float64{0, 0},
			p2:   [2]float64{10, 0},
			p3:   [2]float64{20, 10},
			p4:   [2]float64{30, 10},
			want: false,
		},
		{
			name: "touching at endpoint",
			p1:   [2]float64{0, 0},
			p2:   [2]float64{10, 10},
			p3:   [2]float64{10, 10},
			p4:   [2]float64{20, 0},
			want: false,
		},
		{
			name: "vertical cross horizontal",
			p1:   [2]float64{5, 0},
			p2:   [2]float64{5, 10},
			p3:   [2]float64{0, 5},
			p4:   [2]float64{10, 5},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := segmentsIntersect(tt.p1, tt.p2, tt.p3, tt.p4)
			if got != tt.want {
				t.Errorf("segmentsIntersect(%v,%v,%v,%v) = %v, want %v",
					tt.p1, tt.p2, tt.p3, tt.p4, got, tt.want)
			}
		})
	}
}

func TestIsSelfIntersecting(t *testing.T) {
	t.Run("simple square, no self-intersection", func(t *testing.T) {
		verts := [][2]float64{
			{0, 0}, {10, 0}, {10, 10}, {0, 10},
		}
		if isSelfIntersecting(verts) {
			t.Error("expected no self-intersection for simple square")
		}
	})

	t.Run("bowtie, is self-intersecting", func(t *testing.T) {
		verts := [][2]float64{
			{0, 0}, {10, 10}, {10, 0}, {0, 10},
		}
		if !isSelfIntersecting(verts) {
			t.Error("expected self-intersection for bowtie")
		}
	})

	t.Run("3 vertices, no self-intersection", func(t *testing.T) {
		verts := [][2]float64{
			{0, 0}, {10, 0}, {5, 10},
		}
		if isSelfIntersecting(verts) {
			t.Error("triangle cannot self-intersect")
		}
	})

	t.Run("pentagon, no self-intersection", func(t *testing.T) {
		verts := [][2]float64{
			{0, 0}, {10, 0}, {12, 5}, {10, 10}, {0, 10},
		}
		if isSelfIntersecting(verts) {
			t.Error("expected no self-intersection for simple pentagon")
		}
	})

	t.Run("single segment", func(t *testing.T) {
		verts := [][2]float64{
			{0, 0}, {10, 10},
		}
		if isSelfIntersecting(verts) {
			t.Error("single segment cannot self-intersect")
		}
	})
}

func TestCross(t *testing.T) {
	result := cross(
		[2]float64{0, 0},
		[2]float64{10, 0},
		[2]float64{5, 5},
	)
	if result != 50 {
		t.Errorf("cross = %f, want 50", result)
	}
}
