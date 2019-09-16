package worker

import (
	// "sync"
	"log"
	"strconv"
	"time"

	"k8s.io/api/core/v1"

	"task_scheduler/pkg/utils"
)

type TaskWorker struct {
	Name            string
	Pod             *v1.Pod
	Ip              string
	Port            string
	DeviceFrac      int
	Online          bool

	Workload        int
	WorkCount       int
	LateLatency     time.Duration
	AvegLatency     time.Duration
	MaxBatch        int
	AccBatch        int
	BatchLatency    time.Duration
	// mut         *sync.Mutex

	MaxWeight       int
	Weight          int
	Capacity        float64
}

func newTaskWorker(name string, pod *v1.Pod) *TaskWorker {
	m := &TaskWorker{
		Name:        name,
		Workload:    0,
		MaxWeight:   utils.InitMaxWeight,
		
		BatchLatency:utils.BatchLatency,
	}
	// to avoid div 0 problem, give initial non-zero profiling values to the worker
	// m.WorkCount = 1
	// m.AccBatch = m.MaxBatch
	m.Weight = m.MaxWeight
	m.MaxBatch = utils.Batch * m.MaxWeight
	m.LateLatency = m.BatchLatency
	m.AvegLatency = m.BatchLatency

	m.Capacity = float64(m.MaxBatch) / (float64(m.BatchLatency) / float64(time.Second))
	m.updatePod(pod)
	return m
}

func (m *TaskWorker) updatePod(pod *v1.Pod) {
	m.Pod = pod
	m.Port = strconv.Itoa(utils.GetPodListenPort(pod))
	m.DeviceFrac = utils.GetPodDeviceFraction(pod)
	switch pod.Spec.NodeName {
	case utils.Node1:
		m.Ip = utils.Ip1
	case utils.Node2:
		m.Ip = utils.Ip2
	default:
		m.Online = false
		log.Printf("info: worker %s is not assigned to any known node, offline", m.Name)
		return
	}
	if pod.Status.Phase == "Running" {
		if !m.Online {
			go func() {
				for {
					if utils.IsPortOccupied(m.Ip, utils.GetPodListenPort(pod)) {
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
				m.Online = true
				log.Printf("info: worker %s is online", m.Name)
			}()
		} else {
			m.Online = true
			log.Printf("info: worker %s stays online", m.Name)
		}
	} else {
		m.Online = false
		log.Printf("info: worker %s is offline", m.Name)
	}
}