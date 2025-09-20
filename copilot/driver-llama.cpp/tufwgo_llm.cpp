#include "llama.h"
#include <iostream>
#include <fstream>
#include <sstream>
#include <vector>

std::string read_file(const std::string &path) {
    std::ifstream f(path);
    std::stringstream ss;
    ss << f.rdbuf();
    return ss.str();
}

int main(const int argc, char **argv) {
    if (argc < 4) {
        std::cerr << "Usage: tufwgo_llm <model.gguf> <grammar.gbnf> <prompt>\n";
        return 1;
    }
    std::string model_path = argv[1];
    std::string grammar_path = argv[2];
    std::string prompt = argv[3];

    llama_backend_init();
    llama_model_params mparams = llama_model_default_params();
    llama_model *model = llama_model_load_from_file(model_path.c_str(),mparams);
    if (!model) {
        std::cerr << "Failed to load model\n";
        return 1;
    }

    llama_context_params cparams = llama_context_default_params();
    cparams.n_ctx = 2048;
    llama_context *ctx = llama_init_from_model(model,cparams);

    std::string grammar = read_file(grammar_path);
}