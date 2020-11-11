cmake_minimum_required(VERSION 2.8)
project(carrier)

file(READ "VERSION" ver)

string(REGEX MATCH "VERSION_MAJOR ([0-9]*)" _ ${ver})
set(PROJECT_VERSION_MAJOR ${CMAKE_MATCH_1})
string(REGEX MATCH "VERSION_MINOR ([0-9]*)" _ ${ver})
set(PROJECT_VERSION_MINOR ${CMAKE_MATCH_1})
string(REGEX MATCH "VERSION_PATCH ([0-9]*)" _ ${ver})
set(PROJECT_VERSION_PATCH ${CMAKE_MATCH_1})
set(PROJECT_VERSION  ${PROJECT_VERSION_MAJOR}.${PROJECT_VERSION_MINOR}.${PROJECT_VERSION_PATCH})
message("version: ${PROJECT_VERSION}")


INCLUDE(CheckCCompilerFlag)
INCLUDE(CheckCXXCompilerFlag)
set(CMAKE_EXPORT_COMPILE_COMMANDS TRUE)

CHECK_CXX_COMPILER_FLAG("-std=c++11" HAVE_CXX11)

if(NOT HAVE_CXX11)
    message(FATAL_ERROR "this project need c++11")
endif()

set(CMAKE_CXX_FLAGS "-std=c++11 -Wall -Wextra -fno-omit-frame-pointer ${CMAKE_CXX_FLAGS}")

if(CMAKE_INSTALL_PREFIX_INITIALIZED_TO_DEFAULT)
  set(CMAKE_INSTALL_PREFIX "/usr" CACHE PATH "default install prefix" FORCE)
endif()

# Find gRPC installation
# Looks for gRPCConfig.cmake file installed by gRPC's cmake installation.
find_package(gRPC 1.33 REQUIRED)
message(STATUS "Using gRPC ${gRPC_VERSION}")
set(_GRPC_GRPCPP gRPC::grpc++)

include_directories(${PROJECT_SOURCE_DIR}/api/v1alpha1)

file(GLOB SdkSource "api/v1alpha1/*.cc" "api/v1alpha1/google/api/*.cc")
file(GLOB SdkHeader "api/v1alpha1/*.h")

add_library(objs OBJECT ${SdkSource})
# shared libraries need PIC
set_property(TARGET objs PROPERTY POSITION_INDEPENDENT_CODE 1)

add_library(carrier_static STATIC $<TARGET_OBJECTS:objs>)
set_target_properties(carrier_static PROPERTIES OUTPUT_NAME "carrier")
target_link_libraries(carrier_static)


add_library(carrier_share SHARED $<TARGET_OBJECTS:objs>)
set_target_properties(carrier_share PROPERTIES OUTPUT_NAME "carrier")
#set_target_properties(carrier_share PROPERTIES PUBLIC_HEADER "${SdkHeader}") 
target_link_libraries(carrier_share)

add_executable(sdktest sdk_test.cc)
target_link_libraries(sdktest carrier_static ${_GRPC_GRPCPP})

add_executable(mock_server mock_server.cc)
target_link_libraries(mock_server carrier_share ${_GRPC_GRPCPP})

include(CMakePackageConfigHelpers)
include(GNUInstallDirs)
install(TARGETS carrier_static EXPORT carrier_static
        LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR})

install(TARGETS carrier_share  EXPORT carrier_share
        LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR})

install(DIRECTORY  ${PROJECT_SOURCE_DIR}/api/v1alpha1/
        DESTINATION ${CMAKE_INSTALL_INCLUDEDIR}/${CMAKE_PROJECT_NAME} 
        FILES_MATCHING PATTERN "*.h")

install(EXPORT carrier_static DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/${PROJECT_NAME}/)
install(EXPORT carrier_share DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/${PROJECT_NAME}/)

configure_file(cmake/carrierConfig.cmake.in ${PROJECT_BINARY_DIR}/carrierConfig.cmake @ONLY)
write_basic_package_version_file("${PROJECT_BINARY_DIR}/carrierConfigVersion.cmake"
    VERSION ${PROJECT_VERSION}
    COMPATIBILITY SameMajorVersion)

install(FILES ${PROJECT_BINARY_DIR}/carrierConfig.cmake DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/${PROJECT_NAME}/)
install(FILES ${PROJECT_BINARY_DIR}/carrierConfigVersion.cmake DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/${PROJECT_NAME}/)

set(CPACK_PACKAGE_NAME "${PROJECT_NAME}")
set(CPACK_PACKAGE_VENDOR "ocgi")
set(CPACK_PACKAGE_DESCRIPTION_FILE "${PROJECT_SOURCE_DIR}/README.md")
set(CPACK_PACKAGE_VERSION "${PROJECT_VERSION}")
set(CPACK_PACKAGE_CONTACT "choury chouryzhou@tencent.com")
#set(CPACK_SET_DESTDIR "ON")

set(CPACK_GENERATOR "ZIP;TGZ")
if(CMAKE_SYSTEM_NAME MATCHES Linux)
    list(APPEND CPACK_GENERATOR "RPM")
    set(CPACK_RPM_PACKAGE_RELEASE "1")
    set(CPACK_RPM_PACKAGE_GROUP "development")
    set(CPACK_RPM_PACKAGE_LICENSE "GPL")
    set(CPACK_RPM_PACKAGE_REQUIRES "grpc >= 1.33")
    set(CPACK_RPM_PACKAGE_RELEASE_DIST "ON")
    set(CPACK_RPM_EXCLUDE_FROM_AUTO_FILELIST_ADDITION "/usr/${CMAKE_INSTALL_LIBDIR}/cmake")
endif()

message(STATUS "CPack generators: ${CPACK_GENERATOR}")
include(CPack)

add_custom_target(image
    COMMAND docker build ${PROJECT_SOURCE_DIR} -t hub.oa.com/library/${PROJECT_NAME}:v${PROJECT_VERSION}
)

