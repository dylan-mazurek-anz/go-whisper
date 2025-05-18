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
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --cflags "-I$DOLLAR{prefix}/include" libwhisper.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --cflags "-fopenmp" --libs "-L$DOLLAR{prefix}/lib -lwhisper -lggml -lggml-base -lggml-cpu -lgomp -lm -lstdc++" libwhisper-linux.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --libs "-L$DOLLAR{prefix}/lib -lwhisper -lggml -lggml-base -lggml-cpu -lggml-blas -lggml-metal -lm -lstdc++ -framework Accelerate -framework Metal -framework Foundation -framework CoreGraphics" libwhisper-darwin.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --libs "-L$DOLLAR{prefix}/lib -lggml-cuda" libwhisper-cuda.pc
//go:generate go run ../pkg-config --version "0.0.0" --prefix "${PREFIX}" --libs "-L$DOLLAR{prefix}/lib -lvulkan -lggml-vulkan" libwhisper-vulkan.pc
