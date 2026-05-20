#!/bin/sh
echo "ProcessID=$$ begins ($0)"
echo "$$" >> pid
setsid sh -c './_testdata/grandchild.sh detached' >/dev/null 2>&1 &
DETACHED_PID=$!
echo "$DETACHED_PID" >> pid
sleep 9999
echo "ProcessID=$$ ends ($0)"
