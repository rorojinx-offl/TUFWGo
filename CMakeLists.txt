cmake_minimum_required(VERSION 4.0)
project(TUFWGo)

set(CMAKE_CXX_STANDARD 14)

include_directories(copilot/driver-llama.cpp)

add_executable(TUFWGo
        copilot/driver-llama.cpp/tufwgo_llm.cpp
)
