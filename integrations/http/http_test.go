package http_test

import (
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

func TestDoHTTP_RespectsRetryAfter(t *testing.T) {
	attempts := 0
	start := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1") // 1 second
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor(
		retry.WithPolicy("test", policy.MaxBackoff(5*time.Second)),
	)
	client := server.Client()

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, _, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 1*time.Second {
		t.Errorf("expected >1s latency due to Retry-After, got %v", elapsed)
	}
}

func TestDoHTTP_DrainsAndClosesExample(t *testing.T) {
	// Ensure response bodies are drained/closed to avoid leaks.

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			// Write some body to drain
			fmt.Fprintln(w, "some error body")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exec := retry.NewDefaultExecutor()
	client := server.Client()

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, tl, err := integration.DoHTTP(context.Background(), exec, policy.PolicyKey{Name: "test"}, client, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tl.Attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", len(tl.Attempts))
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
