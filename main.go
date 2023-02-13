package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"syscall"
	"time"
)

// Bind ethernet interface
func bindInterface(interfaceName string) *net.Dialer {
	d := &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(network, address string, conn syscall.RawConn) error {
			//log.Println(fmt.Sprintf("Open connection to %s", address))
			var operr error
			if err := conn.Control(func(fd uintptr) {
				operr = syscall.BindToDevice(int(fd), interfaceName)
			}); err != nil {
				return err
			}
			return operr
		},
	}
	return d
}

func handleTunneling(w http.ResponseWriter, r *http.Request, interfaceName string) {
	dest_conn, err := bindInterface(interfaceName).Dial("tcp", r.Host)
	if err != nil {
		log.Print(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hitjacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		log.Println("error", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func handelHTTP(w http.ResponseWriter, req *http.Request, interfaceName string) {
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: bindInterface(interfaceName).DialContext,
			Dial:        bindInterface(interfaceName).Dial,
		},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("error", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func main() {
	port := flag.Int("p", 0, "Port")
	interfaceName := flag.String("i", "", "Outbound interface")
	flag.Parse()

	if port == nil || *port <= 0 || interfaceName == nil || *interfaceName == "" {
		fmt.Print("Please type correct args")
		flag.PrintDefaults()
		return
	}

	server := &http.Server{
		Addr: fmt.Sprintf("0.0.0.0:%d", *port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RequestURI = ""
			if r.Method == http.MethodConnect {
				handleTunneling(w, r, *interfaceName)
			} else {
				handelHTTP(w, r, *interfaceName)
			}
		}),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	log.Println("Make proxy on address 0.0.0.0 port", *port)
	log.Println("Outbound interface is: ", *interfaceName)
	log.Println("Outbound IP is:", GetOutboundIP())
	log.Fatal(server.ListenAndServe())
}
