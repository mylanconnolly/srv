package srv

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Protocol constants
const (
	ProtocolTCP  = "tcp"
	ProtocolUnix = "unix"
)

var (
	errInvalidProtocol = errors.New("invalid protocol specified")
	errInvalidEndpoint = errors.New("invalid endpoint specified")
)

// NewServer is used to return a default Server.
func NewServer(protocol, uri string) (*Server, error) {
	switch protocol {
	case ProtocolTCP, ProtocolUnix:
	default:
		return nil, errInvalidProtocol
	}
	return &Server{
		MaxRetries:       10,
		Protocol:         protocol,
		URI:              uri,
		requestEndpoints: map[string]RequestEndpoint{},
		willShutdown:     make(chan struct{}),
		didShutdown:      make(chan struct{}),
	}, nil
}

// Server is used to handle serving requests.
type Server struct {
	MaxRetries int
	MaxTimeout time.Duration
	Protocol   string
	URI        string
	Log        bool

	// Internal fields; used to keep track of connection state, etc.
	requestEndpoints   map[string]RequestEndpoint   // A map of endpoints, representing all the possible handlers for requests.
	streamingEndpoints map[string]StreamingEndpoint // A map of streaming endpionts, representing all the possible handlers for streaming requests.
	willShutdown       chan struct{}                // Notifies the listen process that we should shutdown.
	didShutdown        chan struct{}                // Notifies the shutdown process that we did shutdown.
	wg                 sync.WaitGroup               // This keeps a counter of how many clients are connected for gracefully shutting down
}

// AddRequestEndpoint is used to add an endpoint to the internal set of
// endpoints.
func (s *Server) AddRequestEndpoint(name string, endpoint RequestEndpoint) {
	s.requestEndpoints[name] = endpoint
}

// AddStreamingEndpoint is used to add an endpoint to the internal set of
// endpoints.
func (s *Server) AddStreamingEndpoint(name string, endpoint StreamingEndpoint) {
	s.streamingEndpoints[name] = endpoint
}

// ListenTLS is used to listen for requests using TLS encryption. This is only
// possible when using TCP.
func (s *Server) ListenTLS(cert, key, ca string) error {
	switch s.Protocol {
	case ProtocolTCP:
		return s.listenTCPTLS(cert, key, ca)
	default:
		return errInvalidProtocol
	}
}

// Listen is used to listen for requests on the specified URI and protocol.
func (s *Server) Listen() error {
	switch s.Protocol {
	case ProtocolTCP:
		return s.listenTCP()
	case ProtocolUnix:
		return s.listenUnix()
	default:
		return errInvalidProtocol
	}
}

// Shutdown is used to tell the server to stop listening for requests.
func (s *Server) Shutdown() {
	s.willShutdown <- struct{}{}
	<-s.didShutdown
}

func (s *Server) handleShutdown(listener net.Listener) error {
	if err := listener.Close(); err != nil {
		return err
	}
	s.wg.Wait()
	s.didShutdown <- struct{}{}
	return nil
}

func (s *Server) listenTCPTLS(cert, key, ca string) error {
	return nil
}

