# bcache

[![godoc](https://godoc.org/github.com/iwanbk/bcache?status.svg)](http://godoc.org/github.com/iwanbk/bcache)
[![Build Status](https://travis-ci.org/iwanbk/bcache.svg?branch=master)](https://travis-ci.org/iwanbk/bcache)
[![codecov](https://codecov.io/gh/iwanbk/bcache/branch/master/graph/badge.svg)](https://codecov.io/gh/iwanbk/bcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/iwanbk/bcache)](https://goreportcard.com/report/github.com/iwanbk/bcache)

A Go Library to create distributed in-memory cache.

It uses [Gossip Protocol](https://en.wikipedia.org/wiki/Gossip_protocol) for data synchronization between peers.

## Features

- LRU cache
- Eventual Consistency synchronization between peers
- Data are replicated to all peers
- cache filling mechanism (TODO)

## Why using it

- if extra network hops needed by `redis` or `memcached` is not acceptable for you
- you only need cache with simple `Set` & `Get` operation

## Quick Start

In server 1
```go
bc, err := New(Config{
	PeerID:     1,
	ListenAddr: "192.168.0.1:12345",
	Peers:      nil,
	MaxKeys:    1000,
	Logger:     logrus.New(),
})
if err != nil {
    log.Fatalf("failed to create cache: %v", err)
}
bc.Set("my_key", "my_val")
```

In server 2
```go
bc, err := New(Config{
	PeerID:     2,
	ListenAddr: "192.168.0.2:12345",
	Peers:      []string{"192.168.0.1:12345"},
	MaxKeys:    1000,
	Logger:     logrus.New(),
})
if err != nil {
    log.Fatalf("failed to create cache: %v", err)
}
bc.Set("my_key2", "my_val2")
```

In server 3
```go
bc, err := New(Config{
	PeerID:     3,
	ListenAddr: "192.168.0.3:12345",
	Peers:      []string{"192.168.0.1:12345"},
	MaxKeys:    1000,
	Logger:     logrus.New(),
})
if err != nil {
    log.Fatalf("failed to create cache: %v", err)
}
val, exists := bc.Get("my_key2")
```

