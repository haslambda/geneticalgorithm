package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Gene struct {
	Center [2]int
	Radius int
	Color  color.RGBA
}

type Result struct {
	fitness float32
	genome  []Gene
	out     image.Image
}

var (
	initialGenes int
	population   int
	mutationProb float32
	addProb      float32
	removeProb   float32

	minRadius int
	maxRadius int

	saveIter int

	width  int
	height int

	best Result

	img image.Image

	wg sync.WaitGroup

	genome []Gene
	out image.Image
)

func (g Gene) init() Gene {
	g.Radius = rand.Intn(maxRadius-minRadius) + minRadius
	g.Center = [2]int{rand.Intn(width), rand.Intn(height)}
	g.Color.R, g.Color.G, g.Color.B, g.Color.A = uint8(rand.Intn(256)), uint8(rand.Intn(256)), uint8(rand.Intn(256)), 255
	return g
}

func (g Gene) MutateRadius(mutationSize float32) int {
	max := float32(g.Radius) * (1 + mutationSize)
	min := float32(g.Radius) * (1 - mutationSize)
	if int(max-min) == 0 {
		g.Radius = int(min)
	} else {
		g.Radius = rand.Intn(int(max-min)) + int(min)
	}
	g.Radius = Clip(g.Radius, 1, 100)
	return g.Radius
}

func (g Gene) MutateCenter(mutationSize float32) [2]int {
	min0, max0 := float32(g.Center[0])*(1-mutationSize), float32(g.Center[0])*(1+mutationSize)
	min1, max1 := float32(g.Center[1])*(1-mutationSize), float32(g.Center[1])*(1+mutationSize)

	if int(max0-min0) == 0 {
		g.Center[0] = int(min0)
	} else {
		g.Center[0] = rand.Intn(int(max0-min0)) + int(min0)
	}

	if int(max1-min1) == 0 {
		g.Center[1] = int(min1)
	} else {
		g.Center[1] = rand.Intn(int(max1-min1)) + int(min1)
	}

	g.Center = [2]int{
		Clip(g.Center[0], 0, width),
		Clip(g.Center[1], 0, height)}

	return g.Center
}

func (g Gene) MutateColor(mutationSize float32) color.RGBA {
	cR := int(g.Color.R)
	cG := int(g.Color.G)
	cB := int(g.Color.B)
	minR, maxR := float32(cR)*(1-mutationSize), float32(cR)*(1+mutationSize)
	minG, maxG := float32(cG)*(1-mutationSize), float32(cG)*(1+mutationSize)
	minB, maxB := float32(cB)*(1-mutationSize), float32(cB)*(1+mutationSize)

	if int(maxR-minR) == 0 {
		g.Color.R = uint8(minR)
	} else {
		g.Color.R = uint8(rand.Intn(int(maxR-minR)) + int(minR))
	}

	if int(maxG-minG) == 0 {
		g.Color.G = uint8(minG)
	} else {
		g.Color.G = uint8(rand.Intn(int(maxG-minG)) + int(minG))
	}

	if int(maxB-minB) == 0 {
		g.Color.B = uint8(minB)
	} else {
		g.Color.B = uint8(rand.Intn(int(maxB-minB)) + int(minB))
	}

	g.Color.R = uint8(Clip(int(g.Color.R), 0, 255))
	g.Color.G = uint8(Clip(int(g.Color.G), 0, 255))
	g.Color.B = uint8(Clip(int(g.Color.B), 0, 255))

	return g.Color
}

func (g Gene) Mutate() Gene {
	mutationSize := float32(math.Max(1, math.Round(rand.NormFloat64()*float64(4)+float64(15)))) / float32(100)

	r := rand.Float64()

	if r < 0.33 {
		g.Radius = g.MutateRadius(mutationSize)
	} else if r < 0.66 {
		g.Center = g.MutateCenter(mutationSize)
	} else {
		g.Color = g.MutateColor(mutationSize)
	}
	return g
}

func ComputeFitness(genome []Gene) (float32, image.Image) {
	var fitness float32
	out := image.NewRGBA(image.Rect(0, 0, width, height))

	UnTransparent(out)

	for _, gene := range genome {
		DrawCircle(out, gene.Center[0], gene.Center[1], gene.Radius, gene.Color)
	}

	if img, ok := img.(*image.RGBA); ok {
		compVal, _ := CompareImage(img, out)
		fitness = float32(255) / float32(compVal)
	}
	return fitness, out
}

func ComputePopulation(gnm []Gene) Result {
	gene := Gene{}
	gene = gene.init()

	//genome := gnm

	genome2 := make([]Gene, len(gnm))

	for i, gn := range gnm {
		r := gn.Color.R
		g := gn.Color.G
		b := gn.Color.B

		cx := gn.Center[0]
		cy := gn.Center[1]
		rad := gn.Radius

		genome2[i].Color = color.RGBA{r, g, b, 255}
		genome2[i].Center = [2]int{cx, cy}
		genome2[i].Radius = rad
	}

	if len(genome2) < 200 {
		for _, g := range genome2 {
			if rand.Float32() < mutationProb {
				g = g.Mutate()
			}
		}
	} else {
		mut := RandomSampleGene(genome2, int(float32(len(genome2))*mutationProb))
		for _, g := range mut {
			g = g.Mutate()
		}
	}

	if rand.Float32() < addProb {
		genome2 = append(genome2, gene)
	}

	if len(genome2) > 0 && rand.Float32() < removeProb {
		genome2 = RemoveGene(genome2, rand.Intn(len(genome2)))
	}

	fitness, out2 := ComputeFitness(genome2)

	result := Result{fitness, genome2, out2}
	return result
}

func worker(results chan Result) {
	defer wg.Done()
	for i := 0; i < 10; i++ {
		result := ComputePopulation(genome)
		results <- result
	}
}

func findBest(results chan Result) {
	best = <-results
	max := best.fitness
	for i := 0; i < population-1; i++ {
		result := <-results
		if result.fitness > max {
			max = result.fitness
			best = result
		}
	}
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	initialGenes = 50
	population = 50
	mutationProb = 0.01
	addProb = 0.3
	removeProb = 0.2

	minRadius = 5
	maxRadius = 15

	//imgPath := os.Args[1]
	imgPath := "test.png"
	imgF, img_check := os.Open(imgPath)
  if img_check != nil {
    fmt.Println(img_check)
    return
  }
	defer imgF.Close()
	img, _, _ = image.Decode(imgF)
	b := img.Bounds()

	width = b.Max.X
	height = b.Max.Y
	gen := 0
	workers := 5

	os.Mkdir("result", 777)

	genome = make([]Gene, initialGenes)

	for i, gene := range genome {
		genome[i] = gene.init()
	}

	_, out = ComputeFitness(genome)

	for gen < 2000 {

		//genomes := make(chan []Gene, population)
		results := make(chan Result, population)

		//for i := 0; i < population; i++ {
		//	genomes <- genome
		//}

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go worker(results)
		}

		wg.Wait()

		//for r := 0; r < population; r++ {
		//	<-results
		//} )

		findBest(results)

		genome = best.genome
		out = best.out

		close(results)

		gen++
		fmt.Printf("Currently on generation %d, fitness %.6f\n", gen, best.fitness)
	}
	output, _ := os.Create("testsave.png")
	err := png.Encode(output, out)

	if err != nil {
		fmt.Println(err)
	}
}
