package transcripts

type AudioFormat string

const (
	AudioFormatPcm_8000  AudioFormat = "pcm_8000"
	AudioFormatPcm_16000 AudioFormat = "pcm_16000"
	AudioFormatPcm_22050 AudioFormat = "pcm_22050"
	AudioFormatPcm_24000 AudioFormat = "pcm_24000"
	AudioFormatPcm_44100 AudioFormat = "pcm_44100"
	AudioFormatPcm_48000 AudioFormat = "pcm_48000"
	AudioFormatUlaw_8000 AudioFormat = "ulaw_8000"
)

type CommitStrategy string

const (
	CommitStrategyManual CommitStrategy = "manual"
	CommitStrategyVAD    CommitStrategy = "vad"
)
