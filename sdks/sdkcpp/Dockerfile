From hub.oa.com/public/tlinux2.2-slim:v0.6 as builder

RUN yum install -y tlinux-release-gcc-update && \
    yum update -y libstdc++ && \
    curl http://mirror.tenc.oa.com/iegsa-test/iegsa.repo -o /etc/yum.repos.d/iegsa.repo && \
    yum install -y cmake3 gcc-c++ make rpm-build grpc-devel

ADD . /root/sdkcpp

WORKDIR /root/sdkcpp
RUN rm -rf build && \
    mkdir build && \
    cd build && \
    cmake3 .. && \
    cpack3 -G RPM 

From hub.oa.com/public/tlinux2.2-slim:v0.6 as worker

RUN yum install -y tlinux-release-gcc-update && \
    yum update -y libstdc++ && \
    curl http://mirror.tenc.oa.com/iegsa-test/iegsa.repo -o /etc/yum.repos.d/iegsa.repo && \
    yum install -y grpc-devel

COPY --from=0 /root/sdkcpp/build/carrier-*-Linux.rpm .

RUN rpm -Uvh carrier-*-Linux.rpm
