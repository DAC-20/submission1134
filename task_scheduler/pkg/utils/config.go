package utils

import (
	"time"
)

var (
	SamplePodFile string = "./vmult-slr-assign-client-pod.yaml"

	Batch int = 15
	BatchLatency time.Duration = 70 * time.Millisecond
	InitMaxWeight int = 3

	LatencyRecordFilename string = "latency.rec"
	IntensityRecordFilename string = "intensity.rec"
	WorkerNum int = 1
)