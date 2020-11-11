.PHONY: all clean

CXXFLAGS=-Iapi/v1alpha1 -std=c++11 -fPIC
OBJS= $(subst .cc,.o,$(wildcard api/v1alpha1/*.cc)) $(subst .cc,.o,$(wildcard api/v1alpha1/google/api/*.cc))

all: sdk sdk_test sdk_test_static mock_server mock_server_static

libcarrier.a: $(OBJS)
	$(AR) crv $@ $^

libcarrier.so: $(OBJS)
	$(CXX) -shared -o $@ $^

sdk: libcarrier.a libcarrier.so api/v1alpha1/sdk.h

sdk_test: sdk_test.cc
	$(CXX)  $(CXXFLAGS) -O0 -ggdb $< -L. -lcarrier  `pkg-config --libs protobuf grpc++` `pkg-config --cflags protobuf grpc++` -o $@

sdk_test_static: sdk_test.cc
	$(CXX)  $(CXXFLAGS) -O0 -ggdb $<  libcarrier.a  `pkg-config --libs protobuf grpc++` `pkg-config --cflags protobuf grpc++` -o $@

mock_server: mock_server.cc
	$(CXX)  $(CXXFLAGS) -O0 -ggdb $< -L. -lcarrier `pkg-config --libs protobuf grpc++` `pkg-config --cflags protobuf grpc++` -o $@

mock_server_static: mock_server.cc
	$(CXX)  $(CXXFLAGS) -O0 -ggdb $<  libcarrier.a `pkg-config --libs protobuf grpc++` `pkg-config --cflags protobuf grpc++` -o $@

clean:
	rm -f api/v1alpha1/*.o api/v1alpha1/google/api/*.o
	rm -f libcarrier.a libcarrier.so
	rm -f sdk_test mock_server sdk_test_static mock_server_static
