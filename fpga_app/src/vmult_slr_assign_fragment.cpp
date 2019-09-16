/**********
Copyright (c) 2018, Xilinx, Inc.
All rights reserved.

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice,
this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO,
THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE,
EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
**********/
#include <iostream>
#include <sstream>
#include <string>
#include "xcl2.hpp"
#include <algorithm>
#include <vector>
#include "vmult_slr_assign_fragment.h"
// #define DATA_SIZE 4096
// #define MAX_BATCH_SIZE 60
int data_size = 4096;
int max_batch_size = 60;
cl_int err;
char* fileBuf;
cl::CommandQueue *p_q_slr0;
cl::CommandQueue *p_q_slr1;
cl::CommandQueue *p_q_slr2;
cl::Context* p_context;
cl::Kernel* p_vector_mult_slr0;
cl::Kernel* p_vector_mult_slr1;
cl::Kernel* p_vector_mult_slr2;
std::vector<int, aligned_allocator<int>>* p_A_slr0;
std::vector<int, aligned_allocator<int>>* p_B_slr0;
std::vector<int, aligned_allocator<int>>* p_C_slr0;

std::vector<int, aligned_allocator<int>>* p_A_slr1;
std::vector<int, aligned_allocator<int>>* p_B_slr1;
std::vector<int, aligned_allocator<int>>* p_C_slr1;

std::vector<int, aligned_allocator<int>>* p_A_slr2;
std::vector<int, aligned_allocator<int>>* p_B_slr2;
std::vector<int, aligned_allocator<int>>* p_C_slr2;


