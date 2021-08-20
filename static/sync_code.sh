#!/bin/bash

find . -name '*.go' -exec sed -i 's@github.com/golang/mock/mockgen@github.com/qjpcpu/common.v2/static/mockgen@g' {} \;
sed -i 's@package main@package mockgen@' mockgen/*.go
