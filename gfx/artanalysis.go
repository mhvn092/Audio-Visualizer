package gfx

import (
	"image"
	"math"
	"sort"
)

type ArtFeatures struct {
	DominantColors [3][3]float32 // RGB 0..1 per color: [0]=Vibrant Accent, [1]=Mid Tone, [2]=Atmosphere/Shadow
	Brightness     float32       // Mean luminance 0..1
	Warmth         float32       // -1.0 (cold/blue) to +1.0 (warm/amber)
	Complexity     float32       // Edge density 0..1
	Contrast       float32       // Stddev of luminance 0..1
}

// DefaultArtFeatures returns fallback features if analysis fails or no image is available
func DefaultArtFeatures() ArtFeatures {
	return ArtFeatures{
		DominantColors: [3][3]float32{
			{0.90, 0.55, 0.25}, // Accent (Vibrant Warm Amber)
			{0.30, 0.45, 0.70}, // Midtone (Deep Blue)
			{0.12, 0.15, 0.25}, // Atmosphere (Dark Void)
		},
		Brightness: 0.45,
		Warmth:     0.2,
		Complexity: 0.5,
		Contrast:   0.4,
	}
}

// AnalyzeArtwork analyzes an image.Image and extracts visual features.
func AnalyzeArtwork(img image.Image) ArtFeatures {
	if img == nil {
		return DefaultArtFeatures()
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return DefaultArtFeatures()
	}

	// Downsample to 32x32 grid for fast computation
	targetSize := 32
	grid := make([][3]float32, targetSize*targetSize) // R, G, B in 0..1
	lumaGrid := make([]float32, targetSize*targetSize)

	var totalLuma float32
	var totalWarmth float32

	stepX := float64(w) / float64(targetSize)
	stepY := float64(h) / float64(targetSize)

	idx := 0
	for ty := 0; ty < targetSize; ty++ {
		iy := bounds.Min.Y + int(float64(ty)*stepY)
		for tx := 0; tx < targetSize; tx++ {
			ix := bounds.Min.X + int(float64(tx)*stepX)

			r, g, b, _ := img.At(ix, iy).RGBA()
			rf := float32(r) / 65535.0
			gf := float32(g) / 65535.0
			bf := float32(b) / 65535.0

			grid[idx] = [3]float32{rf, gf, bf}

			luma := 0.299*rf + 0.587*gf + 0.114*bf
			lumaGrid[idx] = luma
			totalLuma += luma

			warmth := (rf + gf*0.5 - bf*1.2)
			totalWarmth += warmth

			idx++
		}
	}

	totalPixels := float32(targetSize * targetSize)
	avgLuma := totalLuma / totalPixels
	avgWarmth := totalWarmth / totalPixels
	if avgWarmth > 1.0 {
		avgWarmth = 1.0
	} else if avgWarmth < -1.0 {
		avgWarmth = -1.0
	}

	// Contrast: Stddev of luminance
	var varLuma float32
	for _, l := range lumaGrid {
		diff := l - avgLuma
		varLuma += diff * diff
	}
	contrast := float32(math.Sqrt(float64(varLuma / totalPixels)))

	// Complexity: 3x3 Sobel edge magnitude average
	var edgeSum float32
	for y := 1; y < targetSize-1; y++ {
		for x := 1; x < targetSize-1; x++ {
			gx := -lumaGrid[(y-1)*targetSize+(x-1)] + lumaGrid[(y-1)*targetSize+(x+1)] +
				-2*lumaGrid[y*targetSize+(x-1)] + 2*lumaGrid[y*targetSize+(x+1)] +
				-lumaGrid[(y+1)*targetSize+(x-1)] + lumaGrid[(y+1)*targetSize+(x+1)]

			gy := -lumaGrid[(y-1)*targetSize+(x-1)] - 2*lumaGrid[(y-1)*targetSize+x] - lumaGrid[(y-1)*targetSize+(x+1)] +
				lumaGrid[(y+1)*targetSize+(x-1)] + 2*lumaGrid[(y+1)*targetSize+x] + lumaGrid[(y+1)*targetSize+(x+1)]

			edgeMag := float32(math.Sqrt(float64(gx*gx + gy*gy)))
			edgeSum += edgeMag
		}
	}
	complexity := edgeSum / float32((targetSize-2)*(targetSize-2))
	if complexity > 1.0 {
		complexity = 1.0
	}

	// K-Means clustering for 3 dominant colors
	centroids := [3][3]float32{
		{0.80, 0.40, 0.20}, // Warm
		{0.30, 0.50, 0.70}, // Cool
		{0.15, 0.15, 0.20}, // Dark
	}

	for iter := 0; iter < 10; iter++ {
		sums := [3][3]float32{}
		counts := [3]int{}

		for _, p := range grid {
			bestCluster := 0
			bestDist := float32(1e9)

			for c := 0; c < 3; c++ {
				dr := p[0] - centroids[c][0]
				dg := p[1] - centroids[c][1]
				db := p[2] - centroids[c][2]
				dist := dr*dr + dg*dg + db*db
				if dist < bestDist {
					bestDist = dist
					bestCluster = c
				}
			}

			sums[bestCluster][0] += p[0]
			sums[bestCluster][1] += p[1]
			sums[bestCluster][2] += p[2]
			counts[bestCluster]++
		}

		for c := 0; c < 3; c++ {
			if counts[c] > 0 {
				centroids[c][0] = sums[c][0] / float32(counts[c])
				centroids[c][1] = sums[c][1] / float32(counts[c])
				centroids[c][2] = sums[c][2] / float32(counts[c])
			}
		}
	}

	// Sort centroids by saturation/vibrancy so:
	// [0] = Highest saturation / brightest accent (for highlights & glow)
	// [1] = Midtone dominant color
	// [2] = Base background atmosphere color
	type colorCluster struct {
		color      [3]float32
		saturation float32
		luma       float32
	}
	clusters := make([]colorCluster, 3)
	for i := 0; i < 3; i++ {
		r, g, b := centroids[i][0], centroids[i][1], centroids[i][2]
		maxC := math.Max(float64(r), math.Max(float64(g), float64(b)))
		minC := math.Min(float64(r), math.Min(float64(g), float64(b)))
		sat := float32(0.0)
		if maxC > 0.001 {
			sat = float32((maxC - minC) / maxC)
		}
		luma := 0.299*r + 0.587*g + 0.114*b

		// Ensure colors aren't completely black
		if luma < 0.1 {
			r += 0.15
			g += 0.15
			b += 0.15
		}

		clusters[i] = colorCluster{color: [3]float32{r, g, b}, saturation: sat, luma: luma}
	}

	// Sort: highest (saturation * 0.7 + luma * 0.3) first
	sort.Slice(clusters, func(i, j int) bool {
		scoreI := clusters[i].saturation*0.7 + clusters[i].luma*0.3
		scoreJ := clusters[j].saturation*0.7 + clusters[j].luma*0.3
		return scoreI > scoreJ
	})

	var sortedColors [3][3]float32
	sortedColors[0] = clusters[0].color // Accent
	sortedColors[1] = clusters[1].color // Midtone
	sortedColors[2] = clusters[2].color // Atmosphere

	return ArtFeatures{
		DominantColors: sortedColors,
		Brightness:     avgLuma,
		Warmth:         avgWarmth,
		Complexity:     complexity,
		Contrast:       contrast,
	}
}
