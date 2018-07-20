# srv

This is a library used to help set up a microservice in Go using lighter-weight
technology than HTTP to transport the messages. Focus will initially be on
getting TCP to work, but Unix domain sockets will also be added. This may
extend in the future.

The goal is to create this using no dependencies outside of the stdlib, in order
to keep compilation size / time down, as well as to prevent pulling in a ton of
dependencies not everyone will need.

> **NOTE**
>
> Right now, this is mostly a learning exercise. It is not complete and should
> not be considered for use in production environments. Hopefully it will be,
> one day!
>
> There will be breaking changes.

## Protocol

The wire protocol is text-based, and is fairly light-weight. There is a
fixed-length header and a body of a declared length. There is by no means
anything clever or groundbreaking going on here, although it seems to be working
well enough in my basic testing.

### Header

The header is 256 bytes long, consisting of the following header values, in
order:

| Order | Size (bytes) | Type           | Description                                                         |
| :---- | :----------- | :------------- | :------------------------------------------------------------------ |
| 0     | 1            | Byte           | Endpoint type (request/response or streaming)                       |
| 1     | 8            | 64-bit Integer | User ID for authentication (if applicable)                          |
| 2     | 8            | 64-bit Integer | Timeout in milliseconds (used to set a timeout, if greater than 0)  |
| 3     | 8            | 64-bit Integer | Size of the body (used for decoding purposes)                       |
| 4     | 231          | String         | Name of the endpoint to handle the request (used to route requests) |

### Endpoint Types

There are two possible types of endpoints:

- Request
- Streaming

Request endpoints are used to emulate the traditional request / response cycle.

Streaming will open a streaming connection where the endpoint has access to the
`Client`, and manages the connection more directly. This could enable
streaming media, chat servers, etc.

## Server

The server is able to listen on either TCP or Unix domain sockets. Additionally,
we can utilize TLS encryption for added security.

## Client

The client is designed to be a wrapper around the underlying `net.Conn`, so that
we can implement some helper functions related to the wire protocol. Otherwise,
it could be used as an `io.Reader` or `io.Writer` in streaming connections. The
only real requirement is that the first communication with the server must be a
metadata header, so that the server knows how to dispatch the connection. Some
functions automate this, if you want to take advantage of it.

## TODO

There is a lot of functionality that is missing, which I would like to add. This
includes:

- [ ] A callback for verifying authentication (we currently have very weak
      authentication support, consisting of a user ID).
- [ ] A way to implement middleware.
