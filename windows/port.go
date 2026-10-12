package windows

import (
	"errors"
	"net"
	"syscall"

	"golang.org/x/sys/windows"
)

// PortInUse 检查本机是否已有程序监听 port。
func PortInUse(port string) (bool, error) {
	for _, network := range []string{"tcp4", "tcp6"} {
		ln, err := net.Listen(network, ":"+port)
		if err == nil {
			ln.Close()
			continue
		}
		if addrInUse(err) {
			return true, nil
		}
		if network == "tcp6" {
			continue
		}
		return false, err
	}
	return false, nil
}

func addrInUse(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && errno == windows.WSAEADDRINUSE
}
