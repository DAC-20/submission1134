package main

// backup: CFLAGS: -I${SRCDIR} -I${SRCDIR}/../../libs/xcl2 -I${SRCDIR}/../../libs/bitmap -I/opt/xilinx/xrt/include/ -I/tools/Xilinx/Vivado/2018.3/include/

// #cgo CFLAGS: -I${SRCDIR} 
// #cgo LDFLAGS: -L${SRCDIR} -lfragment -L/opt/xilinx/xrt/lib/ -lOpenCL -lpthread -lrt  -lstdc++
// #include "src/fragment_watermarking.h"
import "C"

func main() {
//   C.bar()
//   C.bar_again()
  // C.another_bar()
  C.init(C.CString("xclbin/apply_watermark.hw.xilinx_u200_xdma_201830_1.xclbin"), C.CString("data/inputImage.bmp"), C.CString("outputImage.bmp"))
  C.compute_batch(C.CString("100"))
  C.cleanup()
}