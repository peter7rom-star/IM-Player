#!/bin/bash
filename=$basename $0
echo $filename
go install .
sudo cp ~/go/bin/$basename /usr/local/bin