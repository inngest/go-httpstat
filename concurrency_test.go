package httpstat

import (
	"context"
	"fmt"
	"net/http/httptrace"
	"sync"
	"testing"
	"time"
)

// Cancellation can return control to the caller while transport callbacks
// are still finishing. Exercise the same overlap without network timing.
func TestResultConcurrentTraceAndReaders(t *testing.T) {
	var result Result
	trace := httptrace.ContextClientTrace(WithHTTPStat(context.Background(), &result))
	start := make(chan struct{})
	var workers sync.WaitGroup
	for _, action := range []func(){
		func() {
			trace.ConnectStart("tcp", "127.0.0.1:80")
			trace.ConnectDone("tcp", "127.0.0.1:80", nil)
			trace.WroteRequest(httptrace.WroteRequestInfo{})
			trace.GotFirstResponseByte()
		},
		func() { result.End(time.Now()) },
		func() { _ = result.ContentTransfer(time.Now()) },
		func() { _ = fmt.Sprintf("%+v %s", &result, &result) },
	} {
		workers.Add(1)
		go func(action func()) {
			defer workers.Done()
			<-start
			for range 1000 {
				action()
			}
		}(action)
	}
	close(start)
	workers.Wait()
}
