package srv

import (
	"bytes"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
)

var errConnectionClosed = errors.New("connection already closed")

// Client is used to interact with a `Server`. It implements the following
// interfaces to make it easy to replace a raw `net.Conn`:
//
// - `io.Closer`
// - `io.Reader`
// - `io.Writer`
//
// Because of this, it also implements all of the combinations of these
// interfaces.
type Client struct {
	conn     net.Conn
	protocol string
	uri      string
	closed   bool
}

// NewClientConn is used to create a new client from the net.Conn. This client
// is mostly useful for server-side interactions, where the read functions come
// in handy.
func NewClientConn(conn net.Conn) *Client {
	return &Client{conn: conn}
}

// NewClient is used to return a new client that can be used to interact with a
// server.
func NewClient(protocol string, uri string) (*Client, error) {
	switch protocol {
	case ProtocolTCP, ProtocolUnix:
	default:
		return nil, errInvalidProtocol
	}
	conn, err := net.Dial(protocol, uri)

	if err != nil {
		return nil, errors.Wrap(err, "could not dial")
	}
	return &Client{conn: conn, protocol: protocol, uri: uri}, nil
}

// RemoteAddr is a wrapper around the conn's RemoteAddr func.
func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Write is used to implement io.Writer. Operations on a closed connection
// result in an immediate failure. Otherwise, it defers to the underlying
// `net.Conn`. **NOTE** this method has no knowledge of the structure of the
// protocol, so it should be used only in special circumstances.
func (c *Client) Write(b []byte) (n int, err error) {
	if c.closed {
		return 0, errConnectionClosed
	}
	return c.conn.Write(b)
}

// WriteMeta is used to write the metadata to the connection.
func (c *Client) WriteMeta(meta Metadata) (n int, err error) {
	if c.closed {
		return 0, errConnectionClosed
	}
	return c.Write(meta.Encode())
}

// WriteData is used as a convenience wrapper around the Write operation. It
// accepts an endpoint name and a byte slice as the body.
func (c *Client) WriteData(endpoint string, body []byte) (n int, err error) {
	if c.closed {
		return 0, errConnectionClosed
	}
	meta := Metadata{BodySize: int64(len(body)), Endpoint: endpoint}
	req := meta.Encode()

	return c.Write(append(req, body...))
}

// WriteDataString is used as a convenience wrapper around the WriteData
// operation. It just accepts a string instead of a byte slice.
func (c *Client) WriteDataString(endpoint, body string) (n int, err error) {
	return c.WriteData(endpoint, []byte(body))
}

// WriteDataReader accepts an endpoint name and an `io.Reader` as the body.
func (c *Client) WriteDataReader(endpoint string, body io.Reader) (n int, err error) {
	if c.closed {
		return 0, errConnectionClosed
	}
	buf := &bytes.Buffer{}
	bytes, err := io.Copy(buf, body)

	if err != nil {
		return 0, errors.Wrap(err, "could not copy from reader")
	}
	meta := Metadata{BodySize: bytes, Endpoint: endpoint}
	req := meta.Encode()

	return c.Write(append(req, buf.Bytes()...))
}

// Read is used to implement io.Reader. Operations on a closed connection result
// in an immediate failure. Otherwise, it defers to the underlying `net.Conn`.
// **NOTE** this method has no knowledge of the structure of the protocol, so it
// should be used only in special circumstances.
func (c *Client) Read(b []byte) (n int, err error) {
	if c.closed {
		return 0, errConnectionClosed
	}
	return c.conn.Read(b)
}

// ReadMeta is used to read the metadata from a connection. It returns the
// metadata and an error, if one occurred.
func (c *Client) ReadMeta() (meta Metadata, err error) {
	header := make([]byte, headerSize)

	if _, err = c.Read(header); err != nil {
		return meta, err
	}
	meta, err = DecodeMetadata(header)

	switch err {
	case io.EOF:
		return meta, err
	case nil:
		return meta, nil
	default:
		return meta, errors.Wrap(err, "could not decode metadata")
	}
}

// ReadBody is used to read the body from a connection, with the metadata as a
// reference (describing the size of the body).
func (c *Client) ReadBody(meta Metadata) (body []byte, err error) {
	body = make([]byte, meta.BodySize)
	_, err = c.Read(body)

	switch err {
	case io.EOF:
		return body, err
	case nil:
	default:
		return body, errors.Wrap(err, "could not read body")
	}
	return body, nil
}

// ReadData is used to read a request from the connection. It returns the
// metadata, the body as a byte slice, and an error, if one occurred.
func (c *Client) ReadData() (meta Metadata, body []byte, err error) {
	meta, err = c.ReadMeta()

	if err != nil {
		return meta, body, err
	}
	body, err = c.ReadBody(meta)
	return meta, body, err
}

// ReadDataString is used to wrap ReadData, returning a string instead of a
// byte slice.
func (c *Client) ReadDataString() (meta Metadata, body string, err error) {
	meta, bodyBytes, err := c.ReadData()
	return meta, string(bodyBytes), err
}

// Close is used to implement io.Closer. Operations on a closed connection
// result in an immediate failure. Otherwise, it defers to the underlying
// `net.Conn`.
func (c *Client) Close() error {
	c.closed = true
	return c.conn.Close()
}

// SetDeadline is used to set a deadline on the underlying connection to do some
// IO.
func (c *Client) SetDeadline(deadline time.Time) error {
	return c.conn.SetDeadline(deadline)
}
