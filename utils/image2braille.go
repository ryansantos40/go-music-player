package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"github.com/dhowden/tag"
	"github.com/nfnt/resize"
)

var brailleMap = [4][2]int{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

type pixelInfo struct {
	x, y    int
	lum     float64
	r, g, b uint8
}

func ExtractAlbumArt(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	picture := m.Picture()
	if picture == nil {
		return nil, nil
	}

	img, _, err := image.Decode(bytes.NewReader(picture.Data))
	if err != nil {
		return nil, err
	}

	return img, nil
}

func normalizeImage(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	normalized := image.NewRGBA(bounds)

	var minLum, maxLum float64 = 255, 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			lum := getLuminance(img.At(x, y))
			if lum < minLum {
				minLum = lum
			}
			if lum > maxLum {
				maxLum = lum
			}
		}
	}

	lumRange := maxLum - minLum
	if lumRange < 1 {
		lumRange = 1
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			rNorm := normalizeChannel(float64(r>>8), minLum, lumRange)
			gNorm := normalizeChannel(float64(g>>8), minLum, lumRange)
			bNorm := normalizeChannel(float64(b>>8), minLum, lumRange)

			normalized.Set(x, y, color.RGBA{
				R: uint8(rNorm),
				G: uint8(gNorm),
				B: uint8(bNorm),
				A: uint8(a >> 8),
			})
		}
	}

	return normalized
}

func normalizeChannel(val, minLum, lumRange float64) float64 {
	normalized := ((val - minLum) / lumRange) * 255
	if normalized < 0 {
		return 0
	}
	if normalized > 255 {
		return 255
	}
	return normalized
}

func getLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}

func enhanceEdges(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	kernel := [][]float64{
		{0, -0.5, 0},
		{-0.5, 3, -0.5},
		{0, -0.5, 0},
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var rSum, gSum, bSum float64

			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					px := clamp(x+kx, bounds.Min.X, bounds.Max.X-1)
					py := clamp(y+ky, bounds.Min.Y, bounds.Max.Y-1)

					r, g, b, _ := img.At(px, py).RGBA()
					weight := kernel[ky+1][kx+1]

					rSum += float64(r>>8) * weight
					gSum += float64(g>>8) * weight
					bSum += float64(b>>8) * weight
				}
			}

			result.Set(x, y, color.RGBA{
				R: uint8(clampFloat(rSum, 0, 255)),
				G: uint8(clampFloat(gSum, 0, 255)),
				B: uint8(clampFloat(bSum, 0, 255)),
				A: 255,
			})
		}
	}

	return result
}

func rgbToHsl(r, g, b float64) (h, s, l float64) {
	r /= 255
	g /= 255
	b /= 255

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))

	l = (max + min) / 2

	if max == min {
		h = 0
		s = 0
	} else {
		d := max - min

		if l > 0.5 {
			s = d / (2 - max - min)
		} else {
			s = d / (max + min)
		}

		switch max {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}
		h /= 6
	}

	return h, s, l
}

func hslToRgb(h, s, l float64) (r, g, b float64) {
	if s == 0 {
		r = l * 255
		g = l * 255
		b = l * 255
		return
	}

	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q

	r = hueToRgb(p, q, h+1.0/3.0) * 255
	g = hueToRgb(p, q, h) * 255
	b = hueToRgb(p, q, h-1.0/3.0) * 255

	return
}

