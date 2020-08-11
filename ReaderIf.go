package dashreader

import (
	"context"
	"net/url"
	"time"
)

//ChunkURL - URL extracted from MPD for playback
type ChunkURL struct {
	//ChunkURL - Actual URL
	ChunkURL url.URL
	//FetchAt - WallClock Time when URL becomes available
	FetchAt time.Time
	//Duration - Duration of content available in this URL
	Duration time.Duration
}

//ChunkURLChannel - Channel of Chunk URLs
type ChunkURLChannel chan ChunkURL

//ReaderContext - Unique data for each Reader
type ReaderContext interface {
	//NextURL -
	//-- Once end is reached (io.EOF)
	//-- MakeDASHReaderContext has to be called again
	// Parameters;
	//   None
	// Return:
	//   1: Next URL
	//   2: error
	NextURL() (ret ChunkURL, err error)

	//GetURLs - Get URLs from Current MPD context
	//-- Once end of this list is reached
	//-- MakeDASHReaderContext has to be called again
	// Parameters;
	//   context for cancellation
	// Return:
	//   1: Channel of URLs, can be read till closed
	//   2: error
	GetURLs(context.Context) (ret <-chan ChunkURL, err error)
}

//Reader - Read any DASH file and get Playback URLs
type Reader interface {
	//Update -
	// Parameters:
	//   MPD read
	// Return:
	//   1: MPD Updated - PublishTime Updated?
	//   2: New Period  - NewPeriod Updated?
	//   3: error
	Update(*MPDtype) (bool, bool, error)

	//MakeDASHReaderContext - Makes Reader Context
	// Parameters:
	//   1: Context received earlier... if first time pass nil
	//   2: StreamSelector for the ContentType to select AdaptationSet
	//   3: RepresentationSelector ... selector for Representation
	// Return:
	//   1: Context for current AdaptationSet,Representation
	//   2: error
	MakeDASHReaderContext(ReaderContext, StreamSelector, RepresentationSelector) (ReaderContext, error)
}
