package auth

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"github.com/mdp/qrterminal/v3"
)

// decodeQRText reads the payload string encoded in a QR-code image.
//
// The login page renders the QR as an anti-aliased raster (often with a logo
// overlaid in the center). Instead of resampling that raster into terminal
// cells — which never aligns to module boundaries and produces a smeared,
// unscannable grid — we recover the exact payload and re-encode it cleanly.
func decodeQRText(img image.Image) (string, error) {
	img = prepareForDecode(img)

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("prepare QR bitmap: %w", err)
	}

	// TryHarder trades a little speed for tolerance of the logo overlay and
	// the anti-aliased edges in the screenshot.
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}

	result, err := qrcode.NewQRCodeReader().Decode(bmp, hints)
	if err != nil {
		return "", fmt.Errorf("decode QR code: %w", err)
	}
	return result.GetText(), nil
}

// prepareForDecode makes a raw QR screenshot easier for the decoder to lock
// onto: it upscales small images with nearest-neighbor sampling (keeping module
// edges sharp) and adds a white quiet zone, which many detectors require to
// find the finder patterns. Both are common reasons a tightly-cropped, low-res
// capture fails with an "invalid dimension" error.
func prepareForDecode(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return img
	}

	// Nearest-neighbor upscale so the smaller side is at least ~400px.
	smaller := w
	if h < smaller {
		smaller = h
	}
	scale := 1
	for smaller*scale < 400 {
		scale++
	}

	const quiet = 24 // white border in output pixels
	dstW := w*scale + 2*quiet
	dstH := h*scale + 2*quiet
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)

	for y := 0; y < h*scale; y++ {
		sy := b.Min.Y + y/scale
		for x := 0; x < w*scale; x++ {
			sx := b.Min.X + x/scale
			dst.Set(quiet+x, quiet+y, img.At(sx, sy))
		}
	}
	return dst
}

// renderQRText draws a QR code for the given payload using Unicode half-block
// characters. Each terminal cell carries two vertical modules, so the code
// stays square (no vertical stretch) and every module maps to exactly one
// pixel — no resampling, sharp contrast, and terminal-theme independent.
func renderQRText(w io.Writer, text string) {
	qrterminal.GenerateWithConfig(text, qrterminal.Config{
		Level:      qrterminal.M,
		Writer:     w,
		HalfBlocks: true,
		BlackChar:  qrterminal.BLACK_BLACK,
		WhiteChar:  qrterminal.WHITE_WHITE,
		QuietZone:  2,
	})
}