func hueToRgb(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func clampFloat(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func atkinsonDither(img image.Image) *image.Gray {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Converter para escala de cinza com buffer de float
	pixels := make([][]float64, height)
	for y := 0; y < height; y++ {
		pixels[y] = make([]float64, width)
		for x := 0; x < width; x++ {
			pixels[y][x] = getLuminance(img.At(x+bounds.Min.X, y+bounds.Min.Y))
		}
	}

	result := image.NewGray(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldPixel := pixels[y][x]

			threshold := getLocalThreshold(pixels, x, y, width, height)

			var newPixel float64
			if oldPixel > threshold {
				newPixel = 255
			} else {
				newPixel = 0
			}

			result.SetGray(x+bounds.Min.X, y+bounds.Min.Y, color.Gray{Y: uint8(newPixel)})

			quantError := (oldPixel - newPixel) / 8

			if x+1 < width {
				pixels[y][x+1] += quantError
			}
			if x+2 < width {
				pixels[y][x+2] += quantError
			}
			if y+1 < height {
				if x > 0 {
					pixels[y+1][x-1] += quantError
				}
				pixels[y+1][x] += quantError
				if x+1 < width {
					pixels[y+1][x+1] += quantError
				}
			}
			if y+2 < height {
				pixels[y+2][x] += quantError
			}
		}
	}

	return result
}

func getLocalThreshold(pixels [][]float64, x, y, width, height int) float64 {
	blockSize := 5
	half := blockSize / 2

	var sum float64
	var count int

	for dy := -half; dy <= half; dy++ {
		for dx := -half; dx <= half; dx++ {
			nx := x + dx
			ny := y + dy
			if nx >= 0 && nx < width && ny >= 0 && ny < height {
				sum += pixels[ny][nx]
				count++
			}
		}
	}

	if count == 0 {
		return 128
	}

	return (sum / float64(count)) * 0.95
}

func imageToBraille(img image.Image, width, height int) string {
	if img == nil {
		return ""
	}

	pixelWidth := uint(width * 2)
	pixelHeight := uint(height * 4)

	resized := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)
	normalized := normalizeImage(resized)
	enhanced := enhanceEdges(normalized)
	dithered := atkinsonDither(enhanced)

	bounds := dithered.Bounds()
	var result strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			brailleChar := getBrailleCharFromGray(dithered, x, y)
			result.WriteRune(brailleChar)
		}
		result.WriteRune('\n')
	}

	return result.String()
}

func imageToBrailleColored(img image.Image, width, height int) string {
	if img == nil {
		return ""
	}

	pixelWidth := uint(width * 2)
	pixelHeight := uint(height * 4)

	resized := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)

	boosted := boostColors(resized, 1.4, 1.2)

	bounds := boosted.Bounds()
	var result strings.Builder

	var lastR, lastG, lastB uint8 = 0, 0, 0
	firstChar := true

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			brailleChar, avgColor := getBrailleCharForColoredBlockWeighted(boosted, x, y)

			r8, g8, b8 := avgColor.R, avgColor.G, avgColor.B

			if firstChar || colorDiff(r8, g8, b8, lastR, lastG, lastB) > 20 {
				result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm", r8, g8, b8))
				lastR, lastG, lastB = r8, g8, b8
				firstChar = false
			}

			result.WriteRune(brailleChar)
		}
		result.WriteString("\033[0m")
		result.WriteRune('\n')
		firstChar = true
	}

	return result.String()
}

func imageToBraille256(img image.Image, width, height int) string {
	if img == nil {
		return ""
	}

	pixelWidth := uint(width * 2)
	pixelHeight := uint(height * 4)

	resized := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)
	boosted := boostColors(resized, 1.4, 1.2)

	bounds := boosted.Bounds()
	var result strings.Builder

	lastColor := -1

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			brailleChar, avgColor := getBrailleCharForColoredBlockWeighted(boosted, x, y)

			colorCode := rgbTo256(avgColor)

			if colorCode != lastColor {
				result.WriteString(fmt.Sprintf("\033[38;5;%dm", colorCode))
				lastColor = colorCode
			}

			result.WriteRune(brailleChar)
		}
		result.WriteString("\033[0m")
		result.WriteRune('\n')
		lastColor = -1
	}

	return result.String()
}

func getBrailleCharFromGray(img *image.Gray, startX, startY int) rune {
	bounds := img.Bounds()
	var brailleValue int

	for row := 0; row < 4; row++ {
		for col := 0; col < 2; col++ {
			x := startX + col
			y := startY + row

			if x >= bounds.Max.X || y >= bounds.Max.Y {
				continue
			}

			// Pixels escuros (< 128) ativam o ponto Braille
			if img.GrayAt(x, y).Y < 128 {
				brailleValue |= brailleMap[row][col]
			}
		}
	}

	return rune(0x2800 + brailleValue)
}

