#!/bin/bash

function run_test_in()
{
  dir=$1
  echo "Run test in $dir"
  go test $dir
  if [ $? -ne 0 ];then
    exit 1
  fi
}

for dir in `find . -name '*.go' -exec dirname {} \; |sort|uniq`;do
  run_test_in $dir
done
