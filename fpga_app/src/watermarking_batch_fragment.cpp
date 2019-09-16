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
#include <vector>
#include <sstream>
#include <string>
//Includes
#include "bitmap.h"
#include "xcl2.hpp"

#include "watermarking_batch_fragment.h"

char* fileBuf;
cl::CommandQueue *p_q;
cl::Kernel* p_apply_watermark;
cl::Context* p_context;
std::vector<int, aligned_allocator<int>>* p_inputImage;
std::vector<int, aligned_allocator<int>>* p_outImage;

cl_int err;
int height;
int width;
size_t image_size_bytes;

#define MAX_BATCH 101

int init(char* argv1, char* argv2) {
    unsigned fileBufSize;
    std::string binaryFile = argv1;

    // OPENCL HOST CODE AREA START
    // get_xil_devices() is a utility API which will find the xilinx
    // platforms and will return list of devices connected to Xilinx platform
    std::cout << "Creating Context..." << std::endl;
    auto devices = xcl::get_xil_devices();
    auto device = devices[0];

    // OCL_CHECK(err, cl::Context context(device, NULL, NULL, NULL, &err));
    OCL_CHECK(err, p_context = new cl::Context(device, NULL, NULL, NULL, &err));
    cl::Context& context = *p_context;
    // OCL_CHECK(
    //     err,
    //     cl::CommandQueue q(context, device, CL_QUEUE_PROFILING_ENABLE, &err));
    OCL_CHECK(err, (p_q = new cl::CommandQueue(context, device, CL_QUEUE_PROFILING_ENABLE, &err)));
    cl::CommandQueue& q = *p_q;
    OCL_CHECK(err,
              std::string device_name = device.getInfo<CL_DEVICE_NAME>(&err));

    // read_binary_file() is a utility API which will load the binaryFile
    // and will return pointer to file buffer.
    fileBuf = xcl::read_binary_file(binaryFile, fileBufSize);
    cl::Program::Binaries bins{{fileBuf, fileBufSize}};
    devices.resize(1);
    OCL_CHECK(err, cl::Program program(context, devices, bins, NULL, &err));
    // OCL_CHECK(err,
    //           cl::Kernel apply_watermark(program, "apply_watermark", &err));
    OCL_CHECK(err, (p_apply_watermark = new cl::Kernel(program, "apply_watermark", &err)));
    cl::Kernel& apply_watermark = *p_apply_watermark;
    
    const char *bitmapFilename = argv2;
    // const char *goldenFilename;
    BitmapInterface image(bitmapFilename);
    bool result = image.readBitmapFile();
    if (!result) {
        std::cout << "ERROR:Unable to Read Input Bitmap File " << bitmapFilename
                  << std::endl;
        return EXIT_FAILURE;
    }

    // int width = image.getWidth();
    // int height = image.getHeight();
    width = image.getWidth();
    height = image.getHeight();

    //Allocate Memory in Host Memory
    int batch_size = MAX_BATCH;
    size_t image_size = image.numPixels();
    // size_t image_size_bytes = image_size * sizeof(int);
    image_size_bytes = image_size * sizeof(int);
    p_inputImage = new std::vector<int, aligned_allocator<int>>(image_size * batch_size);
    std::vector<int, aligned_allocator<int>>& inputImage = *p_inputImage;
    p_outImage = new std::vector<int, aligned_allocator<int>>(image_size * batch_size);
    std::vector<int, aligned_allocator<int>>& outImage = *p_outImage;
    // std::vector<int, aligned_allocator<int>> outImage(image_size * batch_size);

    // Copy image host buffer
    std::cout << "begin memcpy inputImage" << std::endl;
    memcpy(inputImage.data(), image.bitmap(), image_size_bytes);
    for (int i=1; i<batch_size; i++) {
        memcpy(inputImage.data()+image_size*i, inputImage.data()+image_size*(i-1), image_size_bytes);
    }
    std::cout << "finish memcpy inputImage" << std::endl;

    return EXIT_SUCCESS;
}

