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

const (
	tileSize     = 100
	imgSize  int = 1e3
)

func main() {
	// create final image container
	img := image.NewRGBA(image.Rect(0, 0, imgSize, imgSize))

	// create channels and waitgroups
	in := make(chan work, 1e3)
	out := make(chan *image.RGBA, 1e3)

	wkrs, rdc := &sync.WaitGroup{}, &sync.WaitGroup{}

	// start workers
	for i := 0; i < img.Bounds().Dx()/tileSize; i++ {
		wkrs.Add(1)

		go worker(wkrs, in, out)
	}

	// launch "reduce" closure
	rdc.Add(1)
	go func() {
		defer rdc.Done() // signal we are done on exit

		// until tiles arrive, add them to the final image
		for tile := range out {
			draw.Draw(img, tile.Bounds(), tile, tile.Bounds().Min, draw.Src)
		}
	}()

	// send work:
	// it would be not efficient to create a worker per pixel,
	// much better use a pool of workers, each for a tile containing
	// a certain number of pixels
	for x := 0; x < imgSize; x += tileSize {
		for y := 0; y < imgSize; y += tileSize {

			in <- work{
				x: x, y: y, dx: tileSize, dy: tileSize,
				// here you can change the type of resulting image by choosing
				// how to color each tile
				colorFunc: newCF(imgSize), // single image
				// colorFunc: newCF(tileSize), // identical tiles
			}
		}
	}

	// close channel and wait for the goroutines to complete
	close(in)
	wkrs.Wait()

	close(out)
	rdc.Wait()

	// save final image
	err := save(img)
	if err != nil {
		log.Fatal(err)
	}
}

type cf = func(x, y int) color.RGBA

func newCF(sideSize int) cf {
	return func(x, y int) color.RGBA {
		return color.RGBA{uint8(x * 255 / sideSize), uint8(y * 255 / sideSize), 100, 255}
	}
}

type work struct {
	x, y, dx, dy int
	colorFunc    func(x, y int) color.RGBA
}

func worker(wkrs *sync.WaitGroup, in chan work, out chan *image.RGBA) {
	defer wkrs.Done() // signal the goroutine is over

	for w := range in { // until we have work to do
		tile := image.NewRGBA(image.Rect(w.x, w.y, w.x+w.dx, w.y+w.dy)) // create empty tile

		// loop over the tile pixels
		for xx := w.x; xx < w.x+w.dx; xx++ {
			for yy := w.y; yy < w.y+w.dy; yy++ {

				// set pixel color using the choosen color function
				tile.SetRGBA(xx, yy, w.colorFunc(xx, yy))
			}
		}

		// send the result to the reducer
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
