package main

import (
	"log"
	"net"
	"time"
	"encoding/json"
	"flag"
	"sync"
	"os"
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
	Time   TimePack
}

const (
	master_ip   = "192.168.202.94"
	master_port = "9999"
	// default_cmd = "./watermark xclbin/apply_watermark.hw.xilinx_u200_xdma_201830_1.xclbin data/inputImage.bmp /outputImage.bmp data/golden.bmp 100"
	default_cmd = "./vmult_slr_assign xclbin/vmult_vadd.hw.xilinx_u200_xdma_201830_1.xclbin 1"
)

type Setting struct {
	duration    float64 
	rpc         float64
	cmd         string
	stride      int
	interval    float64
}

func tcpDail(ip string, port string) (string, error) {
	response := Response{}
	response.Time.Get = time.Now()
	conn, err := net.Dial("tcp", ip + ":" + port)
	if err != nil {
		log.Println("tcp dial error:", err)
		return "", err
	}
	defer conn.Close()
	request := Request{}
	request.Input = "call"
	jsonreq, err := json.Marshal(request)
	if err != nil {
		log.Fatalln(err)
	}
	// log.Println("send request json:", string(jsonreq))
	log.Println("send request json:", string(jsonreq))
	response.Time.Issue = time.Now()
	// conn.Write([]byte("call"))
	conn.Write(jsonreq)
	data := make([]byte, 10000)
	conn.Read(data)
	response.Time.Finish = time.Now()
	log.Println("get response json:", string(data))
	i := 0
	for i = range data {
		if data[i] == 0 {
			break
		}
	}
	clientres := Response{}
	if err := json.Unmarshal(data[0:i], &clientres); err != nil {
		log.Fatalln(err)
	}
	// log.Println("response json:", string(data))
	response.Time.Next = new(TimePack)
	*(response.Time.Next) = clientres.Time
	response.Output = clientres.Output
	response.Time.Response = time.Now()
	jsonres, err := json.Marshal(response)
	if err != nil {
		log.Fatalln(err)
	}
	// log.Println("final response json:", jsonres)
	return string(jsonres), nil
}

func testCall(cnt int, wg *sync.WaitGroup) {
	log.Println(cnt)
	wg.Done()
}


func Call(ip string, port string, id int, wg *sync.WaitGroup, setting *Setting) {
	defer wg.Done()
	response := Response{}
	response.Time.Get = time.Now()
	conn, err := net.Dial("tcp", ip + ":" + port)
	if err != nil {
		log.Println(id, "tcp dial error:", err)
		// return
		os.Exit(1)
	}
	defer conn.Close()
	request := Request{}
	// request.Input = "call"
	request.Input = setting.cmd
	jsonreq, err := json.Marshal(request)
	if err != nil {
		log.Println(id, "json marshal error:", err)
		return
	}
	log.Println(id, "send request json:", string(jsonreq))
	response.Time.Issue = time.Now()
	// conn.Write([]byte("call"))
	conn.Write(jsonreq)
	data := make([]byte, 10000)
	conn.Read(data)
	response.Time.Finish = time.Now()
	i := 0
	for i = range data {
		if data[i] == 0 {
			break
		}
	}
	clientres := Response{}
	if err := json.Unmarshal(data[0:i], &clientres); err != nil {
		log.Println(id, "json unmarshal error:", err)
		return
	}
	// log.Println("response json:", string(data))
	response.Time.Next = new(TimePack)
	*(response.Time.Next) = clientres.Time
	response.Output = clientres.Output
	response.Time.Response = time.Now()
	log.Println(id, "get response, issue", response.Time.Finish.Sub(response.Time.Issue), "scheduler", response.Time.Next.Response.Sub(response.Time.Next.Get), "scheduler issue", response.Time.Next.Finish.Sub(response.Time.Issue), "function", response.Time.Next.Next.Response.Sub(response.Time.Next.Next.Get))
	// jsonres, err := json.Marshal(response)
	// if err != nil {
	// 	log.Println(id, "json unmarshal error:", err)
	// 	return
	// }
	// log.Println("final response json:", jsonres)
	return
}

func main() {
	setting := Setting{}
	flag.Float64Var(&setting.duration, "d", 60, "duration of the calling(in second)")
	flag.Float64Var(&setting.rpc, "r", 1, "request per second")
	flag.StringVar(&setting.cmd, "c", default_cmd, "cmd to be executed by the call")
	flag.IntVar(&setting.stride, "s", 0, "stride of rpc increase")
	flag.Float64Var(&setting.interval, "i", 1, "interval of a stride")
	flag.Parse()
	// log.Println(setting.duration, setting.rpc, setting.cmd)
	var wg sync.WaitGroup
	if setting.stride != 0 {
		initRpc := int(setting.rpc) % int(setting.stride)
		if initRpc == 0 {
			initRpc = setting.stride
		}
		for strideRpc := initRpc; strideRpc < int(setting.rpc); strideRpc += setting.stride {
			round := int(setting.interval) * strideRpc
			interval := time.Second / time.Duration(strideRpc)
			for i := 0; i < round; i++ {
				wg.Add(1)
				go Call(master_ip, master_port, i, &wg, &setting)
				time.Sleep(interval)
			}
		}
	}
	
	round := int(setting.duration * setting.rpc)
	interval := time.Second / time.Duration(setting.rpc) 
	issueBeginTime := time.Now()
	// var wg sync.WaitGroup
	for i := 0; i < round; i++ {
		wg.Add(1)
		go Call(master_ip, master_port, i, &wg, &setting)
		time.Sleep(interval)
	}
	issueEndTime := time.Now()
	wg.Wait()
	log.Printf("issue interval %v, average issue interval %v\n", issueEndTime.Sub(issueBeginTime), issueEndTime.Sub(issueBeginTime) / (time.Duration)(round))
	// msg, err := tcpDail(master_ip, master_port)
	// if err != nil {
	// 	log.Println("call failed:", err)
	// } else {
	// 	log.Println("response:", msg)
	// }
}
