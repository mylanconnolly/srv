package srv

import (
	"net"
	"strconv"
	"testing"
)

func TestNewClient(t *testing.T) {
	port := 12309

	go func() {
		listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))

		if err != nil {
			t.Errorf("Could not start listener: %v", err)
			t.Fail()
		}
		for {
			conn, _ := listener.Accept()
			conn.Close()
		}
	}()

	tests := []struct {
		name     string
		protocol string
		uri      string
		wantErr  bool
	}{
		{"invalid protocol", "foo", "127.0.0.1:1234", true},
		{"invalid URI", "tcp", "-", true},
		{"valid URI", "tcp", "127.0.0.1:" + strconv.Itoa(port), false},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.protocol, tt.uri)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Should return an error")
				}
			} else {
				if err != nil {
					t.Errorf("Should not return an error, got %v", err)
				}
				if client == nil {
					t.Errorf("Client should not be nil")
					return
				}
				if client.conn == nil {
					t.Errorf("Connection should not be nil")
				}
				if client.protocol != tt.protocol {
					t.Errorf("Client protocol = %v, want %v", client.protocol, tt.protocol)
				}
				if client.uri != tt.uri {
					t.Errorf("Client uri = %v, want %v", client.uri, tt.uri)
				}
			}
		})
	}
}