func (s *Server) listenTCP() error {
	timeout, tries := defaultRetries()
	addr, err := net.ResolveTCPAddr(ProtocolTCP, s.URI)

	if err != nil {
		return err
	}
	listener, err := net.ListenTCP(ProtocolTCP, addr)

	if err != nil {
		return err
	}
	if s.Log {
		log.Printf("Listening for requests on tcp://%s", s.URI)
	}
	defer listener.Close()

	for {
		select {
		case <-s.willShutdown:
			return s.handleShutdown(listener)
		default:
		}
		if err = listener.SetDeadline(newDeadline(1 * time.Second)); err != nil {
			return err
		}
		conn, err := listener.AcceptTCP()

		switch e := err.(type) {
		case net.Error:
			if e.Timeout() {
				continue
			}
			if e.Temporary() {
				if tries > s.MaxRetries {
					return e
				}
				timeout, tries = incrementRetries(timeout, tries)
				time.Sleep(timeout)
				continue
			}
		default:
			if err != nil {
				conn.Close()
				return e
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) listenUnix() error {
	addr, err := net.ResolveUnixAddr(ProtocolUnix, s.URI)

	if err != nil {
		return err
	}
	listener, err := net.ListenUnix(ProtocolUnix, addr)

	if err != nil {
		return err
	}
	if s.Log {
		log.Printf("Listening for requests on unix://%s", s.URI)
	}
	defer listener.Close()

	for {
		select {
		case <-s.willShutdown:
			return s.handleShutdown(listener)
		default:
		}
		if err = listener.SetDeadline(newDeadline(1 * time.Second)); err != nil {
			return err
		}
		conn, err := listener.AcceptUnix()

		switch e := err.(type) {
		default:
			if err != nil {
				conn.Close()
				return e
			}
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		s.wg.Done()
		conn.Close()
		s.maybeLogf("Client disconnected: %v", conn.RemoteAddr())
	}()

	s.wg.Add(1)
	s.maybeLogf("Client connected: %v", conn.RemoteAddr())

	client := NewClientConn(conn)

	if err := s.setDeadline(client); err != nil {
		s.maybeLogf("Error setting deadline on connection: %v", err)
	}
	var (
		meta Metadata
		err  error
	)
	for {
		meta, err = client.ReadMeta()

		if err != nil {
			s.logReadError(err, "Unable to read metadata")
			return
		}
		switch meta.EndpointType {
		case EndpointRequest:
			if err = s.handleRequestConn(meta, client); err != nil {
				return
			}
		case EndpointStream:
			if err = s.handleStreamingConn(meta, client); err != nil {
				return
			}
		default:
			s.maybeLogf("Invalid endpoint type specified: %v", meta.EndpointType)
			return
		}
	}
}

func (s *Server) handleStreamingConn(meta Metadata, client *Client) error {
	endpoint, ok := s.streamingEndpoints[meta.Endpoint]

	if !ok {
		s.maybeLogf("Could not find requested endpoint: %v", meta.Endpoint)
		return errInvalidEndpoint
	}
	return endpoint(meta, client)
}

func (s *Server) handleRequestConn(meta Metadata, client *Client) error {
	endpoint, ok := s.requestEndpoints[meta.Endpoint]

	if !ok {
		s.maybeLogf("Could not find requested endpoint: %v", meta.Endpoint)
		return errInvalidEndpoint
	}
	body, err := client.ReadBody(meta)

	if err != nil {
		s.logReadError(err, "Unable to read body")
		return err
	}
	wbuf := &bytes.Buffer{}
	rbuf := bytes.NewBuffer(body)

	if err = endpoint(meta, wbuf, rbuf); err != nil {
		s.maybeLogf("Error serving endpoint: %v", err)
		return err
	}
	if _, err := client.WriteData(meta.Endpoint, wbuf.Bytes()); err != nil {
		s.maybeLogf("Error writing response: %v", err)
		return err
	}
	if err = s.setDeadline(client); err != nil {
		s.maybeLogf("Error setting deadline on connection: %v", err)
		return err
	}
	return nil
}

// This is a simple wrapper around the stdlib's logging facilities. It checks
// if logging was requested, then logs the message. If logging was not requested,
// nothing happens.
func (s *Server) maybeLogf(format string, v ...interface{}) {
	if s.Log {
		log.Printf(format, v...)
	}
}

// This is a simple wrapper around the stdlib's logging facilities. It checks
// if logging was requested, then logs the message. If logging was not requested,
// nothing happens.
func (s *Server) maybeLogln(v ...interface{}) {
	if s.Log {
		log.Println(v...)
	}
}

// This function is designed to help simplify logging errors from io.Readers. We
// can  encounter the io.EOF error, which doesn't mean a problem occurred,
// simply that the client disconnected, so we just discard those errors.
func (s *Server) logReadError(err error, msg string) {
	if err == io.EOF { // Client disconnected; no need to log
		return
	}
	s.maybeLogln(msg, err)
}

// If timeouts were requested, we set the deadline here. This could prevent
// users from saturating connections by holding onto connections and not
// actually performing any IO.
func (s *Server) setDeadline(client *Client) error {
	if s.MaxTimeout > 0 {
		if err := client.SetDeadline(newDeadline(s.MaxTimeout)); err != nil {
			return err
		}
	}
	return nil
}

func newDeadline(duration time.Duration) time.Time {
	return time.Now().Add(duration)
}

func defaultRetries() (timeout time.Duration, tries int) {
	return 10 * time.Millisecond, 0
}

func incrementRetries(timeout time.Duration, tries int) (time.Duration, int) {
	return timeout * 2, tries + 1
}
