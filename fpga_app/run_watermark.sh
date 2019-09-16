#!/bin/bash

# LOCAL_WORKSPACE="/workspace"
# mkdir -p $LOCAL_WORKSPACE
# for item in $(ls $FPGA_APP_DIR); do
#   ln -s $FPGA_APP_DIR/$item $LOCAL_WORKSPACE/
# done
# cd $LOCAL_WORKSPACE
# ls ./
# go build ./watermark_client.go
cd $FPGA_APP_DIR
./watermark_client