func getBrailleCharForColoredBlockWeighted(img image.Image, startX, startY int) (rune, color.RGBA) {
	bounds := img.Bounds()

	type pixelData struct {
		row, col int
		lum      float64
		r, g, b  uint8
	}

	var pixels []pixelData

	for row := 0; row < 4; row++ {
		for col := 0; col < 2; col++ {
			x := startX + col
			y := startY + row

			if x >= bounds.Max.X || y >= bounds.Max.Y {
				continue
			}

			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

			pixels = append(pixels, pixelData{
				row: row, col: col,
				lum: getLuminance(c),
				r:   r8, g: g8, b: b8,
			})
		}
	}

	if len(pixels) == 0 {
		return '⠀', color.RGBA{0, 0, 0, 255}
	}

	var minLum, maxLum, sumLum float64 = 255, 0, 0
	for _, p := range pixels {
		if p.lum < minLum {
			minLum = p.lum
		}
		if p.lum > maxLum {
			maxLum = p.lum
		}
		sumLum += p.lum
	}

	lumRange := maxLum - minLum
	avgLum := sumLum / float64(len(pixels))

	var threshold float64
	if lumRange < 20 {
		threshold = 128
	} else {
		threshold = avgLum
	}

	var brailleValue int
	var weightedR, weightedG, weightedB float64
	var totalWeight float64

	for _, p := range pixels {
		if p.lum < threshold {
			brailleValue |= brailleMap[p.row][p.col]
		}

		saturation := getSaturation(float64(p.r), float64(p.g), float64(p.b))
		weight := 1.0 + saturation*2.0

		weightedR += float64(p.r) * weight
		weightedG += float64(p.g) * weight
		weightedB += float64(p.b) * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		totalWeight = 1
	}

	avgColor := color.RGBA{
		R: uint8(weightedR / totalWeight),
		G: uint8(weightedG / totalWeight),
		B: uint8(weightedB / totalWeight),
		A: 255,
	}

	return rune(0x2800 + brailleValue), avgColor
}

func getSaturation(r, g, b float64) float64 {
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))

	if max == 0 {
		return 0
	}

	return (max - min) / max
}

func boostColors(img image.Image, saturationFactor, contrastFactor float64) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	var sumLum float64
	var count int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sumLum += getLuminance(img.At(x, y))
			count++
		}
	}
	avgLum := sumLum / float64(count)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			r8, g8, b8 := float64(r>>8), float64(g>>8), float64(b>>8)

			h, s, l := rgbToHsl(r8, g8, b8)
			s = clampFloat(s*saturationFactor, 0, 1)
			r8, g8, b8 = hslToRgb(h, s, l)

			r8 = clampFloat(((r8-avgLum)*contrastFactor)+avgLum, 0, 255)
			g8 = clampFloat(((g8-avgLum)*contrastFactor)+avgLum, 0, 255)
			b8 = clampFloat(((b8-avgLum)*contrastFactor)+avgLum, 0, 255)

			result.Set(x, y, color.RGBA{
				R: uint8(r8),
				G: uint8(g8),
				B: uint8(b8),
				A: uint8(a >> 8),
			})
		}
	}

	return result
}

func colorDiff(r1, g1, b1, r2, g2, b2 uint8) int {
	dr := int(r1) - int(r2)
	dg := int(g1) - int(g2)
	db := int(b1) - int(b2)

	if dr < 0 {
		dr = -dr
	}
	if dg < 0 {
		dg = -dg
	}
	if db < 0 {
		db = -db
	}

	return dr + dg + db
}

func rgbTo256(c color.Color) int {
	r, g, b, _ := c.RGBA()
	r8 := int(r >> 8)
	g8 := int(g >> 8)
	b8 := int(b >> 8)

	// Verificar se é tom de cinza
	if absDiff(r8, g8) < 10 && absDiff(g8, b8) < 10 {
		gray := (r8 + g8 + b8) / 3
		if gray < 8 {
			return 16
		}
		if gray > 248 {
			return 231
		}
		return int(math.Round(float64(gray-8)/247*24)) + 232
	}

	// Converter para cubo de cores 6x6x6
	r6 := int(math.Round(float64(r8) / 255 * 5))
	g6 := int(math.Round(float64(g8) / 255 * 5))
	b6 := int(math.Round(float64(b8) / 255 * 5))

	return 16 + (36 * r6) + (6 * g6) + b6
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func GetAlbumArtBraille(path string, width, height int) string {
	img, err := ExtractAlbumArt(path)
	if err != nil || img == nil {
		return ""
	}

	return imageToBraille(img, width, height)
}

func GetAlbumArtBrailleColored(path string, width, height int) string {
	img, err := ExtractAlbumArt(path)
	if err != nil || img == nil {
		return ""
	}

	return imageToBrailleColored(img, width, height)
}

func GetAlbumArtBraille256(path string, width, height int) string {
	img, err := ExtractAlbumArt(path)
	if err != nil || img == nil {
		return ""
	}

	return imageToBraille256(img, width, height)
}
