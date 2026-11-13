package httpstat

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http/httptrace"
	"testing"
)

func TestSnapshotRequestProgress(t *testing.T) {
	var result Result
	trace := httptrace.ContextClientTrace(WithHTTPStat(context.Background(), &result))
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
}
