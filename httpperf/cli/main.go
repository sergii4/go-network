package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tracer"

	"github.com/peterbourgon/ff/v3/ffcli"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
)

const (
	method = http.MethodPost
	url    = "%s://localhost:8080/"
)

var urlWithSchema string

var client *tracer.HttpWrapper

func main() {

	fs := flag.NewFlagSet("repeat", flag.ExitOnError)
	rps := fs.Int("rps", 100, "request per seconds")
	total := fs.Int("total", 100, "total requests")
	intervalMS := fs.Int("int", 0, "interval between request, milliseconds")
	http2 := fs.Bool("http2", false, "use http2")
	https := fs.Bool("https", false, "use https")

	maxIdleConns := fs.Int("mic", 100, "max idle conns")
	maxIdleConnsPerHost := fs.Int("micph", 2, "max idle conns per host")

	root := &ffcli.Command{
		Name:       "",
		ShortUsage: "repeat [-n times] <arg>",
		ShortHelp:  "Repeatedly print the argument to stdout.",
		FlagSet:    fs,
		Exec: func(_ context.Context, _ []string) error {
			urlWithSchema = fmt.Sprintf(url, "http")
			config := &tls.Config{InsecureSkipVerify: true}
			if *https {
				urlWithSchema = fmt.Sprintf(url, "https")
//				cer, err := tls.LoadX509KeyPair("../server.crt", "../server.key")
//				if err != nil {
//					return err
//				}
//				config = &tls.Config{Certificates: []tls.Certificate{cer}}

			}
			client = tracer.WrapHttpClient(&http.Client{
				Timeout:   10 * time.Minute,
				Transport: GetHttpTransport(*http2, *maxIdleConns, *maxIdleConnsPerHost, config),
			})
			client.StartSatistics()
			intMs := *intervalMS * int(time.Millisecond)

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
	req, err := http.NewRequest(method, urlWithSchema, http.NoBody)

	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	//	fmt.Println("body:", string(body))

	return nil
}

func GetHttpTransport(useHTTP2 bool, maxIdleConns, maxIdleConnsPerHost int, config *tls.Config) http.RoundTripper {
	if useHTTP2 {
		// workaround to get the golang standard HTTP/2 client to connect to an H2C enabled server.
		return &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
	}
	// To configure the number of connections in the pool, we must override http.Transport.MaxIdleConns.
	// This value is set by default to 100. Yet, there’s something important to note: the existence of a limit per host with http.Transport.MaxIdleConnsPerHost, which is set by default to 2.
	// So, for example, if we trigger 100 requests to the same host, only 2 connections will remain in the connection pool after that.
	// Hence, if we trigger 100 requests again, we will have to reopen at least 98 connections. This is also an important configuration to note as it can impact the average latency if we have to deal with a significant number of parallel requests to the same host.
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   0,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       config,
	}

}
