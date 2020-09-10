#!/bin/bash
cd `dirname $0`/../
rm function.zip
rm main
zip -r9 function.zip templates
zip -g -r9 function.zip constant
GOOS=linux go build main.go
zip -g function.zip main
