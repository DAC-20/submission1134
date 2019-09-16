package main
import (
	"log"
	"strings"
	"os"
	"io"
	"bytes"
	"net"
	"strconv"
	"time"
	"encoding/json"
	"sync"
)

// #cgo CFLAGS: -I${SRCDIR} 
// #cgo LDFLAGS: -L${SRCDIR} -lvmult_slr_assign_fragment -L/opt/xilinx/xrt/lib/ -lOpenCL -lpthread -lrt  -lstdc++
// #include "src/vmult_slr_assign_fragment.h"
import "C"

type Request struct {
	Input string 
}

type TimePack struct {
  Get         time.Time
  Issue       time.Time
  Finish      time.Time
  Response    time.Time
  Next *TimePack
}

type Response struct {
	Output string 
	Time  TimePack
}

// const (
// 	exe   = "kernel_global"
// 	arg   = "xclbin/krnl_kernel_global.hw.xilinx_u200_xdma_201830_1.xclbin"
// )

var (
	FPGA_APP_DIR        string
	FPGA_LISTEN_PORT    string
	mutex               sync.Mutex
	taskCount           int
	mutex0              sync.Mutex
	mutex1              sync.Mutex
	mutex2              sync.Mutex
)

func handleConn(conn net.Conn, get time.Time, req Request, id int) {
	defer conn.Close()
	mutex.Lock()
	taskId := taskCount
	taskCount += 1
	mutex.Unlock()
	response := Response{}
	response.Time.Get = get
	argv := strings.Fields(req.Input)
	oriStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	var code int
	switch taskId % 3 {
	case 0:
		mutex0.Lock()
		response.Time.Issue = time.Now()
		code = int(C.compute_batch_frag0(C.CString(argv[2])))
		response.Time.Finish = time.Now()
		mutex0.Unlock()
	case 1:
		mutex1.Lock()
		response.Time.Issue = time.Now()
		code = int(C.compute_batch_frag1(C.CString(argv[2])))
		response.Time.Finish = time.Now()
		mutex1.Unlock()
	case 2:
		mutex2.Lock()
		response.Time.Issue = time.Now()
		code = int(C.compute_batch_frag2(C.CString(argv[2])))
		response.Time.Finish = time.Now()
		mutex2.Unlock()
	}
	// mutex.Lock()
	// // err := cmd.Run()
	// // code := C.compute_batch(C.CString(batchNum))
	// code := C.compute_batch(C.CString(argvs[2]), C.CString(argvs[3]), C.CString(argvs[4]), C.CString(argvs[5]))
	// mutex.Unlock()
	w.Close()
	os.Stdout = oriStdout
	if code != 0 {
		log.Printf("compute batch error code %d\n", code)
		return
	} else {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		response.Output = buf.String()
	}
	response.Time.Response = time.Now()
	jres, err := json.Marshal(response)
	if err != nil {
		log.Println(err)
		return
	}
	conn.Write(jres)
	log.Printf("task %d (issued to %d) time %v, function time %v\n", taskId, taskId % 3, response.Time.Response.Sub(response.Time.Get), response.Time.Finish.Sub(response.Time.Issue))
}

func tcpListen() error {
	log.Println("begin tcpListen...")
	count := 0
	tcp_int_port, _ := strconv.Atoi(FPGA_LISTEN_PORT)
	l, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: tcp_int_port,
	})
	if err != nil {
		log.Println("listen error:", err)
		return err
	}
	defer l.Close()
	latestAcceptTime := time.Now()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("accept error:", err)
			return err
		}
		count += 1
		data := make([]byte, 10000)
		if _, err := conn.Read(data); err != nil {
			log.Printf("conn read err: %v\n", err)
			conn.Close()
			continue
		}
		get := time.Now()
		i := 0
		for i = range data {
			if data[i] == 0 {
				break
			}
		}
		// if string(data[0:i]) == "healthy?" {
		// 	conn.Write("healthy!")
		// 	conn.Close()
		// 	continue
		// }
		var request Request
		if err := json.Unmarshal(data[0:i], &request); err != nil {
			log.Println(err)
			return err
		}
		log.Println(count, "tcp get request input:", request.Input, "request distance", get.Sub(latestAcceptTime))
		latestAcceptTime = get
		go handleConn(conn, get, request, count)
	}
}

var (
	maxBatchSize int = 16
	dataSize     int = 49152 * 5
  )

func main() {
	FPGA_APP_DIR = os.Getenv("FPGA_APP_DIR")
	FPGA_LISTEN_PORT = os.Getenv("FPGA_LISTEN_PORT")
	log.Println("FPGA_APP_DIR =", FPGA_APP_DIR, "FPGA_LISTEN_PORT =", FPGA_LISTEN_PORT)

	if code := C.init(C.CString("xclbin/vmult_vadd.hw.xilinx_u200_xdma_201830_1.xclbin"), C.CString(strconv.Itoa(dataSize)), C.CString(strconv.Itoa(maxBatchSize))); code != 0 {
		log.Fatalf("init error: %d", code)
	}

	if err := tcpListen(); err != nil {
		C.cleanup()
		log.Printf("exit now: %v\n", err)
	}
}
