package auth

import (
	"image"
	"image/color"
	"testing"

	"rsc.io/qr"
)

func TestDecodeQRTextRoundTrip(t *testing.T) {
	payload := "https://open.weixin.qq.com/connect/confirm?uuid=abc123XYZ"
	code, err := qr.Encode(payload, qr.M)
	if err != nil {
		t.Fatal(err)
	}
	scale := 6
	sz := code.Size
	img := image.NewGray(image.Rect(0, 0, (sz+8)*scale, (sz+8)*scale))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	for y := 0; y < sz; y++ {
		for x := 0; x < sz; x++ {
			if code.Black(x, y) {
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.SetGray((x+4)*scale+dx, (y+4)*scale+dy, color.Gray{Y: 0})
					}
				}
			}
		}
	}
	got, err := decodeQRText(img)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if got != payload {
		t.Fatalf("payload mismatch: got %q want %q", got, payload)
	}
}
