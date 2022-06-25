package main

import (
	"fmt"
	"math/rand"
	"time"

	"./easyvnc"
)

func main() {
	vnc, err := easyvnc.NewEasyVNC(5900, 800, 600)
	if err != nil {
		fmt.Println(err)
		return
	}

	width := vnc.GetWidth()
	height := vnc.GetHeight()

	rand.Seed(time.Now().UnixNano())
	ox1 := 0
	oy1 := 0
	ox2 := 0
	oy2 := 0
	for {
		nx1 := rand.Intn(width)
		ny1 := rand.Intn(height)
		nx2 := rand.Intn(width)
		ny2 := rand.Intn(height)
		split := 20
		for t := 0; t <= split; t++ {
			x1 := (nx1-ox1)*t/split + ox1
			y1 := (ny1-oy1)*t/split + oy1
			x2 := (nx2-ox2)*t/split + ox2
			y2 := (ny2-oy2)*t/split + oy2
			for x := 0; x < width; x++ {
				color := x * 256 / width
				color = (color << 8) | color
				vnc.Line(x, 0, x, height, color)
			}
			vnc.Line(0, 0, 200, 600, 0xff0000)
			vnc.Line(0, 0, 600, 600, 0x00ff00)
			vnc.Line(0, 0, 800, 600, 0x0000ff)
			vnc.Line(x1, y1, x2, y2, 0xffffff)
			vnc.Arc(400, 300, 300, 200, 0xff0000)
			vnc.SendAllFrameData()
		}
		ox1 = nx1
		oy1 = ny1
		ox2 = nx2
		oy2 = ny2
	}
	// ch2 := make(chan int)
	// <-ch2
}
