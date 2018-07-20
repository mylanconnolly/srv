package srv

import (
	"fmt"
	"io"
	"math"
	"os"
	"testing"
	"time"
)

func TestNewDeadline(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"test", 100 * time.Millisecond},
		{"test", 1000 * time.Millisecond},
		{"test", 10000 * time.Millisecond},
		{"test", 100 * time.Second},
		{"test", 1000 * time.Second},
		{"test", 10000 * time.Second},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now()
			deadline := newDeadline(tt.duration)
			delta := now.Add(tt.duration).Sub(deadline).Nanoseconds()

			// We account for 10 microseconds of difference, since sometimes there is
			// a delay in the test that causes this to be off.
			if math.Abs(float64(delta)) > 10000 {
				t.Errorf("Unexpected deadline value was off by %v", delta)
			}
		})
	}
}

func TestDefaultRetries(t *testing.T) {
	wantTimeout := 10 * time.Millisecond
	wantTries := 0
	timeout, tries := defaultRetries()

	if timeout != wantTimeout {
		t.Errorf("timeout = %v, want %v", timeout, wantTimeout)
	}
	if tries != wantTries {
		t.Errorf("tries = %v, want %v", tries, wantTries)
	}
}

func TestIncrementTries(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		tries   int
	}{
		{"initial", 10 * time.Millisecond, 0},
		{"incremented", 20 * time.Millisecond, 1},
		{"incremented", 40 * time.Millisecond, 2},
		{"incremented", 80 * time.Millisecond, 3},
		{"incremented", 160 * time.Millisecond, 4},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wantTimeout := tt.timeout * 2
			wantTries := tt.tries + 1
			timeout, tries := incrementRetries(tt.timeout, tt.tries)

			if timeout != wantTimeout {
				t.Errorf("timeout = %v, want %v", timeout, wantTimeout)
			}
			if tries != wantTries {
				t.Errorf("tries = %v, want %v", tries, wantTries)
			}
		})
	}
}

func BenchmarkEchoServerSharedConnections(b *testing.B) {
	s, _ := NewServer(ProtocolTCP, "127.0.0.1:1337")
	body := []byte("hello world")

	s.AddRequestEndpoint("hello", func(meta Metadata, w io.Writer, r io.Reader) error {
		w.Write(body)
		return nil
	})

	go func() {
		s.Listen()
	}()

	client, err := NewClient(ProtocolTCP, "localhost:1337")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		client.WriteData("hello", body)
		client.ReadData()
	}
}
