package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"sync"
)

func main() {
	tileSize := 100
	var xSize, ySize int = 1e3, 1e3

	// create final image container
	img := image.NewRGBA(image.Rect(0, 0, xSize, ySize))

	// create channels
	in := make(chan work, 1e3)
	out := make(chan *image.RGBA, 1e3)

	// start workers
	wg, wgOut := &sync.WaitGroup{}, &sync.WaitGroup{}
	for i := 0; i < img.Bounds().Dx()/tileSize; i++ {
		wg.Add(1)

		// launch worker
		go worker(wg, in, out)
	}

	// launch "reduce" clojure
	wgOut.Add(1)
	go func() {
		defer wgOut.Done() // signal we are done on exit

		// until tiles arrive, add them to the final image
		for tile := range out {
			draw.Draw(img, tile.Bounds(), tile, tile.Bounds().Min, draw.Src)
		}
	}()

	// send work
	for x := 0; x < xSize; x += tileSize {
		for y := 0; y < ySize; y += tileSize {
			in <- work{
				x: x, y: y, dx: tileSize, dy: tileSize,
				// colorFunc: func(x, y int) color.RGBA {
				// 	return color.RGBA{uint8(x * 255 / img.Bounds().Dx()), uint8(y * 255 / img.Bounds().Dy()), 100, 255}
				// },
				colorFunc: func(x, y int) color.RGBA {
					return color.RGBA{uint8(x * 255 / tileSize), uint8(y * 255 / tileSize), 100, 255}
				},
			}
		}
	}

	// close channel and wait for the goroutines to complete
	close(in)
	wg.Wait()
	close(out)
	wgOut.Wait()

	// save final image
	err := save(img)
	if err != nil {
		log.Fatal(err)
	}
}

type work struct {
	x, y, dx, dy int
	colorFunc    func(x, y int) color.RGBA
}

func worker(wg *sync.WaitGroup, in chan work, out chan *image.RGBA) {
	defer wg.Done()     // signal the goroutine is over
	for w := range in { // until we have work to do
		tile := image.NewRGBA(image.Rect(w.x, w.y, w.x+w.dx, w.y+w.dy)) // create empty tile

		// loop over the tile pixels
		for xx := w.x; xx < w.x+w.dx; xx++ {
			for yy := w.y; yy < w.y+w.dy; yy++ {

				// set pixel color
				tile.SetRGBA(xx, yy, w.colorFunc(xx, yy))
			}
		}
		out <- tile
	}
}

func save(img image.Image) error {
	f, err := os.Create("example.png")
	if err != nil {
		return err
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		return err
	}
	return nil
}
