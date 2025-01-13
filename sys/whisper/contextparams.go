package whisper

import "encoding/json"

///////////////////////////////////////////////////////////////////////////////
// CGO

/*
#cgo pkg-config: libwhisper
#include <whisper.h>
*/
import "C"

///////////////////////////////////////////////////////////////////////////////
// TYPES

type (
	ContextParams C.struct_whisper_context_params
)

///////////////////////////////////////////////////////////////////////////////
// LIFECYCLE

func DefaultContextParams() ContextParams {
	return (ContextParams)(C.whisper_context_default_params())
}

///////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (ctx ContextParams) MarshalJSON() ([]byte, error) {
	type j struct {
		UseGpu          bool `json:"use_gpu"`
		GpuDevice       int  `json:"gpu_device"`
		FlashAttn       bool `json:"flash_attn"`
		TokenTimestamps bool `json:"dtw_token_timestamps"`
	}
	return json.Marshal(j{
		UseGpu:          bool(ctx.use_gpu),
		GpuDevice:       int(ctx.gpu_device),
		FlashAttn:       bool(ctx.flash_attn),
		TokenTimestamps: bool(ctx.dtw_token_timestamps),
	})
}

func (ctx ContextParams) String() string {
	str, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(str)
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

func (ctx *ContextParams) UseGpu() bool {
	return bool(ctx.use_gpu)
}

func (ctx *ContextParams) SetUseGpu(v bool) {
	ctx.use_gpu = (C.bool)(v)
}

func (ctx *ContextParams) GpuDevice() int {
	return int(ctx.gpu_device)
}

func (ctx *ContextParams) SetGpuDevice(v int) {
	ctx.gpu_device = (C.int)(v)
}

func (ctx *ContextParams) FlashAttn() bool {
	return bool(ctx.flash_attn)
}

func (ctx *ContextParams) SetFlashAttn(v bool) {
	ctx.flash_attn = (C.bool)(v)
}

func (ctx *ContextParams) TokenTimestamps() bool {
	return bool(ctx.dtw_token_timestamps)
}

func (ctx *ContextParams) SetTokenTimestamps(v bool) {
	ctx.dtw_token_timestamps = (C.bool)(v)
}

func (ctx *ContextParams) DTWAheadsPreset() AlignmentAheadsPreset {
	return AlignmentAheadsPreset(ctx.dtw_aheads_preset)
}

func (ctx *ContextParams) SetDTWAheadsPreset(v AlignmentAheadsPreset) {
	ctx.dtw_aheads_preset = (C.enum_whisper_alignment_heads_preset)(v)
}

func (ctx *ContextParams) DTWNTop() int {
	return int(ctx.dtw_n_top)
}

func (ctx *ContextParams) SetDTWNTop(nTop int) {
	ctx.dtw_n_top = (C.int)(nTop)
}

func (ctx *ContextParams) DTWMemSize() uintptr {
	return uintptr(ctx.dtw_mem_size)
}

func (ctx *ContextParams) SetDTWMemSize(memSize uintptr) {
	ctx.dtw_mem_size = (C.size_t)(memSize)
}
