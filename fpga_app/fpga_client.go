package main
import (
	"log"
	"os/exec"
	"os"
	"bytes"
	"net"
	"strconv"
	"time"
	"encoding/json"
	"sync"
)

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

const (
	exe   = "kernel_global"
	arg   = "xclbin/krnl_kernel_global.hw.xilinx_u200_xdma_201830_1.xclbin"
)

var (
	FPGA_APP_DIR        string
	FPGA_LISTEN_PORT    string
	mutex               sync.Mutex
)

func handleConn(conn net.Conn, get time.Time, req Request, id int) {
	defer conn.Close()
	response := Response{}
	response.Time.Get = get
	// arg1 := FPGA_APP_DIR + exe
	// arg2 := FPGA_APP_DIR + arg
	// cmd := exec.Command(arg1, arg2)
	// cmdLine := "cd " + FPGA_APP_DIR + "; ./" + exe + " " + arg
	cmdLine := "cd " + FPGA_APP_DIR + "; " + req.Input
	cmd := exec.Command("bash", "-c", cmdLine)
	//stdout, err := cmd.StdoutPipe()
	//if err != nil {
	//  log.Fatalln(err)
	//}
	//defer stdout.Close()
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	mutex.Lock()
	response.Time.Issue = time.Now()
	err := cmd.Run()
	response.Time.Finish = time.Now()
	mutex.Unlock()
	if err != nil {
		// log.Fatalln(err)
		// return
		log.Println(err)
		response.Output = string(err.Error())
	} else {
		stdout := outbuf.String()
		// stderr := errbuf.String()
		// log.Println("stdout:", stdout)
		// log.Println("stderr:", stderr)
		response.Output = stdout
	}
	response.Time.Response = time.Now()
	jres, err := json.Marshal(response)
	if err != nil {
		// log.Fatalln(err)
		log.Println(err)
		return
	}
	// log.Println("json response:", string(jres))
	// conn.Write([]byte(stdout))
	conn.Write(jres)
	log.Println(id, "time", response.Time.Response.Sub(response.Time.Get), "function time", response.Time.Finish.Sub(response.Time.Issue))
}

func tcpListen() error {
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
		conn.Read(data)
		get := time.Now()
		i := 0
		for i = range data {
			if data[i] == 0 {
				break
			}
		}
		var request Request
		if err := json.Unmarshal(data[0:i], &request); err != nil {
			// log.Fatalln(err)
			log.Println(err)
			return err
		}
		// msg := string(data)
		log.Println(count, "tcp get request input:", request.Input, "request distance", get.Sub(latestAcceptTime))
		latestAcceptTime = get
		go handleConn(conn, get, request, count)
	}
}


func main() {
	FPGA_APP_DIR = os.Getenv("FPGA_APP_DIR")
	FPGA_LISTEN_PORT = os.Getenv("FPGA_LISTEN_PORT")
	log.Println("FPGA_APP_DIR =", FPGA_APP_DIR, "FPGA_LISTEN_PORT =", FPGA_LISTEN_PORT)

	if err := tcpListen(); err != nil {
		log.Printf("exit now: %v\n", err)
	}
}
