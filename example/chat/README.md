# Chat

This is an example chat server and client over TCP. It illustrates using the
`srv.Server` in conjunction with some managed state. In this case, the state is
all of the connected clients (broadcasting the message to each connected client).

This example is inspired by the chat example application in Chapter 8 of
[The Go Programming Language](http://www.gopl.io/) (an excellent book). If you
have not read it, I highly recommend doing so.
