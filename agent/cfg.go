package agent

const (
	// number of frames each agent will have access to in context
	// > previous messages to include
	historical_frames        = 12
	vision_historical_frames = 2 // reduced so vision context doesn't impact frame description.
)
