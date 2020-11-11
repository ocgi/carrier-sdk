# cpp sdk

该sdk依赖grpc的相关库，如果你使用的是tlinux2.x系统，我们提供预编译好的rpm文件，你可以通过下面的方式来安装
首先，grpc需要比较高版本的stdc++库，用下面的命令来更新：
```
yum install tlinux-release-gcc-update
yum update libstdc++
```
然后，如果只是需要运行环境，则依次安装下面几个rpm包：

* [protobuf](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/protobuf-3.13.0-1.tl2.x86_64.rpm)
* [protobuf-compiler](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/protobuf-compiler-3.13.0-1.tl2.x86_64.rpm)
* [grpc-abseil](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/grpc-abseil_20200225.3-1.33.2-2.1.x86_64.rpm)
* [grpc-c-ares](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/grpc-c-ares_1_15_0-1.33.2-2.1.x86_64.rpm)
* [grpc](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/grpc-1.33.2-2.1.x86_64.rpm)

如果需要做编译开发工作，则还需要额外安装下面的开发包：

* [protobuf-devel](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/protobuf-devel-3.13.0-1.tl2.x86_64.rpm)
* [grpc-devel](http://mirror.tenc.oa.com/iegsa-test/tlinux-2.2/x86_64/tenc/grpc-devel-1.33.2-2.1.x86_64.rpm)


对于其他的环境，可以参考下面的文档来编译依赖：

* protobuf 我们使用[3.13.0](https://github.com/protocolbuffers/protobuf/releases/download/v3.13.0/protobuf-all-3.13.0.tar.gz)这个版本，可以使用--disable-shared来编译静态库，有利于软件分发
* grpc++ 具体的编译可以参考[这里](https://github.com/grpc/grpc/blob/master/BUILDING.md)


## 使用方式

执行`make sdk`编译sdk文件，如果成功的话会生成静态库`libcarrier.a`和动态库`libcarrier.so`，项目编译时可以自行选择需要的版本

[sdk_test.cc](sdk_test.cc)中有示例使用方式，编译方式可以参考 [Makefile](Makefile)。
