package httpstat

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http/httptrace"
	"testing"
	"time"
)

func TestSnapshotRequestProgress(t *testing.T) {
	var result Result
	trace := httptrace.ContextClientTrace(WithHTTPStatProgress(context.Background(), &result))
	trace.ConnectStart("tcp", "192.0.2.1:443")
	trace.ConnectDone("tcp", "192.0.2.1:443", nil)
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, errors.New("TLS failed"))
	if got := result.Snapshot(); !got.TLSStarted || got.TLSCompleted || got.RequestWritten || got.FirstByte {
		t.Fatalf("unexpected snapshot after failed handshake: %+v", got)
	}

	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, nil)
	trace.WroteRequest(httptrace.WroteRequestInfo{Err: errors.New("write failed")})
	if got := result.Snapshot(); !got.TLSCompleted || got.RequestWritten || got.FirstByte {
		t.Fatalf("unexpected snapshot after failed write: %+v", got)
	}

	trace.WroteRequest(httptrace.WroteRequestInfo{})
	if got := result.Snapshot(); !got.RequestWritten || got.FirstByte {
		t.Fatalf("unexpected snapshot before response: %+v", got)
	}
	trace.GotFirstResponseByte()
	if got := result.Snapshot(); !got.FirstByte {
		t.Fatalf("unexpected snapshot after first response byte: %+v", got)
	}

	trace.GetConn("redirect.example:443")
	trace.TLSHandshakeStart()
	trace.TLSHandshakeDone(tls.ConnectionState{}, errors.New("reset after redirect"))
	if got := result.Snapshot(); !got.TLSStarted || got.TLSCompleted || got.RequestWritten || got.FirstByte {
		t.Fatalf("unexpected snapshot after redirect: %+v", got)
	}
}

func TestSnapshotFirstByteAfterRedirect(t *testing.T) {
	var result Result
	trace := httptrace.ContextClientTrace(WithHTTPStatProgress(context.Background(), &result))

	trace.GetConn("first.example:443")
	trace.DNSStart(httptrace.DNSStartInfo{})
	trace.DNSDone(httptrace.DNSDoneInfo{})
	// Simulate a slow first hop. The redirected request then uses cached DNS.
	result.dnsStart = time.Now().Add(-time.Hour)
	trace.GetConn("second.example:443")
	start := result.progressStart
	trace.ConnectStart("tcp", "192.0.2.2:443")
	trace.ConnectDone("tcp", "192.0.2.2:443", nil)
	trace.GotFirstResponseByte()

	if got, want := result.Snapshot().TimeToFirstByte, result.serverDone.Sub(start); got != want {
		t.Fatalf("first-byte timing includes a previous hop: got %v, want %v", got, want)
	}
}
