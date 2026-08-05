// Package main ☢.go
package main

import (
	"encoding/binary"

	"crypto/rand"
	"math"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/paint"
)

const (
	numLatLines = 12
	numLonLines = 24
	radius      = 0.4
	centerX     = 0.5
	centerY     = 0.5
)

func randFloat64InRange(n, x float64) float64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("failed to read crypto/rand: " + err.Error())
	}
	u := float64(binary.LittleEndian.Uint64(b[:])) / (1 << 63)
	u = math.Min(u, 1)
	return n + u*(x-n)
}

var (
	rotationX, rotationY, rotationZ = randFloat64InRange(0, 2*math.Pi), randFloat64InRange(0, 2*math.Pi), randFloat64InRange(0, 2*math.Pi)
	dX, dY, dZ                      = randFloat64InRange(-0.02, 0.02), randFloat64InRange(-0.02, 0.02), randFloat64InRange(-0.02, 0.02)
	pause                           = true
)

func globeAnimation(w *core.Canvas) {
	w.SetDraw(func(pc *paint.Painter) {
		//pc.Fill.Color = colors.Uniform(colors.Transparent)
		//pc.Clear()
		sinX, cosX := math.Sin(rotationX), math.Cos(rotationX)
		sinY, cosY := math.Sin(rotationY), math.Cos(rotationY)
		sinZ, cosZ := math.Sin(rotationZ), math.Cos(rotationZ)

		projectPoint := func(x, y, z float64) (float32, float32) {
			ry := y*cosX - z*sinX
			rz := y*sinX + z*cosX

			rx := x*cosY + rz*sinY
			//			z = -x*sinY + rz*cosY

			x = rx*cosZ - ry*sinZ
			y = rx*sinZ + ry*cosZ

			screenX := centerX + float32(x)
			screenY := centerY + float32(y)
			return screenX, screenY
		}

		pc.Stroke.Color = colors.Scheme.Primary.Base
		for lat := -math.Pi / 2; lat <= math.Pi/2; lat += math.Pi / float64(numLatLines) {
			first := true
			var startX, startY float32
			for lon := 0.0; lon <= 2*math.Pi; lon += math.Pi / 100 {
				x := radius * math.Cos(lat) * math.Cos(lon)
				y := radius * math.Cos(lat) * math.Sin(lon)
				z := radius * math.Sin(lat)
				screenX, screenY := projectPoint(x, y, z)
				if first {
					startX, startY = screenX, screenY
					first = false
				} else {
					pc.MoveTo(startX, startY)
					pc.LineTo(screenX, screenY)
					startX, startY = screenX, screenY
				}
			}
			pc.Draw()
		}

		pc.Stroke.Color = colors.Scheme.Primary.Base
		for lon := 0.0; lon <= 2*math.Pi; lon += math.Pi / float64(numLonLines) {
			first := true
			var startX, startY float32
			for lat := -math.Pi / 2; lat <= math.Pi/2; lat += math.Pi / 100 {
				x := radius * math.Cos(lat) * math.Cos(lon)
				y := radius * math.Cos(lat) * math.Sin(lon)
				z := radius * math.Sin(lat)
				screenX, screenY := projectPoint(x, y, z)
				if first {
					startX, startY = screenX, screenY
					first = false
				} else {
					pc.MoveTo(startX, startY)
					pc.LineTo(screenX, screenY)
					startX, startY = screenX, screenY
				}
			}
			pc.Draw()
		}
	})

	w.Animate(func(_ *core.Animation) {
		if pause {
			return
		}
		mySvg.Update()
		rotationX += dX
		rotationY += dY
		rotationZ += dZ
		w.NeedsRender()
	})
}
