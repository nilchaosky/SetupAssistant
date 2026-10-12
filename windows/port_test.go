package windows

import (
	"net"
	"strconv"
	"testing"
)

func TestPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	ln.Close()

	busy, err := PortInUse(port)
	if err != nil {
		t.Fatal(err)
	}
	if busy {
		t.Fatalf("port %s is free", port)
	}

	held, err := net.Listen("tcp4", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	heldPort := strconv.Itoa(held.Addr().(*net.TCPAddr).Port)

	busy, err = PortInUse(heldPort)
	if err != nil {
		t.Fatal(err)
	}
	if !busy {
		t.Fatalf("port %s is taken", heldPort)
	}
}
