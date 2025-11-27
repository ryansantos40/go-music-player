package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
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

func ImageToBraille(img image.Image, width, height int) string {
	if img == nil {
		return ""
	}

	pixelWidth := uint(width * 2)
	pixelHeight := uint(height * 4)

	resized := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)

	dithered := floydSteinbergDither(resized)

	bounds := dithered.Bounds()
	var result strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			brailleChar := getBrailleCharFromDithered(dithered, x, y)
			result.WriteRune(brailleChar)
		}
		result.WriteRune('\n')
	}

	return result.String()
}

func ImageToBrailleColored(img image.Image, width, height int) string {
	if img == nil {
		return " "
	}

	pixelWidth := uint(width * 2)
	pixelHeight := uint(height * 4)

	resized := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)

	dithered := floydSteinbergDither(resized)

	bounds := resized.Bounds()

	var result strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			brailleChar := getBrailleChar(dithered, x, y)

			avgColor := getAverageColor(resized, x, y)

			r, g, b, _ := avgColor.RGBA()
			colorCode := fmt.Sprintf("\033[38;2;%d;%d;%dm", r>>8, g>>8, b>>8)
			resetCode := "\033[0m"

			result.WriteString(colorCode)
			result.WriteRune(brailleChar)
			result.WriteString(resetCode)
		}
		result.WriteRune('\n')
	}

	return result.String()
}

func ImageToBraille256(img image.Image, width, height int) string {
	if img == nil {
		return ""
	}

	pixelWidth := uint(width * 2)
	pixelHeight := uint(height * 4)

	resized := resize.Resize(pixelWidth, pixelHeight, img, resize.Lanczos3)
	dithered := floydSteinbergDither(resized)

	bounds := resized.Bounds()
	var result strings.Builder

	for y := bounds.Min.Y; y < bounds.Max.Y; y += 4 {
		for x := bounds.Min.X; x < bounds.Max.X; x += 2 {
			brailleChar := getBrailleCharFromDithered(dithered, x, y)
			avgColor := getAverageColor(resized, x, y)

			colorCode := rgbTo256(avgColor)
			ansiCode := fmt.Sprintf("\033[38;5;%dm", colorCode)
			resetCode := "\033[0m"

			result.WriteString(ansiCode)
			result.WriteRune(brailleChar)
			result.WriteString(resetCode)
		}
		result.WriteRune('\n')
	}

	return result.String()
}

func floydSteinbergDither(img image.Image) *image.Gray {
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			grayValue := colorToGrayScale(img.At(x, y))
			gray.SetGray(x, y, color.Gray{Y: grayValue})
		}
	}

	errors := make([][]float64, height)
	for i := range errors {
		errors[i] = make([]float64, width)
	}

	result := image.NewGray(bounds)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			oldPixel := float64(gray.GrayAt(x+bounds.Min.X, y+bounds.Min.Y).Y) + errors[y][x]

			var newPixel uint8
			if oldPixel < 127 {
				newPixel = 255
			} else {
				newPixel = 0
			}

			result.SetGray(x+bounds.Min.X, y+bounds.Min.Y, color.Gray{Y: newPixel})

			quantError := oldPixel - float64(newPixel)

			if x+1 < width {
				errors[y][x+1] += quantError * 7 / 16
			}
			if y+1 < height {
				if x > 0 {
					errors[y+1][x-1] += quantError * 3 / 16
				}
				errors[y+1][x] += quantError * 5 / 16
				if x+1 < width {
					errors[y+1][x+1] += quantError * 1 / 16
				}
			}
		}
	}
	return result
}

func adaptiveThreshold(img image.Image, x, y, blockSize int) uint8 {
	bounds := img.Bounds()
	var sum float64
	var count int

	halfBlock := blockSize / 2

	for dy := -halfBlock; dy <= halfBlock; dy++ {
		for dx := -halfBlock; dx <= halfBlock; dx++ {
			nx := x + dx
			ny := y + dy
			if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
				sum += float64(colorToGrayScale(img.At(nx, ny)))
				count++
			}
		}
	}

	if count == 0 {
		return 128
	}

	threshold := sum/float64(count) - 10
	if threshold < 0 {
		threshold = 0
	}

	return uint8(threshold)
}

func getBrailleCharFromDithered(img *image.Gray, startX, startY int) rune {
	bounds := img.Bounds()
	var brailleValue int

	for row := 0; row < 4; row++ {
		for col := 0; col < 2; col++ {
			x := startX + col
			y := startY + row

			if x >= bounds.Max.X || y >= bounds.Max.Y {
				continue
			}

			// Pixels pretos (0) ativam o ponto Braille
			if img.GrayAt(x, y).Y < 128 {
				brailleValue |= brailleMap[row][col]
			}
		}
	}

	return rune(0x2800 + brailleValue)
}

func getAverageColor(img image.Image, startX, startY int) color.Color {
	bounds := img.Bounds()
	var rSum, gSum, bSum uint64
	var count uint64

	for row := 0; row < 4; row++ {
		for col := 0; col < 2; col++ {
			x := startX + col
			y := startY + row

			if x >= bounds.Max.X || y >= bounds.Max.Y {
				continue
			}

			r, g, b, _ := img.At(x, y).RGBA()
			rSum += uint64(r >> 8)
			gSum += uint64(g >> 8)
			bSum += uint64(b >> 8)
			count++
		}
	}

	if count == 0 {
		return color.RGBA{0, 0, 0, 255}
	}

	return color.RGBA{
		R: uint8(rSum / count),
		G: uint8(gSum / count),
		B: uint8(bSum / count),
		A: 255,
	}
}

func rgbTo256(c color.Color) int {
	r, g, b, _ := c.RGBA()
	r8 := int(r >> 8)
	g8 := int(g >> 8)
	b8 := int(b >> 8)

	if r8 == g8 && g8 == b8 {
		if r8 < 8 {
			return 16
		}
		if r8 > 248 {
			return 231
		}
		return int(((float64(r8)-8)/247)*24) + 232
	}

	r6 := int((float64(r8) / 255) * 5)
	g6 := int((float64(g8) / 255) * 5)
	b6 := int((float64(b8) / 255) * 5)

	return 16 + (36 * r6) + (6 * g6) + b6
}

func getBrailleChar(img image.Image, startX, startY int) rune {
	bounds := img.Bounds()
	threshold := uint8(128)

	var brailleValue int

	for row := 0; row < 4; row++ {
		for col := 0; col < 2; col++ {
			x := startX + col
			y := startY + row

			if x >= bounds.Max.X || y >= bounds.Max.Y {
				continue
			}

			gray := colorToGrayScale(img.At(x, y))

			if (gray) < threshold {
				brailleValue |= brailleMap[row][col]
			}
		}
	}

	return rune(0x2800 + brailleValue)
}

func colorToGrayScale(c color.Color) uint8 {
	r, g, b, _ := c.RGBA()

	gray := (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 256
	return uint8(gray)
}

func GetAlbumArtBraille(path string, width, height int) string {
	img, err := ExtractAlbumArt(path)
	if err != nil || img == nil {
		return " "
	}

	return ImageToBraille(img, width, height)
}

func GetAlbumArtBrailleColored(path string, width, height int) string {
	img, err := ExtractAlbumArt(path)
	if err != nil || img == nil {
		return ""
	}

	return ImageToBrailleColored(img, width, height)
}

func GetAlbumArtBraille256(path string, width, height int) string {
	img, err := ExtractAlbumArt(path)
	if err != nil || img == nil {
		return ""
	}

	return ImageToBraille256(img, width, height)
}