int init(char* argv1, char* size, char* max_batch) {
    std::string binaryFile = argv1;
    // get the data_size
    std::string data_size_text = size;
    std::istringstream iss (data_size_text);
    iss >> data_size;
    if (iss.fail()) {
        return EXIT_FAILURE;
    } else {
        std::cout << "Data_Size = " << data_size << std::endl;
    }
    // get the max_batch_size
    std::string max_batch_size_text = max_batch;
    std::istringstream iss2 (max_batch_size_text);
    iss2 >> max_batch_size;
    if (iss2.fail()) {
        return EXIT_FAILURE;
    } else {
        std::cout << "Max_Batch_Size = " << max_batch_size << std::endl;
    }

    auto devices = xcl::get_xil_devices();
    auto device = devices[0];

    // OCL_CHECK(err, cl::Context context(device, NULL, NULL, NULL, &err));
    OCL_CHECK(err, p_context = new cl::Context(device, NULL, NULL, NULL, &err));
    cl::Context& context = *p_context;
    // OCL_CHECK(
    //     err,
    //     cl::CommandQueue q(context, device, CL_QUEUE_PROFILING_ENABLE, &err));
    OCL_CHECK(err, (p_q_slr0 = new cl::CommandQueue(context, device, CL_QUEUE_PROFILING_ENABLE, &err)));
    cl::CommandQueue& q_slr0 = *p_q_slr0;
    OCL_CHECK(err, (p_q_slr1 = new cl::CommandQueue(context, device, CL_QUEUE_PROFILING_ENABLE, &err)));
    cl::CommandQueue& q_slr1 = *p_q_slr1;
    OCL_CHECK(err, (p_q_slr2 = new cl::CommandQueue(context, device, CL_QUEUE_PROFILING_ENABLE, &err)));
    cl::CommandQueue& q_slr2 = *p_q_slr2;
    OCL_CHECK(err,
            std::string device_name = device.getInfo<CL_DEVICE_NAME>(&err));

    printf("INFO: loading vmul kernel\n");
    unsigned fileBufSize;
    fileBuf = xcl::read_binary_file(binaryFile, fileBufSize);
    cl::Program::Binaries bins{{fileBuf, fileBufSize}};
    devices.resize(1);
    OCL_CHECK(err, cl::Program program(context, devices, bins, NULL, &err));
    // OCL_CHECK(err, cl::Kernel vector_mult_slr0(program, "vmult_slr0", &err));
    // OCL_CHECK(err, cl::Kernel vector_mult_slr1(program, "vmult_slr1", &err));
    // OCL_CHECK(err, cl::Kernel vector_mult_slr2(program, "vmult_slr2", &err));
    OCL_CHECK(err, (p_vector_mult_slr0 = new cl::Kernel(program, "vmult_slr0", &err)));
    cl::Kernel& vector_mult_slr0 = *p_vector_mult_slr0;
    OCL_CHECK(err, (p_vector_mult_slr1 = new cl::Kernel(program, "vmult_slr1", &err)));
    cl::Kernel& vector_mult_slr1 = *p_vector_mult_slr1;
    OCL_CHECK(err, (p_vector_mult_slr2 = new cl::Kernel(program, "vmult_slr2", &err)));
    cl::Kernel& vector_mult_slr2 = *p_vector_mult_slr2;
    
    int DATA_SIZE = data_size;
    int batch_size = max_batch_size;
    
    // slr0
    // std::vector<int, aligned_allocator<int>> A_slr0(DATA_SIZE*batch_size);
    // std::vector<int, aligned_allocator<int>> B_slr0(DATA_SIZE*batch_size);
    // std::vector<int, aligned_allocator<int>> C_slr0(DATA_SIZE*batch_size);
    p_A_slr0 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& A_slr0 = *p_A_slr0;
    p_B_slr0 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& B_slr0 = *p_B_slr0;
    p_C_slr0 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& C_slr0 = *p_C_slr0;
    // slr1
    // std::vector<int, aligned_allocator<int>> A_slr1(DATA_SIZE*batch_size);
    // std::vector<int, aligned_allocator<int>> B_slr1(DATA_SIZE*batch_size);
    // std::vector<int, aligned_allocator<int>> C_slr1(DATA_SIZE*batch_size);
    p_A_slr1 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& A_slr1 = *p_A_slr1;
    p_B_slr1 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& B_slr1 = *p_B_slr1;
    p_C_slr1 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& C_slr1 = *p_C_slr1;
    // slr2
    // std::vector<int, aligned_allocator<int>> A_slr2(DATA_SIZE*batch_size);
    // std::vector<int, aligned_allocator<int>> B_slr2(DATA_SIZE*batch_size);
    // std::vector<int, aligned_allocator<int>> C_slr2(DATA_SIZE*batch_size);
    p_A_slr2 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& A_slr2 = *p_A_slr2;
    p_B_slr2 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& B_slr2 = *p_B_slr2;
    p_C_slr2 = new std::vector<int, aligned_allocator<int>>(DATA_SIZE*batch_size);
    std::vector<int, aligned_allocator<int>>& C_slr2 = *p_C_slr2;

    // Create the test data
    // slr0
    std::generate(A_slr0.begin(), A_slr0.end(), std::rand);
    std::generate(B_slr0.begin(), B_slr0.end(), std::rand);
    // slr1
    std::generate(A_slr1.begin(), A_slr1.end(), std::rand);
    std::generate(B_slr1.begin(), B_slr1.end(), std::rand);
    // slr2
    std::generate(A_slr2.begin(), A_slr2.end(), std::rand);
    std::generate(B_slr2.begin(), B_slr2.end(), std::rand);

    return EXIT_SUCCESS;
}

