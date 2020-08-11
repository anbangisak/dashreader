package dashreader

import (
	"fmt"
	"reflect"
)

//DASHReaderLiveMPDUpdate - Implement Reader of MPD
//  * Live
//  * MPD Updating
//  * SegmentTimeLine
//  * $Time$ based url
//  * $Number$ based url
type DASHReaderLiveMPDUpdate struct {
	DASHReaderBaseExtn
}

//MakeDASHReaderContext - Makes Reader Context
// Parameters:
//   1: Context received earlier... if first time pass nil
//   2: StreamSelector for the ContentType to select AdaptationSet
//   3: RepresentationSelector ... selector for Representation
// Return:
//   1: Context for current AdaptationSet,Representation
//   2: error
func (r *DASHReaderLiveMPDUpdate) MakeDASHReaderContext(rdrCtx DASHReaderContext, streamSelector StreamSelector, repSelector RepresentationSelector) (DASHReaderContext, error) {
	var curContext DASHReaderLiveMPDUpdateContext
	if rdrCtx != nil {
		curContext = rdrCtx.(DASHReaderLiveMPDUpdateContext)
	} else {
		curContext = DASHReaderLiveMPDUpdateContext{
			DASHReaderBaseContext: DASHReaderBaseContext{
				adaptSetID:     0,
				repID:          "",
				updCounter:     0,
				repSelector:    repSelector,
				streamSelector: streamSelector,
			},
		}
	}
	curMpd, updCounter := r.DASHReaderBaseExtn.checkUpdate()
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
		curContext.adjustRepUpdate(r.DASHReaderBase, curMpd)
	}
	if updCounter == curContext.updCounter {
		//no update
		return &curContext, nil
	}
	if rdrCtx == nil {
		//Incoming context is nil = new context
		//Locate the livePoint
		err := curContext.livePointLocate(r.DASHReaderBase, curMpd)
		if err != nil {
			//Don't return the newly created context
			return nil, fmt.Errorf("LivePoint Locate Failed: %w", err)
		}
	}
	return &curContext, nil
}
