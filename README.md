# grpc-foundation

A gRPC service needs the same things settled before it answers its first
request: a server with connection limits and panic recovery, telemetry reporting
under one identity, a logger that reaches request handlers carrying the trace
they belong to, errors a caller can act on, and a shutdown that stops serving
and then exports what it recorded.

Each of those is a decision. Made once per service they drift, and two services
in one deployment end up disagreeing about what a timeout is or what an error
looks like. This library makes them once.

## Installation

```bash
go get git.sonicoriginal.software/grpc-foundation
```

## Usage

`server/example_test.go` holds the startup and shutdown sequence as a Go
`Example`. It has no `// Output:` comment, so `go test` compiles it and never
runs it, which keeps it type-checked against the real API. Read it there rather
than from a copy here.