int compute_batch_all(char* argv2) {
    int DATA_SIZE = data_size;
    cl::CommandQueue& q_slr0 = *p_q_slr0;
    cl::CommandQueue& q_slr1 = *p_q_slr1;
    cl::CommandQueue& q_slr2 = *p_q_slr2;
    cl::Context& context = *p_context;
    cl::Kernel& vector_mult_slr0 = *p_vector_mult_slr0;
    cl::Kernel& vector_mult_slr1 = *p_vector_mult_slr1;
    cl::Kernel& vector_mult_slr2 = *p_vector_mult_slr2;
    std::vector<int, aligned_allocator<int>>& A_slr0 = *p_A_slr0;
    std::vector<int, aligned_allocator<int>>& B_slr0 = *p_B_slr0;
    std::vector<int, aligned_allocator<int>>& C_slr0 = *p_C_slr0;
    std::vector<int, aligned_allocator<int>>& A_slr1 = *p_A_slr1;
    std::vector<int, aligned_allocator<int>>& B_slr1 = *p_B_slr1;
    std::vector<int, aligned_allocator<int>>& C_slr1 = *p_C_slr1;
    std::vector<int, aligned_allocator<int>>& A_slr2 = *p_A_slr2;
    std::vector<int, aligned_allocator<int>>& B_slr2 = *p_B_slr2;
    std::vector<int, aligned_allocator<int>>& C_slr2 = *p_C_slr2;
    // get the batch_size
    int batch_size = 0;
    std::string batch_size_text = argv2;
    std::istringstream iss (batch_size_text);
    iss >> batch_size;
    if (iss.fail()) {
        return EXIT_FAILURE;
    } else {
        std::cout << "Batch_Size = " << batch_size << std::endl;
    }
    if (batch_size > max_batch_size) {
        std::cout << "error: allowed max_batch_size = " << max_batch_size << std::endl;
        return EXIT_FAILURE;
    }
    int size = DATA_SIZE*batch_size;
    size_t vector_size_bytes = sizeof(int) * DATA_SIZE*batch_size;

    // slr0 FPGA buffers
    OCL_CHECK(err,
                cl::Buffer buffer_in1_slr0(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    A_slr0.data(),
                                    &err));
    OCL_CHECK(err,
                cl::Buffer buffer_in2_slr0(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    B_slr0.data(),
                                    &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_mul_out_slr0(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_WRITE,
                                    vector_size_bytes,
                                    C_slr0.data(),
                                    &err));
    // slr1 FPGA buffers
    OCL_CHECK(err,
                cl::Buffer buffer_in1_slr1(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    A_slr1.data(),
                                    &err));
    OCL_CHECK(err,
                cl::Buffer buffer_in2_slr1(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    B_slr1.data(),
                                    &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_mul_out_slr1(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_WRITE,
                                    vector_size_bytes,
                                    C_slr1.data(),
                                    &err));

    // slr2 FPGA buffers
    OCL_CHECK(err,
                cl::Buffer buffer_in1_slr2(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    A_slr2.data(),
                                    &err));
    OCL_CHECK(err,
                cl::Buffer buffer_in2_slr2(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    B_slr2.data(),
                                    &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_mul_out_slr2(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_WRITE,
                                    vector_size_bytes,
                                    C_slr2.data(),
                                    &err));

    // kernel argeument slr0
    OCL_CHECK(err, err = vector_mult_slr0.setArg(0, buffer_in1_slr0));
    OCL_CHECK(err, err = vector_mult_slr0.setArg(1, buffer_in2_slr0));
    OCL_CHECK(err, err = vector_mult_slr0.setArg(2, buffer_mul_out_slr0));
    OCL_CHECK(err, err = vector_mult_slr0.setArg(3, size));

    // kernel argeument slr1
    OCL_CHECK(err, err = vector_mult_slr1.setArg(0, buffer_in1_slr1));
    OCL_CHECK(err, err = vector_mult_slr1.setArg(1, buffer_in2_slr1));
    OCL_CHECK(err, err = vector_mult_slr1.setArg(2, buffer_mul_out_slr1));
    OCL_CHECK(err, err = vector_mult_slr1.setArg(3, size));

    // kernel argeument slr2
    OCL_CHECK(err, err = vector_mult_slr2.setArg(0, buffer_in1_slr2));
    OCL_CHECK(err, err = vector_mult_slr2.setArg(1, buffer_in2_slr2));
    OCL_CHECK(err, err = vector_mult_slr2.setArg(2, buffer_mul_out_slr2));
    OCL_CHECK(err, err = vector_mult_slr2.setArg(3, size));

    // Copy input data to device global memory
    OCL_CHECK(err, err = q_slr0.enqueueMigrateMemObjects({buffer_in1_slr0, buffer_in2_slr0}, 0 /* 0 means from host*/));
    OCL_CHECK(err, err = q_slr1.enqueueMigrateMemObjects({buffer_in1_slr1, buffer_in2_slr1}, 0 /* 0 means from host*/));
    OCL_CHECK(err, err = q_slr2.enqueueMigrateMemObjects({buffer_in1_slr2, buffer_in2_slr2}, 0 /* 0 means from host*/));

    // Launch the Kernel
    OCL_CHECK(err, err = q_slr0.enqueueTask(vector_mult_slr0));
    OCL_CHECK(err, err = q_slr1.enqueueTask(vector_mult_slr1));
    OCL_CHECK(err, err = q_slr2.enqueueTask(vector_mult_slr2));

    OCL_CHECK(err, err = q_slr0.enqueueMigrateMemObjects({buffer_mul_out_slr0}, CL_MIGRATE_MEM_OBJECT_HOST));
    OCL_CHECK(err, err = q_slr1.enqueueMigrateMemObjects({buffer_mul_out_slr1}, CL_MIGRATE_MEM_OBJECT_HOST));
    OCL_CHECK(err, err = q_slr2.enqueueMigrateMemObjects({buffer_mul_out_slr2}, CL_MIGRATE_MEM_OBJECT_HOST));
    q_slr0.finish();
    q_slr1.finish();
    q_slr2.finish();

    return EXIT_SUCCESS;
}

int compute_batch_frag0(char* argv2) {
    int DATA_SIZE = data_size;
    cl::CommandQueue& q_slr0 = *p_q_slr0;
    cl::Context& context = *p_context;
    cl::Kernel& vector_mult_slr0 = *p_vector_mult_slr0;
    
    std::vector<int, aligned_allocator<int>>& A_slr0 = *p_A_slr0;
    std::vector<int, aligned_allocator<int>>& B_slr0 = *p_B_slr0;
    std::vector<int, aligned_allocator<int>>& C_slr0 = *p_C_slr0;
    // get the batch_size
    int batch_size = 0;
    std::string batch_size_text = argv2;
    std::istringstream iss (batch_size_text);
    iss >> batch_size;
    if (iss.fail()) {
        return EXIT_FAILURE;
    } else {
        std::cout << "Batch_Size = " << batch_size << std::endl;
    }
    if (batch_size > max_batch_size) {
        std::cout << "error: allowed max_batch_size = " << max_batch_size << std::endl;
        return EXIT_FAILURE;
    }
    int size = DATA_SIZE*batch_size;
    size_t vector_size_bytes = sizeof(int) * DATA_SIZE*batch_size;

    // slr0 FPGA buffers
    OCL_CHECK(err,
                cl::Buffer buffer_in1_slr0(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    A_slr0.data(),
                                    &err));
    OCL_CHECK(err,
                cl::Buffer buffer_in2_slr0(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    B_slr0.data(),
                                    &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_mul_out_slr0(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_WRITE,
                                    vector_size_bytes,
                                    C_slr0.data(),
                                    &err));

    // kernel argeument slr0
    OCL_CHECK(err, err = vector_mult_slr0.setArg(0, buffer_in1_slr0));
    OCL_CHECK(err, err = vector_mult_slr0.setArg(1, buffer_in2_slr0));
    OCL_CHECK(err, err = vector_mult_slr0.setArg(2, buffer_mul_out_slr0));
    OCL_CHECK(err, err = vector_mult_slr0.setArg(3, size));

    // Copy input data to device global memory
    OCL_CHECK(err, err = q_slr0.enqueueMigrateMemObjects({buffer_in1_slr0, buffer_in2_slr0}, 0 /* 0 means from host*/));

    // Launch the Kernel
    OCL_CHECK(err, err = q_slr0.enqueueTask(vector_mult_slr0));

    OCL_CHECK(err, err = q_slr0.enqueueMigrateMemObjects({buffer_mul_out_slr0}, CL_MIGRATE_MEM_OBJECT_HOST));
    q_slr0.finish();

    return EXIT_SUCCESS;
}

int compute_batch_frag1(char* argv2) {
    int DATA_SIZE = data_size;
    cl::CommandQueue& q_slr1 = *p_q_slr1;
    cl::Context& context = *p_context;
    cl::Kernel& vector_mult_slr1 = *p_vector_mult_slr1;
    
    std::vector<int, aligned_allocator<int>>& A_slr1 = *p_A_slr1;
    std::vector<int, aligned_allocator<int>>& B_slr1 = *p_B_slr1;
    std::vector<int, aligned_allocator<int>>& C_slr1 = *p_C_slr1;
    // get the batch_size
    int batch_size = 0;
    std::string batch_size_text = argv2;
    std::istringstream iss (batch_size_text);
    iss >> batch_size;
    if (iss.fail()) {
        return EXIT_FAILURE;
    } else {
        std::cout << "Batch_Size = " << batch_size << std::endl;
    }
    if (batch_size > max_batch_size) {
        std::cout << "error: allowed max_batch_size = " << max_batch_size << std::endl;
        return EXIT_FAILURE;
    }
    int size = DATA_SIZE*batch_size;
    size_t vector_size_bytes = sizeof(int) * DATA_SIZE*batch_size;

    // slr1 FPGA buffers
    OCL_CHECK(err,
                cl::Buffer buffer_in1_slr1(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    A_slr1.data(),
                                    &err));
    OCL_CHECK(err,
                cl::Buffer buffer_in2_slr1(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    B_slr1.data(),
                                    &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_mul_out_slr1(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_WRITE,
                                    vector_size_bytes,
                                    C_slr1.data(),
                                    &err));

    // kernel argeument slr1
    OCL_CHECK(err, err = vector_mult_slr1.setArg(0, buffer_in1_slr1));
    OCL_CHECK(err, err = vector_mult_slr1.setArg(1, buffer_in2_slr1));
    OCL_CHECK(err, err = vector_mult_slr1.setArg(2, buffer_mul_out_slr1));
    OCL_CHECK(err, err = vector_mult_slr1.setArg(3, size));

    // Copy input data to device global memory
    OCL_CHECK(err, err = q_slr1.enqueueMigrateMemObjects({buffer_in1_slr1, buffer_in2_slr1}, 0 /* 0 means from host*/));

    // Launch the Kernel
    OCL_CHECK(err, err = q_slr1.enqueueTask(vector_mult_slr1));

    OCL_CHECK(err, err = q_slr1.enqueueMigrateMemObjects({buffer_mul_out_slr1}, CL_MIGRATE_MEM_OBJECT_HOST));
    q_slr1.finish();

    return EXIT_SUCCESS;
}

int compute_batch_frag2(char* argv2) {
    int DATA_SIZE = data_size;
    cl::CommandQueue& q_slr2 = *p_q_slr2;
    cl::Context& context = *p_context;
    cl::Kernel& vector_mult_slr2 = *p_vector_mult_slr2;
    
    std::vector<int, aligned_allocator<int>>& A_slr2 = *p_A_slr2;
    std::vector<int, aligned_allocator<int>>& B_slr2 = *p_B_slr2;
    std::vector<int, aligned_allocator<int>>& C_slr2 = *p_C_slr2;
    // get the batch_size
    int batch_size = 0;
    std::string batch_size_text = argv2;
    std::istringstream iss (batch_size_text);
    iss >> batch_size;
    if (iss.fail()) {
        return EXIT_FAILURE;
    } else {
        std::cout << "Batch_Size = " << batch_size << std::endl;
    }
    if (batch_size > max_batch_size) {
        std::cout << "error: allowed max_batch_size = " << max_batch_size << std::endl;
        return EXIT_FAILURE;
    }
    int size = DATA_SIZE*batch_size;
    size_t vector_size_bytes = sizeof(int) * DATA_SIZE*batch_size;

    // slr2 FPGA buffers
    OCL_CHECK(err,
                cl::Buffer buffer_in1_slr2(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    A_slr2.data(),
                                    &err));
    OCL_CHECK(err,
                cl::Buffer buffer_in2_slr2(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_ONLY,
                                    vector_size_bytes,
                                    B_slr2.data(),
                                    &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_mul_out_slr2(context,
                                    CL_MEM_USE_HOST_PTR | CL_MEM_READ_WRITE,
                                    vector_size_bytes,
                                    C_slr2.data(),
                                    &err));

    // kernel argeument slr2
    OCL_CHECK(err, err = vector_mult_slr2.setArg(0, buffer_in1_slr2));
    OCL_CHECK(err, err = vector_mult_slr2.setArg(1, buffer_in2_slr2));
    OCL_CHECK(err, err = vector_mult_slr2.setArg(2, buffer_mul_out_slr2));
    OCL_CHECK(err, err = vector_mult_slr2.setArg(3, size));

    // Copy input data to device global memory
    OCL_CHECK(err, err = q_slr2.enqueueMigrateMemObjects({buffer_in1_slr2, buffer_in2_slr2}, 0 /* 0 means from host*/));

    // Launch the Kernel
    OCL_CHECK(err, err = q_slr2.enqueueTask(vector_mult_slr2));

    OCL_CHECK(err, err = q_slr2.enqueueMigrateMemObjects({buffer_mul_out_slr2}, CL_MIGRATE_MEM_OBJECT_HOST));
    q_slr2.finish();

    return EXIT_SUCCESS;
}

void cleanup() {
    delete[] fileBuf;
    delete p_q_slr0;
    delete p_q_slr1;
    delete p_q_slr2;
    delete p_context;
    delete p_vector_mult_slr0;
    delete p_vector_mult_slr1;
    delete p_vector_mult_slr2;
    
    delete p_A_slr0;
    delete p_B_slr0;
    delete p_C_slr0;
    delete p_A_slr1;
    delete p_B_slr1;
    delete p_C_slr1;
    delete p_A_slr2;
    delete p_B_slr2;
    delete p_C_slr2;
}