package task

import (
	"io"
	"time"

	// Packages
	"github.com/mutablelogic/go-whisper/pkg/schema"
	"github.com/mutablelogic/go-whisper/sys/whisper"
)

//////////////////////////////////////////////////////////////////////////////
// LIFECYCLE

func newSegment(ts time.Duration, offset int32, seg *whisper.Segment) *schema.Segment {
	tokens := make([]string, 0, len(seg.Tokens))
	for _, token := range seg.Tokens {
		if token.Text != "" {
			tokens = append(tokens, token.Text)
		}
	}
	return &schema.Segment{
		Id:          offset + seg.Id,
		Text:        seg.Text,
		Tokens:      tokens,
		Start:       schema.Timestamp(seg.T0 + ts),
		End:         schema.Timestamp(seg.T1 + ts),
		SpeakerTurn: seg.SpeakerTurn,
	}
}

//////////////////////////////////////////////////////////////////////////////
// PUBLIC METHODS

func WriteSegmentSrt(w io.Writer, seg *schema.Segment) {
	seg.WriteSRT(w, 0)
}

func WriteSegmentVtt(w io.Writer, seg *schema.Segment) {
	seg.WriteVTT(w, 0)
}

func WriteSegmentText(w io.Writer, seg *schema.Segment) {
	seg.WriteText(w)
}
