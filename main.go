package main

import (
	"flag"
	"fmt"
	"image/color"
	"main/arc"
	"os"
	"strconv"

	"gonum.org/v1/gonum/stat/distuv"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// test efficiency
type Simulator struct {
	cacheType     string
	pageSize      int
	pagesInCache  int
	pagesInMemory int
	readProb      float64
	alpha         float64
	xm            float64
}

func CustomPareto(sim *Simulator, requests int) (int, int, int) {

	// set up memory
	mem := make([]*arc.Page, sim.pagesInMemory)
	pga := arc.PageAllocator{PageSize: sim.pageSize}
	for i := 0; i < sim.pagesInMemory; i++ {
		mem[i] = pga.Allocate()
	}

	// set up cache
	var cache arc.Cache
	if sim.cacheType == "LRU" {
		cache = arc.NewLru(sim.pagesInCache)
	} else if sim.cacheType == "ARC" {
		cache = arc.NewARC(sim.pagesInCache)
	} else {
		return -1, -1, -1
	}
	uniqueReqs := make(map[int]bool)

	// set up pareto
	var par distuv.Pareto
	par.Alpha = sim.alpha
	par.Xm = sim.xm

	// set up coin toss
	var un distuv.Uniform
	un.Max = 1
	un.Min = 0

	for i := 0; i < requests; i++ {
		index := int(float64(sim.pagesInMemory) * par.Rand())

		//fmt.Println(index)

		if index >= sim.pagesInMemory || index < 0 {
			i--
			continue
		}

		pg := mem[index]
		if un.Rand() < sim.readProb {
			uniqueReqs[index] = true
			cache.Get(fmt.Sprintf("%v", index))
		} else {
			cache.Set(fmt.Sprintf("%v", index), pg)
		}
	}

	stats := cache.Stats()
	return stats.Hits, stats.Misses, len(uniqueReqs)
}

func main() {

	// declare graph flag
	plotPtr := flag.Bool("plot", false, "creates a plot of cache stats as a function of the amount of requests\nValues: {true, false}\nDefaultValue = false")
	flag.Parse()
	// CacheSize MemSize Read/Write	Pareto_alpha, Pareto_Xm
	sim := &Simulator{
		cacheType:     "ARC",
		pageSize:      1,
		pagesInCache:  100,
		pagesInMemory: 100000,
		readProb:      0.5,
		alpha:         1.0,
		xm:            0.001,
	}

	var err error
	var args []string

	if len(os.Args) == 1 {
		printHelp()
		return
	}

	if arg1 := os.Args[1]; (len(arg1) >= 6) && (arg1[:6] == "-plot=") {
		args = os.Args[2:]
	} else {
		args = os.Args[1:]
	}

	if len(args) > 7 || len(args) < 1 {
		printHelp()
		return
	}

	sim.cacheType = args[0]
	if sim.cacheType != "ARC" && sim.cacheType != "LRU" {
		fmt.Println("Usage: possible cache_type values are {ARC, LRU}")
		return
	}

	if len(args) > 1 {
		sim.pageSize, err = strconv.Atoi(args[1])
		if err != nil {
			fmt.Printf("Usage: page_size is of type int\n")
			return
		}
	}

	if len(args) > 2 {
		sim.pagesInCache, err = strconv.Atoi(args[2])
		if err != nil {
			fmt.Printf("Usage: pages_in_cache is of type int\n")
			return
		}
	}

	if len(args) > 3 {
		sim.pagesInMemory, err = strconv.Atoi(args[3])
		if err != nil {
			fmt.Printf("Usage: pages_in_memory is of type int\n")
			return
		}
	}

	if len(args) > 4 {
		sim.readProb, err = strconv.ParseFloat(args[4], 64)
		if err != nil {
			fmt.Printf("Usage: read_probability is of type float64\n")
			return
		}
		if sim.readProb < 0.0 || sim.readProb > 1.0 {
			fmt.Printf("Usage: read_probability value is in the interval [0.0, 1.0]\n")
			return
		}
	}

	if len(args) > 5 {
		sim.alpha, err = strconv.ParseFloat(args[5], 64)
		if err != nil {
			fmt.Printf("Usage: pareto_alpha is of type float64\n")
			return
		}
		if sim.alpha == 0.0 {
			fmt.Printf("Usage: pareto_alpha value is non-zero\n")
			return
		}
	}
	if len(args) > 6 {
		sim.xm, err = strconv.ParseFloat(args[6], 64)
		if err != nil {
			fmt.Printf("Usage: pareto_xm is of type float64\n")
			return
		}
		if sim.xm == 0.0 {
			fmt.Printf("Usage: pareto_xm value is non-zero\n")
			return
		}
	}

	// execute
	var requests int
	if !(*plotPtr) {
		requests = 1000 * 1000
		h, m, uniq := CustomPareto(sim, requests)
		fmt.Printf("-------------------------------------------------\n")
		fmt.Printf("%v tracing...\nTotal Requests = %v (Read = %v)\nUniqueReadRequests = %v\nHitRate = %v\n", sim.cacheType, requests, h+m, uniq, 100.0*float32(h)/float32(h+m))
		fmt.Printf("-------------------------------------------------\n")
	} else {
		xy := make(plotter.XYs, 100+1)
		for i := 1; i <= 100; i++ {
			requests = i * 1000
			h, m, _ := CustomPareto(sim, requests)
			xy[i].X = float64(i)
			xy[i].Y = 100.0 * float64(h) / float64(h+m)
		}
		// plotter
		p := plot.New()
		p.Title.Text = fmt.Sprintf("%v: cacheSize = %v, memorySize = %v\npareto_alpha = %v, pareto_xm = %v\n", sim.cacheType, sim.pagesInCache, sim.pagesInMemory,
			sim.alpha, sim.xm)
		p.X.Label.Text = "1000 x Requests"
		p.Y.Label.Text = "HitRate"
		p.Y.Min = 0
		p.Y.Max = 100
		p.Add(plotter.NewGrid())

		line, err := plotter.NewLine(xy)
		line.LineStyle.Width = vg.Points(1)
		line.LineStyle.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		line.LineStyle.Color = color.RGBA{B: 255, A: 255}
		if err != nil {
			panic(err)
		}

		p.Add(line)

		if err = p.Save(5*vg.Inch, 5*vg.Inch, fmt.Sprintf("%v.jpg", sim.cacheType)); err != nil {
			panic(err)
		}

	}

}

func printHelp() {
	fmt.Println("Usage of ./main: <cache_type(LRU or ARC)> -plot optional: <page_size> <pages_in_cache> <pages_in_memory> <read_probability> <pareto_alpha> <pareto_xm>")
	fmt.Printf("Possible values:\ncache_type = {ARC, LRU}\npage_size = int {default = 1}\npages_in_cache = int {default = 100}\npages_in_memory = int {default = 100000}\nread_probability = float64(default = 0.5)\npareto_alpha = float64 {default = 1.0}\npareto_xm = float64 {default = 0.001}\n")
}
