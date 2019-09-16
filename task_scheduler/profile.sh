#!/bin/bash

SCHEDULER="./task_scheduler"
CALLER="../caller/caller"
CMD="vmult_slr_assign"
LATENCY_RECORD="latency.rec"
CALL_INTERVAL="20"

WORKER_NUM_BEGIN=1
WORKER_NUM_END=6

RPC_BEGIN=200
RPC_END=4000
# RPC_FACTOR=3
RPC_STEP=200
mv $LATENCY_RECORD ${LATENCY_RECORD}.old

mkdir -p log/scheduler
mkdir -p log/caller
go build
WORKER_NUM=$WORKER_NUM_BEGIN
while [[ $WORKER_NUM -le $WORKER_NUM_END ]]; do
    FRAC_NUM=`expr $WORKER_NUM \* 3`
    echo "<frac : $FRAC_NUM>" >>$LATENCY_RECORD
    RPC=$RPC_BEGIN
    while [[ $RPC -le $RPC_END ]]; do
        SCHEDULER_LOG="log/scheduler/interval${CALL_INTERVAL}-worker${WORKER_NUM}-rpc${RPC}-${CMD}.scheduler.log"
        CALLER_LOG="log/caller/interval${CALL_INTERVAL}-worker${WORKER_NUM}-rpc${RPC}-${CMD}.caller.log"
        echo "<reqPERsec : $RPC>" >>$LATENCY_RECORD
        $SCHEDULER -w $WORKER_NUM -l $LATENCY_RECORD >$SCHEDULER_LOG 2>&1 &
        SCHEDULER_PID=$!
        sleep 10s
        $CALLER -d $CALL_INTERVAL -r $RPC 2>&1 | tee $CALLER_LOG 
        kill $SCHEDULER_PID
        sleep 10s
        #RPC=`expr $RPC \* $RPC_FACTOR`
        RPC=`expr $RPC \+ $RPC_STEP`
    done
    WORKER_NUM=`expr $WORKER_NUM + 1`
done
