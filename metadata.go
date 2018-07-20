package srv

import (
	"encoding/binary"
	"io"
	"strings"
	"time"
)

// Constants describing the shape of the header.
const (
	headerSize         = 256
	headerEndpointSize = 231
)

// Constants describing endpoint types for the purposes of request routing.
const (
	EndpointRequest = 0
	EndpointStream  = 1
)

// Metadata is used to represent the header metadata extracted from a request.
type Metadata struct {
	// EndpointType, which is used as a flag to determine how to handle the
	// connection starting after the header. If it is `EndpointRequest`, it will
	// have traditional request / response semantics. If it is `EndpointStream`,
	// it will behave as a streaming endpoint.
	EndpointType byte

	// UserID, which can be used for authentication purposes.
	UserID int64

	// BodySize, which is responsible for telling the server how big the body's
	// payload is. This is discarded if the `EndpointType` is set to
	// `EndpointStream`.
	BodySize int64

	// Timeout, which allows the client to instruct the server to cancel an
	// operation if it takes over this amount of time.
	Timeout time.Duration

	// Endpoint, the name of the handler that should process this request.
	Endpoint string
}

// Encode is used to encode the metadata into a byte slice that can be used on
// the wire.
func (m Metadata) Encode() []byte {
	b := make([]byte, headerSize)
	ib := make([]byte, 8)

	b[0] = m.EndpointType

	binary.LittleEndian.PutUint64(ib, uint64(m.UserID))

	for i, bb := range ib {
		b[i+1] = bb
		ib[i] = '\x00'
	}
	binary.LittleEndian.PutUint64(ib, uint64(m.Timeout/time.Millisecond))

	for i, bb := range ib {
		b[i+9] = bb
		ib[i] = '\x00'
	}
	binary.LittleEndian.PutUint64(ib, uint64(m.BodySize))

	for i, bb := range ib {
		b[i+17] = bb
		ib[i] = '\x00'
	}
	for i, c := range m.Endpoint {
		if i >= headerEndpointSize {
			break
		}
		b[i+25] = byte(c)
	}
	return b
}

// DecodeMetadata is used to fetch metadata from a given byte slice.
func DecodeMetadata(bytes []byte) (Metadata, error) {
	m := Metadata{}

	if len(bytes) < headerSize {
		return m, io.EOF
	}
	m.EndpointType = bytes[0]
	m.UserID = int64(binary.LittleEndian.Uint64(bytes[1:9]))
	m.Timeout = time.Millisecond * time.Duration(binary.LittleEndian.Uint64(bytes[9:17]))
	m.BodySize = int64(binary.LittleEndian.Uint64(bytes[17:25]))
	m.Endpoint = strings.Trim(string(bytes[25:headerSize]), "\x00")

	return m, nil
}

// DecodeMetadataReader is used to fetch metadata from a given io.Reader.
func DecodeMetadataReader(r io.Reader) (Metadata, error) {
	var (
		err  error
		m    Metadata
		bbuf = make([]byte, 1)
		nbuf = make([]byte, 8)
		sbuf = make([]byte, headerEndpointSize)
	)
	if _, err = r.Read(bbuf); err != nil {
		return m, err
	}
	m.EndpointType = bbuf[0]

	if _, err = r.Read(nbuf); err != nil {
		return m, err
	}
	m.UserID = int64(binary.LittleEndian.Uint64(nbuf))

	if _, err = r.Read(nbuf); err != nil {
		return m, err
	}
	m.Timeout = time.Millisecond * time.Duration(binary.LittleEndian.Uint64(nbuf))

	if _, err = r.Read(nbuf); err != nil {
		return m, err
	}
	m.BodySize = int64(binary.LittleEndian.Uint64(nbuf))

	if _, err = r.Read(sbuf); err != nil {
		return m, err
	}
	m.Endpoint = strings.Trim(string(sbuf), "\x00")

	return m, nil
}
