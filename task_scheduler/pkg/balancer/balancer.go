package balancer

import (
	"log"
	"time"
	"strconv"
	"net"
	"sync"
	"fmt"
	"strings"
	"encoding/json"
	"sort"

	"task_scheduler/pkg/worker"
	"task_scheduler/pkg/utils"
)

type LoadBalancer struct {
	batch           int
	batchLatency    time.Duration
	accept          int
	done            int
	mut             *sync.Mutex
	workerSet       *worker.WorkerSet

	lastFlushCount  int
	flushChanged    bool

	recorder        *utils.Recorder
	responses       []*utils.Response
	resMut          *sync.Mutex
	recordInterval  time.Duration
}

func NewLoadBalancer(workerSet *worker.WorkerSet) *LoadBalancer {
	m := &LoadBalancer{
		batch:              utils.Batch,
		batchLatency:       utils.BatchLatency,
		accept:             0,
		done:               0,
		mut:                new(sync.Mutex),
		workerSet:          workerSet,
		
		recorder:           utils.NewRecorder(utils.LatencyRecordFilename),
		// responses:          make([]*utils.Response),
		resMut:             new(sync.Mutex),
		recordInterval:     utils.LatencyRecordInterval,
	}
	return m
}

func (m *LoadBalancer) Serve() error {
	// tcp_listen_port, _ := strconv.Atoi(utils.BalancerPort)
	go m.recordLatency()
	tcp_listen_port := utils.BalancerPort
	l, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: tcp_listen_port,
	})
	if err != nil {
		log.Println("listen error:", err)
		return err
	}
	defer l.Close()
	lastBatchEndTime := time.Now()
	for {
		var conns    []net.Conn
		var getTimes []time.Time
		var reqs []*utils.Request
		batchBeginTime := time.Now()
		l.SetDeadline(batchBeginTime.Add(m.batchLatency))
		for i := 0; i < m.batch; i++ {
			conn, err := l.Accept()
			if err != nil {
				if opError, ok := err.(*net.OpError); ok && opError.Timeout() {
					break
				} else {
					log.Println("warning: accept error:", err)
					continue
				}
			}
			data := make([]byte, 10000)
			conn.Read(data)
			getTime := time.Now()
			dataTail := 0
			for dataTail = range data {
				if data[dataTail] == 0 {
					break
				}
			}
			var req utils.Request
			if err := json.Unmarshal(data[0:dataTail], &req); err != nil {
				log.Println("warning: unmarshalling data error:", string(data[0:dataTail]), err)
				continue
			}
			conns = append(conns, conn)
			getTimes = append(getTimes, getTime)
			reqs = append(reqs, &req)
			m.accept += 1
		}
		batchEndTime := time.Now()
		batchsize := len(conns)
		log.Printf("trace: accept batchsize %d, %d in total, batch time %v, batch interval %v, batch distance %v\n", batchsize, m.accept, batchEndTime.Sub(batchBeginTime), batchBeginTime.Sub(lastBatchEndTime), batchEndTime.Sub(lastBatchEndTime))
		lastBatchEndTime = batchEndTime
		if batchsize <= 0 {
			continue
		}
		go m.handleBatchConns(conns, reqs, getTimes)
	}
}

func (m *LoadBalancer) handleBatchConns(conns []net.Conn, reqs []*utils.Request, getTimes []time.Time) error {
	defer utils.CloseConns(conns)
	batchsize := len(conns)
	workerName, workerIp, workerPort, err := m.distributeWorker(batchsize)
	if err != nil {
		return err
	}
	req := reqs[0]
	adjustRequest(req, batchsize)
	jsonreq, err := json.Marshal(req)
	if err != nil {
		log.Println(err)
		return err
	}
	issueTime := time.Now()
	data, err := utils.TcpDial(workerIp, workerPort, jsonreq)
	if err != nil {
		log.Printf("warning: %v\n", err)
		return err
	}
	finishTime := time.Now()
	clientres := utils.Response{}
	if err := json.Unmarshal(data, &clientres); err != nil {
		log.Printf("warning: %v\n", err)
		return err
	}
	go m.finishWorker(workerName, batchsize, clientres.Time.Response.Sub(clientres.Time.Get))
	for i := 0; i < batchsize; i++ {
		res := utils.Response{}
		res.Time.Get = getTimes[i]
		res.Time.Issue = issueTime
		res.Time.Finish = finishTime
		res.Time.Next = new(utils.TimePack)
		*(res.Time.Next) = clientres.Time
		res.Output = clientres.Output
		res.Time.Response = time.Now()
		jres, err := json.Marshal(res)
		if err != nil {
			log.Println(err)
			return err
		}
		conns[i].Write(jres)
		// go m.profiler.Append(req, &res)
		if i == 0 {
			log.Println("trace: batch[0] json response:", string(jres))
		}
		m.resMut.Lock()
		m.responses = append(m.responses, &res)
		m.resMut.Unlock()
	}
	m.mut.Lock()
	m.done += batchsize
	log.Printf("trace: done batchsize %d, %d in total\n", batchsize, m.done)
	m.mut.Unlock()
	return nil
}

