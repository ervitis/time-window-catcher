package main

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func newWindow(per time.Duration) []bool {
	return make([]bool, 0, int(per.Seconds()))
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

func main() {
	end := make(chan os.Signal, 1)
	signal.Notify(end, os.Kill, os.Interrupt, syscall.SIGTERM)

	statusCapt := make(chan bool, 10)
	window := newWindow(timeWindow)
	windowRefresh := time.NewTicker(timeWindow)
	var errCount int

	go startMockServer()

	go checkServer(http.MethodGet, "http://localhost:8080/test", timeWindow, statusCapt)

	go func(errCount *int) {
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
	}(&errCount)

	<-end
	fmt.Println("finish")
}
