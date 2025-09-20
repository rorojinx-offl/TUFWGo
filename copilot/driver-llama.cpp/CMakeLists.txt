cmake_minimum_required(VERSION 4.0)
project(TUFWGo)

set(CMAKE_CXX_STANDARD 14)

include_directories(.)

add_executable(tufwgo_llm tufwgo_llm.cpp)

list(APPEND CMAKE_PREFIX_PATH "/usr/local")
find_package(llama REQUIRED CONFIG)
message("llama_dir = ${llama_DIR}")

if (TARGET llama::llama)
    target_link_libraries(tufwgo_llm PRIVATE llama::llama)
elseif (TARGET llama)
    target_link_libraries(tufwgo_llm PRIVATE llama)
else()
    message(FATAL_ERROR "Found llama package at '${llama_DIR}', but no 'llama::llama' or 'llama' target is exported.")
endif()
