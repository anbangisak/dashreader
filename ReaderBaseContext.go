package dashreader

import (
	"context"
	"fmt"
	"time"

	"github.com/eswarantg/statzagg"
)

//readerBaseContext - Base context
type readerBaseContext struct {
	ID             string                 //ID for the ReaderContext
	updCounter     int64                  //to sync between Context and Reader
	StatzAgg       statzagg.StatzAgg      //Statz Agg
	repSelector    RepresentationSelector //Selector for Representation
	streamSelector StreamSelector         //Selector for stream
	adaptSetID     uint                   //ID of adapatationSet
	repID          StringNoWhitespaceType //selected RepresentationID

	//Context fields
	frameRate   float64
	lang        string
	contentType string
	codecs      string
}

//Select - select AdaptationSet and Representation
func (c *readerBaseContext) Select(p PeriodType) error {
	adaptSet := c.selectAdapationSets(p)
	if adaptSet == nil {
		if c.StatzAgg != nil {
			values := make([]interface{}, 1)
			values[0] = len(p.AdaptationSet)
			c.StatzAgg.PostEventStats(context.TODO(), &statzagg.EventStats{
				EventClock: time.Now(),
				ID:         c.ID,
				Name:       EvtMPDNoAdaptAfterFilter,
				Values:     values,
			})
		}
		return fmt.Errorf("ReaderContext(%v) no AdaptationSet selected", c.ID)
	}
	reps := c.filterRepresentation(*adaptSet)
	rep := c.repSelector.SelectRepresentation(reps)
	if rep == nil {
		if c.StatzAgg != nil {
			values := make([]interface{}, 1)
			values[0] = len(adaptSet.Representation)
			c.StatzAgg.PostEventStats(context.TODO(), &statzagg.EventStats{
				EventClock: time.Now(),
				ID:         c.ID,
				Name:       EvtMPDNoRepresentationAfterFilter,
				Values:     values,
			})
		}
		return fmt.Errorf("ReaderContext(%v) AdaptationSet(%v) no Representation selected", c.ID, adaptSet.Id)
	}
	c.adaptSetID = adaptSet.Id
	c.repID = rep.Id
	return nil
}

//selectAdapationSets for period
func (c *readerBaseContext) selectAdapationSets(p PeriodType) *AdaptationSetType {
	var ret *AdaptationSetType
	ret = nil
	lastMatchResp := MatchResultDontCare
	for i := range p.AdaptationSet {
		//Valid ContentType Check
		if len(p.AdaptationSet[i].ContentType) <= 0 {
			continue
		}
		matchResp := c.streamSelector.IsMatch(p.AdaptationSet[i])
		//Check if it is not a match
		if matchResp == MatchResultNotFound {
			continue
		}
		//Check if found first time or this is a better match
		if ret == nil || matchResp > lastMatchResp {
			ret = &p.AdaptationSet[i]
			matchResp = lastMatchResp
		}
	}
	return ret
}

//filterRepresentation - From among the representations select the right representation
func (c *readerBaseContext) filterRepresentation(a AdaptationSetType) []*RepresentationType {
	var foundList []*RepresentationType
	var partialList []*RepresentationType
	var dontCareList []*RepresentationType
	foundList = []*RepresentationType{}
	foundList = []*RepresentationType{}
	dontCareList = []*RepresentationType{}
	for i := range a.Representation {
		switch c.streamSelector.IsMatchRepresentation(a.Representation[i]) {
		case MatchResultFound:
			foundList = append(foundList, &a.Representation[i])
		case MatchResultDontCare:
			dontCareList = append(dontCareList, &a.Representation[i])
		case MatchResultPartial:
			partialList = append(partialList, &a.Representation[i])
		}
	}
	foundList = append(foundList, partialList...)
	foundList = append(foundList, dontCareList...)
	return foundList
}

//NextURL -
//-- Once end is reached (io.EOF)
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   None
// Return:
//   1: Next URL
//   2: error
func (c *readerBaseContext) NextURL() (*ChunkURL, error) {
	return nil, fmt.Errorf("readerBaseContext NextURL NOT IMPLEMENTED")
}

//NextURLs - Get URLs from Current MPD context
//-- Once end of this list is reached
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   context for cancellation
// Return:
//   1: Channel of URLs, can be read till closed
//   2: error
func (c *readerBaseContext) NextURLs(ctx context.Context) (ret <-chan ChunkURL, err error) {
	return c.getURLs(ctx, ReaderContext(c))
}

//getURLs - Get URLs from Current MPD context
//-- Once end of this list is reached
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   context for cancellation
//   ReaderContext
// Return:
//   1: Channel of URLs, can be read till closed
//   2: error
func (c *readerBaseContext) getURLs(ctx context.Context, rdrCtx ReaderContext) (ret <-chan ChunkURL, err error) {
	var chunkURL *ChunkURL
	chunkURL, err = rdrCtx.NextURL()
	if err != nil {
		return nil, err
	}
	ch := make(chan ChunkURL, 10)
	go func(ctx context.Context, rdrCtx ReaderContext, ch ChunkURLChannel, chunkURL ChunkURL) {
		defer close(ch)
		ch <- chunkURL
		for {
			select {
			case <-ctx.Done():
				return
			default:
				chunkURL, err := rdrCtx.NextURL()
				if err != nil {
					return
				}
				ch <- *chunkURL
			}
		}
	}(ctx, rdrCtx, ch, *chunkURL)
	return ch, nil
}

//GetFramerate - Framerate of content
func (c *readerBaseContext) GetFramerate() float64 {
	return c.frameRate
}

//GetContentType - Content Type of content
func (c *readerBaseContext) GetContentType() string {
	return c.contentType
}

//GetLang - Lang of content
func (c *readerBaseContext) GetLang() string {
	return c.lang
}

//GetCodecs - Codecs of content
func (c *readerBaseContext) GetCodecs() string {
	return c.codecs
}
