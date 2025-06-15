package wav

import (
	"io"

	// Packages
	audio "github.com/go-audio/audio"
	wav "github.com/go-audio/wav"
	writerseeker "github.com/orcaman/writerseeker"
)

type WaveAudio struct {
	io.Reader
}

// Create a new mono WAV file with 16-bit signed integer samples
func NewInt16(data []int16, sampleRate int) (*WaveAudio, error) {
	buf := new(writerseeker.WriterSeeker)
	encoder := wav.NewEncoder(buf, sampleRate, 16, 1, 1)
	pcmbuf := audio.PCMBuffer{
		I16:      data,
		DataType: audio.DataTypeI16,
		Format: &audio.Format{
			SampleRate:  sampleRate,
			NumChannels: 1,
		},
	}
	if err := encoder.Write(pcmbuf.AsIntBuffer()); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	// Return a new WaveAudio with the writerseeker as the reader
	return &WaveAudio{
		Reader: buf.Reader(),
	}, nil
}