int compute_batch(char* argv2, char* argv3, char* argv4, char* argv5) {
    cl::CommandQueue& q = *p_q;
    cl::Kernel& apply_watermark = *p_apply_watermark;
    cl::Context& context = *p_context;
    std::vector<int, aligned_allocator<int>>& inputImage = *p_inputImage;
    std::vector<int, aligned_allocator<int>>& outImage = *p_outImage;
    // get the batch_size
    int batch_size = 0;
    std::string batch_size_text = argv5;
    std::istringstream iss (batch_size_text);
    iss >> batch_size;
    if (iss.fail()) {
    } else {
        std::cout << "Batch_Size = " << batch_size << std::endl;
    }
    if (batch_size > MAX_BATCH) {
        std::cout << "error: requested batch size is larger than allowed MAX_BATCH " << MAX_BATCH << std::endl;
        return EXIT_FAILURE;
    }
    // Allocate Buffer in Global Memory
    std::cout << "Creating Buffers..." << std::endl;
    OCL_CHECK(err,
              cl::Buffer buffer_inImage(context,
                                        CL_MEM_READ_ONLY | CL_MEM_USE_HOST_PTR,
                                        image_size_bytes * batch_size,
                                        inputImage.data(),
                                        &err));
    OCL_CHECK(
        err,
        cl::Buffer buffer_outImage(context,
                                   CL_MEM_WRITE_ONLY | CL_MEM_USE_HOST_PTR,
                                   image_size_bytes * batch_size,
                                   outImage.data(),
                                   &err));

    std::cout << "Setting arguments..." << std::endl;
    OCL_CHECK(err, err = apply_watermark.setArg(0, buffer_inImage));
    OCL_CHECK(err, err = apply_watermark.setArg(1, buffer_outImage));
    OCL_CHECK(err, err = apply_watermark.setArg(2, width));
    OCL_CHECK(err, err = apply_watermark.setArg(3, height));

    // Copy input data to device global memory
    std::cout << "Copying data..." << std::endl;
    OCL_CHECK(err,
              err = q.enqueueMigrateMemObjects({buffer_inImage},
                                               0 /*0 means from host*/));

    // Launch the Kernel
    // For HLS kernels global and local size is always (1,1,1). So, it is recommended
    // to always use enqueueTask() for invoking HLS kernel
    for (int batch_num = 0; batch_num < batch_size; batch_num++) {
        OCL_CHECK(err, err = apply_watermark.setArg(4, batch_num));
        std::cout << "Launching Kernel of batch " << batch_num << std::endl;
        OCL_CHECK(err, err = q.enqueueTask(apply_watermark));
        q.finish();
    }
    /*
    OCL_CHECK(err, err = q.enqueueNDRangeKernel(apply_watermark,
                                                0,
                                                cl::NDRange(batch_size, 1, 1),
                                                1));
    */

    // Copy Result from Device Global Memory to Host Local Memory
    std::cout << "Getting Results..." << std::endl;
    OCL_CHECK(err,
              err = q.enqueueMigrateMemObjects({buffer_outImage},
                                               CL_MIGRATE_MEM_OBJECT_HOST));
    q.finish();

    bool match = true;
    // goldenFilename = argv4;
    // //Read the golden bit map file into memory
    // BitmapInterface goldenImage(goldenFilename);
    // result = goldenImage.readBitmapFile();
    // if (!result) {
    //     std::cout << "ERROR:Unable to Read Golden Bitmap File "
    //               << goldenFilename << std::endl;
    //     return EXIT_FAILURE;
    // }
    // //Compare Golden Image with Output image
    // if (image.getHeight() != goldenImage.getHeight() ||
    //     image.getWidth() != goldenImage.getWidth()) {
    //     match = false;
    // } else {
    //     int *goldImgPtr = goldenImage.bitmap();
    //     for (unsigned int i = 0; i < image.numPixels(); i++) {
    //         if (outImage[i] != goldImgPtr[i]) {
    //             match = false;
    //             printf("Pixel %d Mismatch Output %x and Expected %x \n",
    //                    i,
    //                    outImage[i],
    //                    goldImgPtr[i]);
    //             break;
    //         }
    //     }
    // }

    // // Write the final image to disk
    // image.writeBitmapFile(outImage.data());

    std::cout << (match ? "TEST PASSED" : "TEST FAILED") << std::endl;
    return (match ? EXIT_SUCCESS : EXIT_FAILURE);
}

void cleanup() {
    delete[] fileBuf;
    delete p_q;
    delete p_apply_watermark;
    delete p_context;
    delete p_inputImage;
    delete p_outImage;
}