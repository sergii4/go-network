package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tracer"

	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/sync/errgroup"
)

const (
	method = http.MethodPost
	url    = "http://localhost:8080/"
)

var client *tracer.HttpWrapper

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
			client = tracer.WrapHttpClient(&http.Client{
				Timeout: 10 * time.Minute,
			})
			client.StartSatistics()
			intMs := *intervalMs * int(time.Millisecond)
			err := postWorker(*rps, *total, time.Duration(intMs))
			if err != nil {
				fmt.Println(err)
			}
			return err
		},
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool, 1)

	go func() {
		<-sigs
		done <- true
	}()
	go func() {
		if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
			log.Println(err)
		}
		done <- true
	}()
	<-done
	client.StopStatistics()
	client.PrintStatistics()

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
