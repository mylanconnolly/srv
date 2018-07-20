package srv

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		uri      string
		wantErr  bool
	}{
		{"invalid protocol", "foo", "127.0.0.1:1234", true},
		{"invalid URI", "tcp", "-", true},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, err := NewServer(tt.protocol, tt.uri)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Should return an error")
				}
			} else {
				if err != nil {
					t.Errorf("Should not return an error, got %v", err)
				}
				if server == nil {
					t.Errorf("Server should not be nil")
					return
				}
				if server.protocol != tt.protocol {
					t.Errorf("Server protocol = %v, want %v", server.protocol, tt.protocol)
				}
				if server.uri != tt.uri {
					t.Errorf("Server uri = %v, want %v", server.uri, tt.uri)
				}
			}
		})
	}
}

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

func TestServerMaybeLogf(t *testing.T) {
	tests := []struct {
		name   string
		server *Server
	}{
		{"empty server", &Server{}},
		{"logging set to true", &Server{Log: true}},
		{"logging set to false", &Server{Log: false}},
	}
	// We can't run these tests in parallel.

	// Capture logging output so we can inspect it
	out := bytes.Buffer{}
	log.SetOutput(&out)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ilen := out.Len()

			tt.server.maybeLogf("something %v", "foo")

			if tt.server.Log {
				if out.Len() == ilen {
					t.Error("Did not log message when we expected it to")
				}
			} else {
				if out.Len() > ilen {
					t.Error("Logged message when we did not expect it to")
				}
			}
		})
	}
	// Set the logging output back to normal
	log.SetOutput(os.Stderr)
}

func TestServerMaybeLogln(t *testing.T) {
	tests := []struct {
		name   string
		server *Server
	}{
		{"empty server", &Server{}},
		{"logging set to true", &Server{Log: true}},
		{"logging set to false", &Server{Log: false}},
	}
	// We can't run these tests in parallel.

	// Capture logging output so we can inspect it
	out := bytes.Buffer{}
	log.SetOutput(&out)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ilen := out.Len()

			tt.server.maybeLogln("something")

			if tt.server.Log {
				if out.Len() == ilen {
					t.Error("Did not log message when we expected it to")
				}
			} else {
				if out.Len() > ilen {
					t.Error("Logged message when we did not expect it to")
				}
			}
		})
	}
	// Set the logging output back to normal
	log.SetOutput(os.Stderr)
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

	time.Sleep(1 * time.Second)

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

func BenchmarkEchoServerIndividualConnections(b *testing.B) {
	s, _ := NewServer(ProtocolTCP, "127.0.0.1:1337")
	body := []byte("hello world")

	s.AddRequestEndpoint("hello", func(meta Metadata, w io.Writer, r io.Reader) error {
		w.Write(body)
		return nil
	})

	go func() {
		s.Listen()
	}()

	time.Sleep(1 * time.Second)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		client, _ := NewClient(ProtocolTCP, "localhost:1337")
		client.WriteData("hello", body)
		client.ReadData()
		client.Close()
	}
}

func BenchmarkEchoServerHTTP(b *testing.B) {
	go func() {
		http.ListenAndServe("127.0.0.1:1337", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		}))
	}()

	time.Sleep(1 * time.Second)

	b.ResetTimer()

	body := []byte("hello")

	for n := 0; n < b.N; n++ {
		buf := bytes.NewBuffer(body)
		http.Post("http://127.0.0.1:1337", "text/plain", buf)
	}
}
