package whisper

/*
#cgo pkg-config: libwhisper
#include <whisper.h>
*/
import "C"

type AlignmentAheadsPreset C.enum_whisper_alignment_heads_preset

const (
	AlignmentAheadsPresetNone     = C.WHISPER_AHEADS_NONE
	AlignmentAheadsPresetNTopMost = C.WHISPER_AHEADS_N_TOP_MOST
	AlignmentAheadsPresetCustom   = C.WHISPER_AHEADS_CUSTOM
	AlignmentAheadsPresetTinyEn   = C.WHISPER_AHEADS_TINY_EN
	AlignmentAheadsPresetTiny     = C.WHISPER_AHEADS_TINY
	AlignmentAheadsPresetBaseEn   = C.WHISPER_AHEADS_BASE_EN
	AlignmentAheadsPresetBase     = C.WHISPER_AHEADS_BASE
	AlignmentAheadsPresetSmallEn  = C.WHISPER_AHEADS_SMALL_EN
	AlignmentAheadsPresetSmall    = C.WHISPER_AHEADS_SMALL
	AlignmentAheadsPresetMediumEn = C.WHISPER_AHEADS_MEDIUM_EN
	AlignmentAheadsPresetMedium   = C.WHISPER_AHEADS_MEDIUM
	AlignmentAheadsPresetLargeV1  = C.WHISPER_AHEADS_LARGE_V1
	AlignmentAheadsPresetLargeV2  = C.WHISPER_AHEADS_LARGE_V2
	AlignmentAheadsPresetLargeV3  = C.WHISPER_AHEADS_LARGE_V3
)
