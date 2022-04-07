package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/sync/errgroup"
)

const (
	method = http.MethodPost
	url    = "http://localhost:8080/"
)

var client *HttpWrapper

func main() {

	fs := flag.NewFlagSet("repeat", flag.ExitOnError)
	rps := fs.Int("rps", 100, "request per seconds")
	total := fs.Int("total", 100, "total requests")
	intervalMs := fs.Int("int", 0, "interval between request, milliseconds")

	f, err := os.Create(time.Now().String())
	if err != nil {
		log.Fatal("can't open file")
	}
	defer f.Close()
	log.SetOutput(f)
	root := &ffcli.Command{
		Name:       "",
		ShortUsage: "repeat [-n times] <arg>",
		ShortHelp:  "Repeatedly print the argument to stdout.",
		FlagSet:    fs,
		Exec: func(_ context.Context, _ []string) error {
			client = wrapHttpClient(&http.Client{
				Timeout:   10 * time.Minute,
			})
			intMs := *intervalMs * int(time.Millisecond)
			err := postWorker(*rps, *total, time.Duration(intMs))
			if err != nil {
				fmt.Println(err)
			}
			return err
		},
	}
	// Go signal notification works by sending `os.Signal`
	// values on a channel. We'll create a channel to
	// receive these notifications. Note that this channel
	// should be buffered.
	sigs := make(chan os.Signal, 1)

	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// We could receive from `sigs` here in the main
	// function, but let's see how this could also be
	// done in a separate goroutine, to demonstrate
	// a more realistic scenario of graceful shutdown.
	done := make(chan bool, 1)

	go func() {
		// This goroutine executes a blocking receive for
		// signals. When it gets one it'll print it out
		// and then notify the program that it can finish.
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	go func() {
		if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
			log.Println(err)
		}
		done <- true
	}()
	// The program will wait here until it gets the
	// expected signal (as indicated by the goroutine
	// above sending a value on `done`) and then exit.
	fmt.Println("awaiting signal")
	<-done
	for k, v := range client.statistics {
		fmt.Println("url", k)
		fmt.Printf("total requests: %d\n", *v.total)
		fmt.Printf("total reused connections: %d\n", *v.reused)
		fmt.Printf("reused connections / total request: %f\n", float32(*v.reused)/float32(*v.total))
	}
	fmt.Println("exiting")

}

func postWorker(rps, total int, interval time.Duration) error {
	for i := 0; i < total; i = i + rps {
		g := new(errgroup.Group)
		start := time.Now()
		for j := 0; j < rps; j++ {
			g.Go(post)
		}
		if err := g.Wait(); err != nil {
			return err
		}
		fmt.Println(float64(rps) / time.Since(start).Seconds())
		time.Sleep(interval)
	}
	return nil
}

func post() error {
	//req, err := http.NewRequest(method, url, nil)
	req, err := http.NewRequest(method, url, http.NoBody)

	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}


func wrapHttpClient(client *http.Client) *HttpWrapper {
	return &HttpWrapper{client: client, statistics: make(map[string]struct {
		total  *uint64
		reused *uint64
	})}
}

type HttpWrapper struct {
	client     *http.Client
	statistics map[string]struct {
		total  *uint64
		reused *uint64
	}
}

func (hw *HttpWrapper) Do(req *http.Request) (*http.Response, error) {

	//	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			//			fmt.Printf("time to first response byte is %d, for url: %s \n", time.Since(start).Milliseconds(), req.URL)
		},
		GotConn: func(info httptrace.GotConnInfo) {
			if _, ok := hw.statistics[req.URL.String()]; !ok {
				zeroT := uint64(0)
				zeroR := uint64(0)
				hw.statistics[req.URL.String()] = struct {
					total  *uint64
					reused *uint64
				}{
					total:  &zeroT,
					reused: &zeroR,
				}
			}
			atomic.AddUint64(hw.statistics[req.URL.String()].total, 1)
			if info.Reused {
				atomic.AddUint64(hw.statistics[req.URL.String()].reused, 1)
			}
			//			fmt.Printf("Connection reused for %v? %v\n", req.URL, info.Reused)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	return hw.client.Do(req)
}
