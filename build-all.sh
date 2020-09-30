#!/bin/sh

for TARGETPLATFORM in \
  linux/386 linux/amd64 linux/arm/5 linux/arm/6 linux/arm/7 linux/arm64 linux/mips linux/mips64 linux/mips64le linux/mipsle linux/ppc64 \
  linux/ppc64le linux/riscv64 linux/s390x \
  darwin/386 darwin/amd64 \
  dragonfly/amd64 \
  freebsd/386 freebsd/amd64 freebsd/arm freebsd/arm64 \
  netbsd/386 netbsd/amd64 netbsd/arm netbsd/arm64 \
  windows/386 windows/amd64 \
  solaris/amd64 \
  openbsd/386 openbsd/amd64 openbsd/arm openbsd/arm64
do
  export TARGETPLATFORM
  ./build.sh &
done

wait # to not exit the containing script until all execution has finished