package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

// Server - ResponseWriter - интерфейс, обеспечивающий доступ к методам, необходимым для возврата ответа клиенту
type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("serve!")
	s.proxy.ServeHTTP(rw, req)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]

	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()

	fmt.Printf("Server: %+v\n", targetServer)
	fmt.Printf("forwarding request to address %q\n", targetServer.Address())

	targetServer.Serve(rw, req)
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// main - Содержит NewLoadBalancer, который принимает массив servers (сущность simpleServer)
// И перенаправляет трафик на них (по очереди)
func main() {
	const PORT = "8000"

	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.duckduckgo.com"),
	}

	lb := NewLoadBalancer(PORT, servers)

	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}

	// При открытии этой ссылки, перенаправит трафик на первый сервер, затем на 2-й и так далее
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("serving requests at 'localhost:%s'\n", lb.port)

	err := http.ListenAndServe(":"+lb.port, nil)
	if err != nil {
		log.Fatal("Server error: ", err)
	}
}
