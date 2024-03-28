package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mackerelio/go-osstat/cpu"
)

func newWindow[T bool | uint64](per time.Duration) []T {
	return make([]T, 0, int(per.Seconds()))
}

func startMockServer() {
	rn := rand.New(rand.NewPCG(2, 2048))

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {

		randTimeout := rn.IntN(15)
		throwError := rn.IntN(10)

		time.Sleep(time.Duration(randTimeout) * time.Second)
		if throwError > 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

func checkServer(method string, uri string, per time.Duration, status chan bool) {
	tick := time.NewTicker(2 * time.Second)
	req, _ := http.NewRequest(method, uri, nil)

	cli := &http.Client{Timeout: 4 * time.Second}
	for range tick.C {
		resp, err := cli.Do(req)
		if err != nil {
			status <- false
		} else {
			_ = resp.Body.Close()
			fmt.Println(resp.StatusCode)
			if resp.StatusCode >= http.StatusNoContent {
				status <- false
			} else {
				status <- true
			}
		}
	}
}

const (
	timeWindow = 60 * time.Second
)

func httpServerErrCount(window []bool, statusCapt chan bool, errCount *int) {
	windowRefresh := time.NewTicker(timeWindow)

	for {
		select {
		case status := <-statusCapt:
			window = append(window, status)
			if !status {
				*errCount++
			}
			if len(window) > cap(window) {
				oldest := window[0]
				window = window[1:]
				if !oldest {
					*errCount--
				}
			}
			if *errCount > 5 {
				fmt.Println("server down!")
				fmt.Println(window)
				t := 0
				errCount = &t
			}
		case <-windowRefresh.C:
			window = window[:0]
			t := 0
			errCount = &t
		}
	}
}

func cpuProfilerErrCount(errCount *int) {
	tick := time.NewTicker(2 * time.Second)
	var latest, before uint64
	var totalLatest, totalBefore uint64

	for {
		select {
		case <-tick.C:
			stats, err := cpu.Get()
			if err != nil {
				fmt.Println(err)
				continue
			}
			latest, totalLatest = stats.User, stats.Total
			diffTotal := float64(totalLatest - totalBefore)
			diff := float64(latest-before) / diffTotal * 100

			// I don't understand how the statics are taken
			// while i am running this script and monitor the cpu stats with 'top' it doesn't reflect well the
			// cpu stats
			if before != 0 && diff > 20.0 {
				if *errCount > 5 {
					fmt.Println("cpu high!")
					t := 0
					errCount = &t
				}
				*errCount++
			}
			totalBefore = totalLatest
			before = latest
			fmt.Println(diff)
		}
	}
}

func cpuTestUsage() {
	for i := 0; i < 1000000; i++ {
		for j := 0; j < 100000000; j++ {
			for k := 0; k < 1000000000; k++ {
				_ = i*j + k - i + int(math.Pow(2, float64(i)))
			}
		}
	}
}

func main() {
	end := make(chan os.Signal, 1)
	signal.Notify(end, os.Kill, os.Interrupt, syscall.SIGTERM)

	statusCapt := make(chan bool, 10)
	windowHttp := newWindow[bool](timeWindow)

	var errCountHttp int
	var errCountCPUProf int

	go startMockServer()

	go checkServer(http.MethodGet, "http://localhost:8080/test", timeWindow, statusCapt)

	go cpuTestUsage()

	go cpuProfilerErrCount(&errCountCPUProf)

	go httpServerErrCount(windowHttp, statusCapt, &errCountHttp)

	<-end
	fmt.Println("finish")
}
