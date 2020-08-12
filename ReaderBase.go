package dashreader

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

//readerBase - Fixed values created first time
type readerBase struct {
	ID       string    //ID for the Reader
	baseTime time.Time //WallClock time of start of period
	baseURL  url.URL   //Base URL
	isNumber bool      //Number pattern
	isTime   bool      //Time pattern
}

//readerBaseExtn - Base functionality for all dash readers
type readerBaseExtn struct {
	readerBase
	mutex      sync.RWMutex //Mutex to gaurd updCounter, curMpd, nextMpd
	updCounter int64        //to sync between Context and Reader
	curMpd     *MPDtype     //Current MPD
	lastMpd    *MPDtype     //Next MPD on update
}

//checkUpdate - Invoked by Client to
func (r *readerBaseExtn) checkUpdate() (*MPDtype, int64) {
	//Allow for parallel read and serialized writes
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.curMpd, r.updCounter
}

//Update - Update the MPD content
// Parameters:
//   MPD read
// Return:
//   1: MPD Updated - PublishTime Updated?
//   2: New Period  - NewPeriod Updated?
//   3: error
func (r *readerBaseExtn) Update(newMpd *MPDtype) (bool, bool, error) {
	if !IsPresentTime(newMpd.PublishTime) {
		return false, false, fmt.Errorf("MPD.PublishTime MUST be present")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.curMpd != nil {
		if r.curMpd.PublishTime.Equal(newMpd.PublishTime) {
			return false, false, nil
		}
		if r.curMpd.PublishTime.After(newMpd.PublishTime) {
			return false, false, fmt.Errorf("MPD.PublishTime MUST move forward. Ignoring")
		}
		//TBD - Period Update check
	}
	r.lastMpd = r.curMpd
	r.curMpd = newMpd
	r.updCounter++
	return true, false, nil
}

//MakeDASHReaderContext - Makes Reader Context
// Parameters:
//   1: Context received earlier... if first time pass nil
//   2: StreamSelector for the ContentType to select AdaptationSet
//   3: RepresentationSelector ... selector for Representation
// Return:
//   1: Context for current AdaptationSet,Representation
//   2: error
func (r *readerBaseExtn) MakeDASHReaderContext(*ReaderContext, StreamSelector, RepresentationSelector) (ReaderContext, error) {
	return nil, fmt.Errorf("readerBaseExtn MakeDASHReaderContext NOT IMPLEMENTED")
}
