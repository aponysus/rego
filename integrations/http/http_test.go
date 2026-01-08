package http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	integration "github.com/aponysus/recourse/integrations/http"
	"github.com/aponysus/recourse/policy"
	"github.com/aponysus/recourse/retry"
)

func TestDoHTTP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello")
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor()
	client := server.Client()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, _, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != "Hello" {
		t.Errorf("got body %q, want Hello", body)
	}
}

func TestDoHTTP_RetryOn503(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "Success")
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor()
	client := server.Client()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, tl, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if len(tl.Attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", len(tl.Attempts))
	}
	if resp.StatusCode != 200 {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
}

func TestDoHTTP_RecordsRetryAfterOverride(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "1") // 1 second
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor(
		retry.WithPolicy("test", policy.MaxAttempts(1)),
	)
	client := server.Client()

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, tl, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err == nil {
		t.Fatalf("expected error")
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d, want 1", attempts)
	}
	if len(tl.Attempts) != 1 {
		t.Fatalf("attempts=%d, want 1", len(tl.Attempts))
	}
	out := tl.Attempts[0].Outcome
	if out.BackoffOverride != time.Second {
		t.Fatalf("backoffOverride=%v, want 1s", out.BackoffOverride)
	}
	if got := out.Attributes["retry_after"]; got != "1s" {
		t.Fatalf("retry_after=%q, want %q", got, "1s")
	}
}

func TestDoHTTP_ReplaysBodyWithGetBody(t *testing.T) {
	var bodies [][]byte
	var calls int

	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(req.Body)
		bodies = append(bodies, data)
		calls++
		if calls == 1 {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader("retry")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	client := &http.Client{Transport: rt}
	exec := retry.NewDefaultExecutor(
		retry.WithPolicy("test",
			policy.MaxAttempts(2),
			policy.InitialBackoff(0),
			policy.MaxBackoff(0),
			policy.Jitter(policy.JitterNone),
		),
	)

	body := []byte("payload")
	req, _ := http.NewRequest("PUT", "http://example.test", io.NopCloser(bytes.NewReader(body)))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	resp, tl, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
	if len(tl.Attempts) != 2 {
		t.Fatalf("attempts=%d, want 2", len(tl.Attempts))
	}
	if calls != 2 {
		t.Fatalf("calls=%d, want 2", calls)
	}
	if len(bodies) != 2 || !bytes.Equal(bodies[0], body) || !bytes.Equal(bodies[1], body) {
		t.Fatalf("bodies=%q, want two payload copies", bodies)
	}
}

func TestDoHTTP_ContextCanceled(t *testing.T) {
	called := false
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("unexpected request")
	})

	exec := retry.NewDefaultExecutor()
	client := &http.Client{Transport: rt}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, _ := http.NewRequest("GET", "http://example.test", nil)
	_, _, err := integration.DoHTTP(ctx, exec, policy.PolicyKey{Name: "test"}, client, req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v, want context canceled", err)
	}
	if called {
		t.Fatal("unexpected request execution")
	}

}

func TestDoHTTP_DrainsAndClosesBody(t *testing.T) {
	body := &trackingBody{data: bytes.Repeat([]byte("x"), 5000)}

	rt := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       body,
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	exec := retry.NewDefaultExecutor(retry.WithPolicy("test", policy.MaxAttempts(1)))
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest("GET", "http://example.test", nil)
	_, _, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !body.closed {
		t.Fatalf("expected body to be closed")
	}
	if body.read != 4096 {
		t.Fatalf("read=%d, want 4096", body.read)
	}
}

func TestDoHTTP_NonRetryableStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor()
	client := server.Client()

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, tl, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)

	// DoHTTP returns a StatusError for non-2xx responses when they are classified as failures
	// by the AutoClassifier or default rules. Verify proper error wrapping.

	if err == nil {
		t.Fatal("expected error for 404")
	}

	st, ok := err.(*integration.StatusError)
	if !ok || st.Code != 404 {
		t.Errorf("expected 404 StatusError, got %v", err)
	}

	if len(tl.Attempts) != 1 {
		t.Errorf("expected 1 attempt for non-retryable error, got %d", len(tl.Attempts))
	}
}

func TestDoHTTP_NonReplayableBodyReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor()
	client := server.Client()

	body := &io.LimitedReader{R: strings.NewReader("body"), N: 4}
	req, _ := http.NewRequest("POST", server.URL, body)
	req.GetBody = nil

	_, _, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err == nil {
		t.Fatal("expected error for non-replayable body")
	}
}

func TestStatusError_RetryAfterParsing(t *testing.T) {
	now := time.Now().UTC()
	headers := http.Header{}
	headers.Set("Retry-After", now.Add(2*time.Second).Format(http.TimeFormat))
	err := &integration.StatusError{Header: headers}

	d, ok := err.RetryAfter()
	if !ok || d <= 0 || d > 3*time.Second {
		t.Fatalf("retry-after=%v ok=%v, want >0 and <=3s", d, ok)
	}

	err.Header = http.Header{"Retry-After": []string{"bogus"}}
	if d, ok := err.RetryAfter(); ok || d != 0 {
		t.Fatalf("expected invalid retry-after to return false")
	}
}

func TestStatusError_ErrorPrefersWrappedErr(t *testing.T) {
	wrapped := errors.New("boom")
	err := &integration.StatusError{Err: wrapped}
	if err.Error() != "boom" {
		t.Fatalf("got %q, want %q", err.Error(), "boom")
	}

	err = &integration.StatusError{Code: 500}
	if err.Error() != "http status 500" {
		t.Fatalf("got %q, want %q", err.Error(), "http status 500")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (rt roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt(req)
}

type trackingBody struct {
	data   []byte
	read   int
	closed bool
}

func (b *trackingBody) Read(p []byte) (int, error) {
	if b.read >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.read:])
	b.read += n
	return n, nil
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}
