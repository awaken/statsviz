package plot

import (
	"encoding/json"
	"fmt"
	"image/color"
)

// RGBString returns a CSS/Plotly-valid color string.
func RGBString(r, g, b uint8) string {
	return fmt.Sprintf("rgb(%d,%d,%d)", r, g, b)
}

type WeightedColor struct {
	Value float64
	Color color.RGBA
}

func (c WeightedColor) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]any{c.Value, RGBString(c.Color.R, c.Color.G, c.Color.B)})
}

// NOTE: shades obtained from https://mdigi.tools/color-shades/

var BlueShades = []WeightedColor{
	{Value: 0.0, Color: color.RGBA{0xea, 0xf8, 0xfd, 0xff}}, // rgb(234, 248, 253)
	{Value: 0.1, Color: color.RGBA{0xbf, 0xeb, 0xfa, 0xff}}, // rgb(191, 235, 250)
	{Value: 0.2, Color: color.RGBA{0x94, 0xdd, 0xf6, 0xff}}, // rgb(148, 221, 246)
	{Value: 0.3, Color: color.RGBA{0x69, 0xd0, 0xf2, 0xff}}, // rgb(105, 208, 242)
	{Value: 0.4, Color: color.RGBA{0x3f, 0xc2, 0xef, 0xff}}, // rgb(63, 194, 239)
	{Value: 0.5, Color: color.RGBA{0x14, 0xb5, 0xeb, 0xff}}, // rgb(20, 181, 235)
	{Value: 0.6, Color: color.RGBA{0x10, 0x94, 0xc0, 0xff}}, // rgb(16, 148, 192)
	{Value: 0.7, Color: color.RGBA{0x0d, 0x73, 0x96, 0xff}}, // rgb(13, 115, 150)
	{Value: 0.8, Color: color.RGBA{0x09, 0x52, 0x6b, 0xff}}, // rgb(9, 82, 107)
	{Value: 0.9, Color: color.RGBA{0x05, 0x31, 0x40, 0xff}}, // rgb(5, 49, 64)
	{Value: 1.0, Color: color.RGBA{0x02, 0x10, 0x15, 0xff}}, // rgb(2, 16, 21)
}

var PinkShades = []WeightedColor{
	{Value: 0.0, Color: color.RGBA{0xfe, 0xe7, 0xf3, 0xff}}, // rgb(254, 231, 243)
	{Value: 0.1, Color: color.RGBA{0xfc, 0xb6, 0xdc, 0xff}}, // rgb(252, 182, 220)
	{Value: 0.2, Color: color.RGBA{0xf9, 0x85, 0xc5, 0xff}}, // rgb(249, 133, 197)
	{Value: 0.3, Color: color.RGBA{0xf7, 0x55, 0xae, 0xff}}, // rgb(247, 85, 174)
	{Value: 0.4, Color: color.RGBA{0xf5, 0x24, 0x96, 0xff}}, // rgb(245, 36, 150)
	{Value: 0.5, Color: color.RGBA{0xdb, 0x0a, 0x7d, 0xff}}, // rgb(219, 10, 125)
	{Value: 0.6, Color: color.RGBA{0xaa, 0x08, 0x61, 0xff}}, // rgb(170, 8, 97)
	{Value: 0.7, Color: color.RGBA{0x7a, 0x06, 0x45, 0xff}}, // rgb(122, 6, 69)
	{Value: 0.8, Color: color.RGBA{0x49, 0x03, 0x2a, 0xff}}, // rgb(73, 3, 42)
	{Value: 0.9, Color: color.RGBA{0x18, 0x01, 0x0e, 0xff}}, // rgb(24, 1, 14)
	{Value: 1.0, Color: color.RGBA{0x00, 0x00, 0x00, 0xff}}, // rgb(0, 0, 0)
}

var GreenShades = []WeightedColor{
	{Value: 0.0, Color: color.RGBA{0xed, 0xf7, 0xf2, 0xff}}, // rgb(237, 247, 242)
	{Value: 0.1, Color: color.RGBA{0xc9, 0xe8, 0xd7, 0xff}}, // rgb(201, 232, 215)
	{Value: 0.2, Color: color.RGBA{0xa5, 0xd9, 0xbc, 0xff}}, // rgb(165, 217, 188)
	{Value: 0.3, Color: color.RGBA{0x81, 0xca, 0xa2, 0xff}}, // rgb(129, 202, 162)
	{Value: 0.4, Color: color.RGBA{0x5e, 0xbb, 0x87, 0xff}}, // rgb(94, 187, 135)
	{Value: 0.5, Color: color.RGBA{0x44, 0xa1, 0x6e, 0xff}}, // rgb(68, 161, 110)
	{Value: 0.6, Color: color.RGBA{0x35, 0x7e, 0x55, 0xff}}, // rgb(53, 126, 85)
	{Value: 0.7, Color: color.RGBA{0x26, 0x5a, 0x3d, 0xff}}, // rgb(38, 90, 61)
	{Value: 0.8, Color: color.RGBA{0x17, 0x36, 0x25, 0xff}}, // rgb(23, 54, 37)
	{Value: 0.9, Color: color.RGBA{0x08, 0x12, 0x0c, 0xff}}, // rgb(8, 18, 12)
	{Value: 1.0, Color: color.RGBA{0x00, 0x00, 0x00, 0xff}}, // rgb(0, 0, 0)
}
