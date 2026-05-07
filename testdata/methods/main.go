package main

import (
	"fmt"
	"time"
)

type Server struct {
	name string
}

func NewServer(name string) *Server {
	return &Server{name: name}
}

func (s *Server) Start() {
	time.Sleep(10 * time.Millisecond)
	s.listen()
}

func (s *Server) listen() {
	time.Sleep(5 * time.Millisecond)
}

func main() {
	srv := NewServer("test")
	srv.Start()
	fmt.Println("done")
}
