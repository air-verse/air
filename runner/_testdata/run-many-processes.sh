#!/bin/sh
echo "ProcessID=$$ begins ($0)"
./_testdata/child.sh background &
./_testdata/child.sh foreground
echo "ProcessID=$$ ends ($0)"