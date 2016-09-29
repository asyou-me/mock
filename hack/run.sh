#!/bin/bash

# 获取当前shell文件的路径
SOURCE="$0"
while [ -h "$SOURCE"  ]; do
    DIR="$( cd -P "$( dirname "$SOURCE"  )" && pwd  )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /*  ]] && SOURCE="$DIR/$SOURCE"
done
DIR="$( cd -P "$( dirname "$SOURCE"  )" && pwd  )"

killall -9 mock-server

# 编译文件 godep 
out=`go build -o "$DIR/../_out/mock-server" github.com/asyou-me/mock 2>&1 >/dev/null`

if [ $? -eq 0 ];then
  echo  -e  "\033[32m程序编译成功,开始执行\033[0m"
  "$DIR/../_out/mock-server" -http ":9090" -dir "$DIR/../test/data"
else
  echo  -e  "\033[31m程序编译出错,请检查代码哦\033[0m"
  echo "$out"
fi