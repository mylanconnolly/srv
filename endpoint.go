package srv

import (
	"io"
)

// RequestEndpoint is the type describing a traditional request / response
// endpoint for the server.
type RequestEndpoint func(meta Metadata, w io.Writer, r io.Reader) error

// StreamingEndpoint is the type describing a streaming endpoint for the server.
// These endpoints have access to the net.Conn object, and should be responsible
// for the full lifetime of the connection, including closing it when they are
// done. This allows maximum flexibility.
type StreamingEndpoint func(meta Metadata, client *Client) error
