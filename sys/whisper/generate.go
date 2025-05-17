package whisper

///////////////////////////////////////////////////////////////////////////////
// CGO

/*
#cgo pkg-config: libwhisper
#cgo linux pkg-config: libwhisper-linux
#cgo darwin pkg-config: libwhisper-darwin
*/
import "C"

// Generate the whisper pkg-config files
// Setting the prefix to the base of the repository
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --cflags "-I$DOLLAR{prefix}/third_party/whisper.cpp/include -I$DOLLAR{prefix}/third_party/whisper.cpp/ggml/include" libwhisper.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --cflags "-fopenmp" --libs "-L$DOLLAR{prefix}/build/ggml/src -L$DOLLAR{prefix}/build/src -lwhisper -lggml -lggml-base -lggml-cpu -lgomp -lm -lstdc++" libwhisper-linux.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --libs "-L$DOLLAR{prefix}/build/ggml/src -L$DOLLAR{prefix}/build/ggml/src/ggml-blas -L$DOLLAR{prefix}/build/ggml/src/ggml-metal -L$DOLLAR{prefix}/build/src -lwhisper -lggml -lggml-base -lggml-cpu -lggml-blas -lggml-metal -lm -lstdc++ -framework Accelerate -framework Metal -framework Foundation -framework CoreGraphics" libwhisper-darwin.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --libs "-L$DOLLAR{prefix}/build/ggml/src/ggml-cuda -lggml-cuda" libwhisper-cuda.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --libs "-L$DOLLAR{prefix}/build/ggml/src/ggml-vulkan -lvulkan -lggml-vulkan" libwhisper-vulkan.pc
