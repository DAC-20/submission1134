package utils

import (
	"net"
	"fmt"
	"log"
	"strconv"
	"time"
)

func IsPortOccupied(ip string, port int) bool {
	addr := ip + ":" + strconv.Itoa(port)
	conn, err := net.DialTimeout("tcp", addr, 200 * time.Millisecond)
	if err != nil {
		return false
	} else {
		conn.Close()
		return true
	}
}

func TcpDial(ip string, port string, msg []byte) ([]byte, error) {
	conn, err := net.Dial("tcp", ip + ":" + port)
	if err != nil {
		err1 := fmt.Errorf("tcp dial error: %v", err)
		return []byte{}, err1
	}
	defer conn.Close()
	conn.Write(msg)
	data := make([]byte, 10000)
	conn.Read(data)
	log.Println("trace: get data:", string(data))
	i := 0
	for i = range data {
		if data[i] == 0 {
			break
		}
	}
	return data[0:i], nil
}

func CloseConns(conns []net.Conn) {
	for _, conn := range conns {
		conn.Close()
	}
}