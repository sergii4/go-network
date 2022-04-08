package tracer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptrace"
)

func WrapHttpClient(client *http.Client) *HttpWrapper {
	return &HttpWrapper{client: client}
}

type HttpWrapper struct {
	client     *http.Client
	started    bool
	statCh     chan stat
	stop       context.CancelFunc
	statistics map[string]counter
}

func (hw *HttpWrapper) Do(req *http.Request) (*http.Response, error) {
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			stat := stat{url: req.URL.String()}
			if info.Reused {
				stat.reused = true
			}
			hw.statCh <- stat
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	return hw.client.Do(req)

}

func (hw *HttpWrapper) StartSatistics() {
	if hw.started {
		return
	}
	hw.statistics = make(map[string]counter)
	hw.statCh = make(chan stat, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	hw.stop = cancel
	go hw.run(ctx)
}

func (hw *HttpWrapper) StopStatistics() {
	hw.stop()
}
func (hw *HttpWrapper) run(ctx context.Context) {
	for {
		select {
		case stat := <-hw.statCh:
			c := hw.statistics[stat.url]
			c.total += 1
			if stat.reused {
				c.reused += 1
			}

			hw.statistics[stat.url] = c
		case <-ctx.Done():
			return
		}
	}

}

func (hw *HttpWrapper) PrintStatistics() {
	for k, v := range hw.statistics {
		fmt.Println("url", k)
		fmt.Printf("total requests: %d\n", v.total)
		fmt.Printf("total reused connections: %d\n", v.reused)
		fmt.Printf("reused connections / total request: %f\n", float32(v.reused)/float32(v.total))
	}

}

type counter struct {
	total  int64
	reused int64
}

type stat struct {
	url    string
	reused bool
}
