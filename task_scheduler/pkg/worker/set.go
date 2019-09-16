package worker

import (
	"sync"
	"log"
	"time"
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"task_scheduler/pkg/utils"
)

type WorkerSet struct {
	Workers         map[string]*TaskWorker
	Mut             *sync.RWMutex
	Throughput      int
	Request         int

	workerCounter   int
	clientset       *kubernetes.Clientset
	samplePod       *v1.Pod
	
	ports           []int
	portsMut        *sync.Mutex
}

func newWorkerSet(clientset *kubernetes.Clientset) (*WorkerSet, error) {
	var err error
	m := &WorkerSet{
		Workers:        make(map[string]*TaskWorker),
		Mut:            new(sync.RWMutex),
		Throughput:     0,
		Request:        0,
		workerCounter:  0,
		clientset:      clientset,
		portsMut:       new(sync.Mutex),
	}
	m.flushPortPool()
	if m.samplePod, err = utils.YamlToPod(utils.SamplePodFile); err != nil {
		return nil, err
	}
	
	return m, nil
}

func (m *WorkerSet) flushPortPool() {
	portCounter := 0
	m.portsMut.Lock()
	for i := utils.ListenPortBegin; i <= utils.ListenPortEnd; i++ {
		if !( utils.IsPortOccupied(utils.Ip1, i) || utils.IsPortOccupied(utils.Ip2, i) ) {
			// m.portPool.Put(i)
			m.ports = append(m.ports, i)
			portCounter += 1
			log.Printf("trace: add %dth port (%d) to portPool", portCounter, i)
		}
	}
	log.Printf("debug: workerset adds %d ports to available port pool", portCounter)
	m.portsMut.Unlock()
}

func (m *WorkerSet) AddWorker(deviceFrac int) (string, float64, error) {
	var port int
	m.portsMut.Lock()
	if len(m.ports) <= 0 {
		log.Printf("error: no port left for future worker")
		m.portsMut.Unlock()
		return "", 0, fmt.Errorf("no port left for future worker")
	} else {
		port = m.ports[len(m.ports) - 1]
		m.ports = m.ports[0 : len(m.ports) - 1]
		log.Printf("debug: pod get port %d", port)
		m.portsMut.Unlock()
	}
	pod := m.samplePod.DeepCopy()
	m.Mut.Lock()
	m.workerCounter += 1
	workerName := fmt.Sprintf("worker-%d", m.workerCounter)
	m.Mut.Unlock()
	pod.Name = workerName
	utils.SetPodListenPort(pod, port)
	utils.SetPodDeviceFraction(pod, deviceFrac)
	if err := utils.ApplyPod(m.clientset, pod); err != nil {
		// should decide whether to return the port to the pool or not... not implemented
		log.Printf("warning: apply pod %s error: %v", pod.Name, err)
		return "", 0, err
	}
	m.Mut.Lock()
	m.Workers[workerName] = newTaskWorker(workerName, pod)
	workerCapacity := m.Workers[workerName].Capacity
	m.Mut.Unlock()
	log.Printf("debug: pod %s applied, port %d", pod.Name, port)
	return workerName, workerCapacity, nil
}

func (m *WorkerSet) DeleteWorker(workerName string) {
	m.Mut.Lock()
	log.Printf("debug: begin to delete worker %s", workerName)
	taskWorker, exist := m.Workers[workerName]
	if !exist {
		log.Printf("warn: worker %s doesn't exist, deletion failed", workerName)
		return
	}
	m.Workers[workerName].Online = false
	m.Mut.Unlock()
	go func() {
		// wait 2 seconds to drain the worker
		for i := 0; i < 20; i++ {
			if taskWorker.Workload <= 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if taskWorker.Workload <= 0 {
			log.Printf("debug: worker %s is drained", taskWorker.Name)
		} else {
			log.Printf("debug: draining worker %s failed, remaining workload %d, force to exit", taskWorker.Name, taskWorker.Workload)
		}
		utils.DeletePod(m.clientset, taskWorker.Pod)
	}()
}

func (m *WorkerSet) updateWorkerPod(pod *v1.Pod) error {
	m.Mut.Lock()
	defer m.Mut.Unlock()
	taskWorker, exist := m.Workers[pod.Name]
	if !exist {
		log.Printf("error: update worker pod %s doesn't exist in the workerset", pod.Name)
		return fmt.Errorf("update worker pod %s doesn't exist in the workerset", pod.Name)
	}
	taskWorker.updatePod(pod)
	return nil
}

func (m *WorkerSet) removeWorkerPod(pod *v1.Pod) {
	m.Mut.Lock()
	delete(m.Workers, pod.Name)
	log.Printf("debug: workerPod %s is deleted", pod.Name)
	m.Mut.Unlock()
	if port := utils.GetPodListenPort(pod); port != 0 {
		m.portsMut.Lock()
		m.ports = append(m.ports, port)
		m.portsMut.Unlock()
	}
}

func (m *WorkerSet) CleanUp() {
	m.Mut.Lock()
	for workerName, _ := range m.Workers {
		m.Workers[workerName].Online = false
		utils.DeletePod(m.clientset, m.Workers[workerName].Pod)
		delete(m.Workers, workerName)
	}
	m.Mut.Unlock()
}