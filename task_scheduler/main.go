package main

import (
	"log"
	"flag"
	"os/signal"
	"syscall"
	"os"

	"task_scheduler/pkg/utils"
	"task_scheduler/pkg/worker"
	"task_scheduler/pkg/scalar"
	"task_scheduler/pkg/balancer"
)

func main() {
	flag.IntVar(&utils.WorkerNum, "w", 1, "worker number")
	flag.StringVar(&utils.LatencyRecordFilename, "l", "latency.rec", "latency record filename")
	flag.StringVar(&utils.IntensityRecordFilename, "i", "intensity.rec", "intensity record filename")
	flag.Parse()

	utils.InitLog("debug")
	clientset, err := utils.InitClient()
	if err != nil {
		log.Fatal(err)
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	
	manager, err := worker.NewWorkerSetManager(clientset, stopCh)
	if err != nil {
		log.Fatal(err)
	}
	workerSet := manager.WorkerSet()
	loadBalancer := balancer.NewLoadBalancer(workerSet)
	workerScalar := scalar.NewWorkerScalar(workerSet)
	
	for i := 0; i < utils.WorkerNum; i++ {
		workerScalar.Scale()
	}

	go loadBalancer.Serve()
	go workerScalar.Watch()

	// <-sig
	// if (workerNum < 3) {
	// 	workerScalar.Scale()
	// } else {
	// 	workerScalar.Shrink()
	// }
	<-sig
	log.Println("info: receive signal, cleaning up")
	workerSet.CleanUp()
	os.Exit(0)
}