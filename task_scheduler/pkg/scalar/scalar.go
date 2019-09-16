package scalar

import (
	"time"
	"log"
	"fmt"

	"task_scheduler/pkg/worker"
	"task_scheduler/pkg/utils"
)

type WorkerScalar struct {
	workerSet       *worker.WorkerSet
	interval        time.Duration
	lastIntensity   float64
	issuedCapacitys map[string]float64
	issueLatency    time.Duration

	recorder        *utils.Recorder
	recordPeriod    float64
}

func NewWorkerScalar(workerSet *worker.WorkerSet) *WorkerScalar {
	m := &WorkerScalar{
		workerSet:          workerSet,
		interval:           1000 * time.Millisecond,
		lastIntensity:      0,
		issuedCapacitys:    make(map[string]float64),
		issueLatency:       6000 * time.Millisecond,
		
		recorder:           utils.NewRecorder(utils.IntensityRecordFilename),
		recordPeriod:       0,
	}
	return m
}

func (m *WorkerScalar) Watch() {
	for {
		time.Sleep(m.interval)
		m.decideScaling()
	}
}

func (m *WorkerScalar) decideScaling() {
	m.workerSet.Mut.Lock()

	workers := m.workerSet.Workers
	onlineCount := 0
	accLatency := time.Millisecond * 0
	accBatch := 0
	var saturateCapacity float64
	saturated := false
	needLog := false
	for workerName, _ := range workers {
		if workers[workerName].Online {
			onlineCount += 1
			accLatency += workers[workerName].AvegLatency
			if workers[workerName].WorkCount > 0 {
				averageBatch := workers[workerName].AccBatch / workers[workerName].WorkCount
				accBatch += averageBatch
				if float64(averageBatch) / float64(workers[workerName].MaxBatch) >= 0.90 {
					saturated = true
				}
			} else {
				accBatch += workers[workerName].MaxBatch
			}
			workers[workerName].WorkCount = 0
			workers[workerName].AccBatch = 0
			saturateCapacity += workers[workerName].Capacity
			if _, exist := m.issuedCapacitys[workerName]; exist {
				delete(m.issuedCapacitys, workerName)
				needLog = true
			}
		}
	}
	request := m.workerSet.Request
	m.workerSet.Request = 0
	m.workerSet.Throughput = 0
	m.workerSet.Mut.Unlock()
	// var avegLatency time.Duration
	var capacity float64
	if onlineCount <= 0 {
		// avegLatency = 0
		capacity = 0
	} else {
		// saturateCapacity /= float64(onlineCount)
		// if capacity < saturateCapacity {
		// 	capacity = saturateCapacity
		// }
		if !saturated {
			capacity = saturateCapacity
		} else {
			avegLatency := accLatency / time.Duration(onlineCount)
			capacity = float64(accBatch) / (float64(avegLatency) / float64(time.Second))
		}
	}
	
	thisIntensity := float64(request) / (float64(m.interval) / float64(time.Second))
	diffIntensitySpeed := (thisIntensity - m.lastIntensity) / (float64(m.interval) / float64(time.Second))
	diffIntensity := diffIntensitySpeed * (float64(m.issueLatency) / float64(time.Second))
	nextIntensity := thisIntensity + diffIntensity
	// use saturateCapacity to estimate all the workers'(issued or to be issued) capacity; must be changed for hetergeneous workers; not implemented yet...
	issuedCapacity := capacity 
	for _, workerCapacity := range m.issuedCapacitys {
		issuedCapacity += workerCapacity
	}
	m.lastIntensity = thisIntensity
	if thisIntensity > 0 {
		needLog = true
	}
	scaleCount := 0
	scaledCapacity := issuedCapacity
	for {
		if scaledCapacity < nextIntensity {
			workerName, workerCapacity, err := m.Scale()
			if err == nil {
				m.issuedCapacitys[workerName] = workerCapacity
				scaledCapacity += workerCapacity
				scaleCount += 1
			} else {
				log.Printf("error: scale failed err: %v", err)
				return
			}
			needLog = true
		} else {
			break
		}
	}
	shrinkCount := 0
	if scaleCount == 0 {
		avegCapacity := capacity / float64(onlineCount)
		if (diffIntensity <= 0) && (capacity - avegCapacity > thisIntensity) {
			m.Shrink()
			shrinkCount += 1
			needLog = true
		}
	}
	needLog = true
	m.recordPeriod += float64(m.interval) / float64(time.Second)
	m.recorder.AddPublic("second", fmt.Sprintf("%.2f", m.recordPeriod))
	m.recorder.AddPublic("capacity", fmt.Sprintf("%.2f", capacity))
	m.recorder.AddPublic("issuedCapacity", fmt.Sprintf("%.2f", issuedCapacity))
	m.recorder.AddPublic("intensity", fmt.Sprintf("%.2f", thisIntensity))
	m.recorder.End()
	if needLog {
		log.Printf("debug: %d online workers, capacity %f/s, issued capacity %f/s, current intensity %f/s, increased intensity speed %f/s/s, estimated next intensity %f/s, scale %d, shrink %d", onlineCount, capacity, issuedCapacity, thisIntensity, diffIntensitySpeed, nextIntensity, scaleCount, shrinkCount)
	}
}

func (m *WorkerScalar) Scale() (string, float64, error) {
	log.Printf("info: scale by 1")
	deviceFrac := 1
	return m.workerSet.AddWorker(deviceFrac)
}

func (m *WorkerScalar) Shrink() {
	m.workerSet.Mut.Lock()
	var workerName string
	for key, _ := range m.workerSet.Workers {
		workerName = key
		break
	}
	m.workerSet.Mut.Unlock()
	log.Printf("info: shrink by 1")
	m.workerSet.DeleteWorker(workerName)
}