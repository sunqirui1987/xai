package xai

// IsStreamTextDelta reports whether r is an incremental text chunk from GenStream
// (e.g. OpenAI chat.completion.chunk deltas). Final aggregated responses are not deltas.
func IsStreamTextDelta(r GenResponse) bool {
	type delta interface {
		IsStreamTextDelta() bool
	}
	d, ok := r.(delta)
	return ok && d.IsStreamTextDelta()
}
