package dashreader

import (
	"fmt"
	"reflect"
)

//readerLiveMPDUpdate - Implement Reader of MPD
//  * Live
//  * MPD Updating
//  * SegmentTimeLine
//  * $Time$ based url
//  * $Number$ based url
type readerLiveMPDUpdate struct {
	readerBaseExtn
}

//MakeDASHReaderContext - Makes Reader Context
// Parameters:
//   1: Context received earlier... if first time pass nil
//   2: StreamSelector for the ContentType to select AdaptationSet
//   3: RepresentationSelector ... selector for Representation
// Return:
//   1: Context for current AdaptationSet,Representation
//   2: error
func (r *readerLiveMPDUpdate) MakeDASHReaderContext(rdrCtx ReaderContext, streamSelector StreamSelector, repSelector RepresentationSelector) (ReaderContext, error) {
	var curContext readerLiveMPDUpdateContext
	if rdrCtx != nil {
		curContext = rdrCtx.(readerLiveMPDUpdateContext)
	} else {
		curContext = readerLiveMPDUpdateContext{
			readerBaseContext: readerBaseContext{
				adaptSetID:     0,
				repID:          "",
				updCounter:     0,
				repSelector:    repSelector,
				streamSelector: streamSelector,
			},
		}
	}
	curMpd, updCounter := r.readerBaseExtn.checkUpdate()
	updateRequired := false
	if reflect.TypeOf(curContext.repSelector) != reflect.TypeOf(repSelector) {
		curContext.repSelector = repSelector
		updateRequired = true
	}
	if reflect.TypeOf(curContext.streamSelector) != reflect.TypeOf(streamSelector) {
		curContext.streamSelector = streamSelector
		updateRequired = true
	}
	if updateRequired {
		curContext.adjustRepUpdate(r.readerBase, curMpd)
	}
	if updCounter == curContext.updCounter {
		//no update
		return &curContext, nil
	}
	if rdrCtx == nil {
		//Incoming context is nil = new context
		//Locate the livePoint
		err := curContext.livePointLocate(r.readerBase, curMpd)
		if err != nil {
			//Don't return the newly created context
			return nil, fmt.Errorf("LivePoint Locate Failed: %w", err)
		}
	}
	return &curContext, nil
}