func (m *LoadBalancer) distributeWorker(batchsize int) (string, string, string, error) {
	targetWorkerName := ""
	targetWorkerIp := ""
	targetWorkerPort := ""
	m.workerSet.Mut.Lock()
	workers := m.workerSet.Workers
	for workerName, _ := range workers {
		if !workers[workerName].Online {
			continue
		}
		if workers[workerName].Weight > 0 {
			workers[workerName].Weight -= 1
			workers[workerName].Workload += 1
			workers[workerName].WorkCount += 1
			workers[workerName].AccBatch += batchsize 
			// return workers[workerName].Name, workers[workerName].Ip, workers[workerName].Port
			targetWorkerName = workers[workerName].Name
			targetWorkerIp = workers[workerName].Ip
			targetWorkerPort = workers[workerName].Port
			break
		}
	}
	if targetWorkerName != "" {
		m.workerSet.Request += batchsize
		m.workerSet.Mut.Unlock()
		if m.flushChanged {
			log.Printf("debug: workload distribute to worker %s(ip %s, port %s)", targetWorkerName, targetWorkerIp, targetWorkerPort)
		}
		return targetWorkerName, targetWorkerIp, targetWorkerPort, nil
	} else {
		m.workerSet.Mut.Unlock()
		if flushCount := m.flushWorkerSetWeight(); flushCount <= 0 {
			log.Printf("error: no online worker available to distribute workload")
			return "", "", "", fmt.Errorf("no online worker available to distribute workload")
		}
		return m.distributeWorker(batchsize)
	}
}

func (m *LoadBalancer) finishWorker(workerName string, batchsize int, latency time.Duration) {
	m.workerSet.Mut.Lock()
	m.workerSet.Throughput += batchsize
	workers := m.workerSet.Workers
	workers[workerName].Workload -= 1
	workers[workerName].LateLatency = latency
	workers[workerName].AvegLatency = (workers[workerName].AvegLatency * 3 + latency) / 4
	m.workerSet.Mut.Unlock()
}

func (m *LoadBalancer) flushWorkerSetWeight() int {
	flushCount := 0
	m.workerSet.Mut.Lock()
	defer m.workerSet.Mut.Unlock()
	workers := m.workerSet.Workers
	weightCount := 0
	for workerName, _ := range workers {
		if !workers[workerName].Online {
			continue
		}
		if workers[workerName].Weight <= 0 {
			workers[workerName].Weight = workers[workerName].MaxWeight
			weightCount += workers[workerName].MaxWeight
			flushCount += 1
		}
	}
	if flushCount != m.lastFlushCount {
		log.Printf("debug: flush %d workers' weights, increased %d", flushCount, weightCount)
		m.flushChanged = true
		m.lastFlushCount = flushCount
	} else {
		m.flushChanged = false
	}
	return flushCount
}

func adjustRequest(req *utils.Request, batch int) {
	args := strings.Fields(req.Input)
	args[len(args)-1] = strconv.Itoa(batch)
	req.Input = ""
	for _, arg := range args {
		req.Input += arg + " "
	}
}

type durationSlice []time.Duration 
func (p durationSlice) Len() int {
	return len(p)
} 
func (p durationSlice) Less(i, j int) bool {
	return p[i] < p[j]
}
func (p durationSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (m *LoadBalancer) recordLatency() {
	lastAccept := 0
	for {
		time.Sleep(m.recordInterval)
		m.resMut.Lock()
		responses := m.responses
		m.responses = nil
		m.resMut.Unlock()

		var latencies durationSlice
		end := len(responses)
		for i := 0; i < end; i++ {
			latencies = append(latencies, responses[i].Time.Response.Sub(
				responses[i].Time.Get))
		}
		sort.Sort(latencies)
		div95 := (int)((float64)(len(latencies)) * 0.95)
		div99 := (int)((float64)(len(latencies)) * 0.99)
		if div95 >= len(latencies) || div99 >= len(latencies) {
			continue
		} else {
			m.mut.Lock()
			accept := m.accept
			m.mut.Unlock()
			acceptDiff := accept - lastAccept
			lastAccept = accept
			log.Printf("debug: average request intensity %f, average response intensity %f, div95: %v, div99: %v", float64(acceptDiff) / float64(m.recordInterval / time.Second), float64(len(latencies)) / float64(m.recordInterval / time.Second), latencies[div95], latencies[div99])
			m.recorder.AddPrivate("div95", strconv.FormatFloat(latencies[div95].Seconds(), 'f', -1, 32))
			m.recorder.AddPrivate("div99", strconv.FormatFloat(latencies[div99].Seconds(), 'f', -1, 32))
			m.recorder.End()
		}
	}
}
