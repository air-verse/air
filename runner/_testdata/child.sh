#!/bin/sh
echo "ProcessID=$$ begins ($0)"
echo "$$" >> pid
./_testdata/grandchild.sh background &
./_testdata/grandchild.sh foreground
echo "ProcessID=$$ ends ($0)"