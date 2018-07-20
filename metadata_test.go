package srv

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"testing"
	"time"
)

const MaxUint = ^uint(0)
const MaxInt = int64(MaxUint >> 1)

func emptySlice(size int) []byte {
	b := make([]byte, size)

	for i := 0; i < size; i++ {
		b[i] = '\x00'
	}
	return b
}

func bigString(size int) string {
	s := ""

	for i := 0; i < size; i++ {
		s += "a"
	}
	return s
}

// Warning; this doesn't sanitize the endpoint to make sure that it is not too
// large.
func makeHeader(endpointType byte, userID, timeout, size int64, contentType, endpoint string) []byte {
	b := make([]byte, headerSize)
	ub := make([]byte, 8)
	tb := make([]byte, 8)
	sb := make([]byte, 8)

	binary.LittleEndian.PutUint64(ub, uint64(userID))
	binary.LittleEndian.PutUint64(tb, uint64(timeout))
	binary.LittleEndian.PutUint64(sb, uint64(size))

	b[0] = endpointType

	for i, bb := range ub {
		b[i+1] = bb
	}
	for i, bb := range tb {
		b[i+9] = bb
	}
	for i, bb := range sb {
		b[i+17] = bb
	}
	for i, bb := range []byte(contentType) {
		b[i+25] = bb
	}
	for i, bb := range []byte(endpoint) {
		b[i+125] = bb
	}
	return b
}

func TestDecodeMetadataReader(t *testing.T) {
	tests := []struct {
		name    string
		reader  io.Reader
		want    Metadata
		wantErr bool
	}{
		{
			"Empty header",
			bytes.NewBuffer(emptySlice(headerSize)),
			Metadata{},
			false,
		},
		{
			"Populated header 1",
			bytes.NewBuffer(makeHeader(0, 123, 456, 789, "text/plain", "foo")),
			Metadata{UserID: 123, Timeout: 456 * time.Millisecond, BodySize: 789, ContentType: "text/plain", Endpoint: "foo"},
			false,
		},
		{
			"Populated header 1",
			bytes.NewBuffer(makeHeader(1, MaxInt, 1972348976, 9817263487916234, "text/plain", "foo")),
			Metadata{EndpointType: 1, UserID: MaxInt, Timeout: 1972348976 * time.Millisecond, BodySize: 9817263487916234, ContentType: "text/plain", Endpoint: "foo"},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := DecodeMetadataReader(tt.reader)

			if !reflect.DeepEqual(metadata, tt.want) {
				t.Errorf("metadata = %#v, want %#v", metadata, tt.want)
			}
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error to be nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected error to be nil, got %v", err)
				}
			}
		})
	}
}

func TestDecodeMetadata(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		want    Metadata
		wantErr bool
	}{
		{
			"Empty header",
			emptySlice(headerSize),
			Metadata{},
			false,
		},
		{
			"Populated header 1",
			makeHeader(0, 123, 456, 789, "text/plain", "foo"),
			Metadata{UserID: 123, Timeout: 456 * time.Millisecond, BodySize: 789, ContentType: "text/plain", Endpoint: "foo"},
			false,
		},
		{
			"Populated header 1",
			makeHeader(1, MaxInt, 1972348976, 9817263487916234, "text/plain", "foo"),
			Metadata{EndpointType: 1, UserID: MaxInt, Timeout: 1972348976 * time.Millisecond, BodySize: 9817263487916234, ContentType: "text/plain", Endpoint: "foo"},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metadata, err := DecodeMetadata(tt.bytes)

			if !reflect.DeepEqual(metadata, tt.want) {
				t.Errorf("metadata = %#v, want %#v", metadata, tt.want)
			}
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error to be nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected error to be nil, got %v", err)
				}
			}
		})
	}
}

func TestMetadataEncode(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		want     []byte
	}{
		{
			"Empty metadata",
			Metadata{},
			emptySlice(headerSize),
		},
		{
			"Populated metadata 1",
			Metadata{UserID: 123, Timeout: 456 * time.Millisecond, BodySize: 789, ContentType: "text/plain", Endpoint: "foo"},
			makeHeader(0, 123, 456, 789, "text/plain", "foo"),
		},
		{
			"Populated metadata 2",
			Metadata{EndpointType: 1, UserID: MaxInt, Timeout: 1098374 * time.Millisecond, BodySize: 7613947812643, ContentType: "text/plain", Endpoint: "foo"},
			makeHeader(1, MaxInt, 1098374, 7613947812643, "text/plain", "foo"),
		},
		{
			"Truncated string",
			Metadata{EndpointType: 1, UserID: MaxInt, Timeout: 1098374 * time.Millisecond, BodySize: 7613947812643, ContentType: "text/plain", Endpoint: bigString(500)},
			makeHeader(1, MaxInt, 1098374, 7613947812643, "text/plain", bigString(headerEndpointSize)),
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			header := tt.metadata.Encode()

			if !reflect.DeepEqual(header, tt.want) {
				t.Errorf("header = %#v, want %#v", header, tt.want)
			}
		})
	}
}

func BenchmarkMetadataEncode(b *testing.B) {
	metadata := Metadata{
		UserID:   118792346,
		Timeout:  12348 * time.Millisecond,
		BodySize: 16408716234,
	}
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		metadata.Encode()
	}
}

func BenchmarkDecodeMetadataReader(b *testing.B) {
	buf := makeHeader(1, 123, 456, 789, "text/plain", "hello")

	for n := 0; n < b.N; n++ {
		b.StopTimer()
		reader := bytes.NewBuffer(buf)
		b.StartTimer()

		DecodeMetadataReader(reader)
	}
}

func BenchmarkDecodeMetadata(b *testing.B) {
	buf := makeHeader(1, 123, 456, 789, "text/plain", "hello")

	for n := 0; n < b.N; n++ {
		DecodeMetadata(buf)
	}
}
