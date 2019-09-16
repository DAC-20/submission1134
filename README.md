## Dependencies
### Software
- kubernetes 1.14 (with Xilinx FPGA device plugin enabled)
- Xilinx SDAccel toolchain (version 2018.3)
- go 1.10, and kubenetes related packages
- python 3.7, and prettyTable package

### Hardware
- Xilinx U200 FPGA x6 (or more)
- FPGA host x86\_64 machine x2 (or more)

## Directory contents
task\_scheduler/: this folder contains the major framework components, such as the task scheduler (balancer & scalar), container manager, etc.

caller/: this folder contains the go script for issuing requests to the task\_scheduler to evaluate its performance.

fpga\_app/: this folder contains some fpga applications (functions) to be managed and scheduled by our framework.

## Usage
### Setup
1. set the related configuration in task\_scheduler/pkg/utils/const.go, such as the ip address of the FPGA machines, the master port, etc.
2. copy the fpga\_app/ directory to some consistent path in all the kubernetes FPGA worker machines, then modify the mount path in task\_scheduler/sample-client-pod.yaml to your specified path.
3. run "go build" in the task\_sheduler/ and caller/ directory to compile and get the binaries.

### Run
1. go to the task\_scheduler/ directory and run "./task\_scheduler" for default setting.
To get the detailed setting usage description, run "./task\_scheduler -h".
This binary will initialize our framework and wait for function requests on the master port.
Note that this binary won't stop running until it receives signals like SIGUP, SIGTERM or SIGINT, or please run this binary in a seperate terminal session.
2. go to the caller/ directory and run "./caller" for default setting. 
To get the detailed setting usage description, run "./caller -h".
This binary will issue requests (with specified request-per-second and change speed) to the master port.
3. send SIGTERM or SIGINT signal to the task\_scheduler process started in step 1 to shutdown the framework.


If you want to do more thourough profiling, you can instead go to task\_scheduler/ directory and run "./profile.sh" which will automatically take care of the above procedures and do the evaluation continuously.
