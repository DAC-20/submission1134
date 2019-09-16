package utils

import (
	"time"
)

const (
	DeviceName   = "xilinx.com/fpga-xilinx_u200_xdma_201830_1-1542252769"
	// ResourceName = "xilinx.com/fpga-xilinx_u200_xdma_201830_1-1542252769-fraction"
	ResourceName = DeviceName
	
	ClientContainerName  = "go-client-fpga"
	ClientListenPortName = "fpga-listen"
	ClientListenPortEnv  = "FPGA_LISTEN_PORT"
	
	Node1 = "u200-1"
	Node2 = "u200-2"
	Ip1   = "192.168.202.94"
	Ip2   = "192.168.202.95"
	ListenPortBegin = 9898
	ListenPortEnd   = 10100

	BalancerPort = 9999

	ResyncPeriod = time.Second * 30

	LatencyRecordInterval = time.Second * 5
)
