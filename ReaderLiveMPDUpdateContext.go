package dashreader

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eswarantg/statzagg"
)

const (
	livePointOK          = 0
	livePointNoEntry     = 1
	livePointFutureEntry = 2
)

type livePointErr struct {
	errType int
	err     error
}

func (e livePointErr) Error() string {
	return e.err.Error()
}

//readerLiveMPDUpdateContext - readerLiveMPDUpdate Context for
type readerLiveMPDUpdateContext struct {
	readerBaseContext

	baseWcTime time.Time           //BaseTime for SegmentTimeline
	timeline   SegmentTimelineType //SegmentTimeline - Active
	timescale  uint                //timescale - Ticks per sec
	isNumber   bool                //Number pattern
	isTime     bool                //Time pattern
	initURL    url.URL             //url for init
	initRange  string              //range Header for init
	baseURL    url.URL             //Base url for chunk

	binitURLServed       bool   //init URL pending to be returned
	curSegTimeLineEntry  int    //cur entry to generate next url from
	curEntry             int    //index within cur entry to generate next url from
	chunkNumber          uint   //Actual number for $Number$ usecase
	chunkTimeTicks       uint64 //Actual number for $Time$ usecase
	elapsedDurationTicks uint64 //Duration elapsed till now
	startNumber          uint   //Number elapsed
}

//moveToWallClock - Usred to adjust to wallClock
// Set the context so that next URL fetch will return required values
func (c *readerLiveMPDUpdateContext) moveToNext(wallClock *time.Time) livePointErr {
	if wallClock != nil {
		//log.Printf("moveToNext %v", wallClock.UTC())
	}
	//c.baseWcTime - Reference Time
	//c.elapsedDurationTicks is record base time from baseTime
	//c.chunkTimeTicks is entry from base time for record
	//current Time
	//c.baseTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale))*time.Microsecond)
	for {
		//Check if within given timeline
		if c.curSegTimeLineEntry >= len(c.timeline.S) {
			//Segment Timeline end reached
			if wallClock != nil {
				//If reference wallclock is present
				entryStartTime := c.baseWcTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale)) * time.Microsecond)
				if c.StatzAgg != nil {
					values := make([]interface{}, 2)
					values[0] = wallClock
					values[1] = entryStartTime
					c.StatzAgg.PostEventStats(context.TODO(), &statzagg.EventStats{
						EventClock: time.Now(),
						ID:         c.ID,
						Name:       EvtMPDTimelineNoLivePointEntries,
						Values:     values,
					})
				}
				//log.Printf("%v Start%v < WC:%v", c.ID, entryStartTime.UTC(), wallClock.UTC())
			}
			return livePointErr{livePointNoEntry, fmt.Errorf("EndOfSegmentTimeline: %w", io.EOF)}
		}
		//If entry is nil... ignore move to next entry
		entry := c.timeline.S[c.curSegTimeLineEntry]
		//Assume repeat = 1
		repeatCount := 1
		//If repeatCount is present
		//number of entry = given value + 1
		repeatCount = entry.R + 1
		//Check if we are done with all entries
		if c.curEntry >= repeatCount {
			//if done... move to next record
			c.curSegTimeLineEntry++
			//start from first entry in the record set to ZERO
			c.curEntry = 0
			//Overall elapsed ... include entry cumulative time
			c.elapsedDurationTicks += c.chunkTimeTicks
			//Reset chunk elapsed to ZERO
			c.chunkTimeTicks = 0
			continue
		}
		//At start of the record if start time is given...
		if entry.T != 0 && c.curEntry == 0 {
			//Check if the computed elapsedDuration and startTime match
			if entry.T > c.elapsedDurationTicks {
				//Time line break... advance to new time
				c.elapsedDurationTicks = entry.T
			}
			if c.elapsedDurationTicks != entry.T {
				//Check if this is the first entry
				if c.curEntry != 0 || c.curSegTimeLineEntry != 0 {
					//if diff found... break in timeline
					if c.StatzAgg != nil {
						values := make([]interface{}, 2)
						values[0] = c.elapsedDurationTicks
						values[1] = entry.T
						c.StatzAgg.PostEventStats(context.TODO(), &statzagg.EventStats{
							EventClock: time.Now(),
							ID:         c.ID,
							Name:       EvtMPDTimelineGapFilled,
							Values:     values,
						})
					}
					//log.Printf("%v Break in segment", c.ID)
				}
			}
			//TBD - check if timeline is not reversed
			c.elapsedDurationTicks = entry.T
		}
		//We have current entry
		entryStartTime := c.baseWcTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale)) * time.Microsecond)
		if wallClock != nil {
			//StartTime matches
			if wallClock.Equal(entryStartTime) {
				//log.Printf("%v WC:%v == Start:%v", c.ID, wallClock.UTC(), entryStartTime.UTC())
				//Found record
				return livePointErr{livePointOK, nil}
			}
			if wallClock.Before(entryStartTime) {
				if c.StatzAgg != nil {
					values := make([]interface{}, 2)
					values[0] = wallClock
					values[1] = entryStartTime
					c.StatzAgg.PostEventStats(context.TODO(), &statzagg.EventStats{
						EventClock: time.Now(),
						ID:         c.ID,
						Name:       EvtMPDTimelineInFuture,
						Values:     values,
					})
				}
				return livePointErr{livePointFutureEntry, fmt.Errorf("Entry_in_future by %v (WC:%v, entryStart:%v)", entryStartTime.Sub(*wallClock), wallClock.UTC(), entryStartTime.UTC())}
			}
			//Found Entry with Past Starttime
			//EndTime in Future
			entryEndTime := entryStartTime.Add(time.Duration(float64(entry.D)*1000000/float64(c.timescale)) * time.Microsecond)
			if wallClock.Equal(entryEndTime) || wallClock.Before(entryEndTime) {
				//log.Printf("%v Start%v <= WC:%v <= End:%v", c.ID, entryStartTime.UTC(), wallClock.UTC(), entryEndTime.UTC())
				//Found record
				return livePointErr{livePointOK, nil}
			}
			//log.Printf("%v Start%v <= End:%v <= WC:%v", c.ID, entryStartTime.UTC(), entryEndTime.UTC(), wallClock.UTC())
			c.curEntry++
			c.chunkTimeTicks += entry.D
			continue //Check next record
		}
		//Just next record
		c.curEntry++
		c.chunkTimeTicks += entry.D
		//log.Printf("%v Current moved to %v", c.ID, entryStartTime.UTC())
		//wallClock is not nil
		return livePointErr{livePointOK, nil}
	}
}

func (c *readerLiveMPDUpdateContext) getActivePeriod(reader readerBase, curMpd *MPDtype) (*PeriodType, time.Time) {
	pSwc := reader.baseTime
	//log.Printf("Base pSwc : %v", pSwc.UTC())
	pEwc := time.Time{}
	//Use PT as the base time to compute live point ref
	curWc := curMpd.PublishTime
	//log.Printf("PublishTime : %v", curWc.UTC())
	for _, period := range curMpd.Period {
		var v time.Duration
		if IsPresentDuration(period.Start) {
			// PSwc = lastPSwc + PS
			v, _ = ParseDuration(period.Start)
			pSwc = pSwc.Add(v)
			//log.Printf("New pSwc : %v", pSwc.UTC())
		}
		if curWc.Before(pSwc) {
			//log.Printf("WallClock (%v) < Period Start (%v)", curWc.UTC(), pSwc.UTC())
			break
		}
		if curWc.After(pSwc) {
			if IsPresentDuration(period.Duration) {
				v, _ = ParseDuration(period.Duration)
				pEwc = pSwc.Add(v)
				//log.Printf("New pEwc : %v", pEwc.UTC())
				if pEwc.Before(curWc) {
					//The entire period is before curWc
					continue
				}
			}
		}
		//curWc.Equal(pSwc)
		//pSwc >= curWc && No Duration present
		//pSwc >= curWc  < pEwc
		return &period, pSwc
	}
	if c.StatzAgg != nil {
		values := make([]interface{}, 1)
		values[0] = len(curMpd.Period)
		c.StatzAgg.PostEventStats(context.TODO(), &statzagg.EventStats{
			EventClock: time.Now(),
			ID:         c.ID,
			Name:       EvtMPDNoActivePeriod,
			Values:     values,
		})
	}
	return nil, pSwc
}

//adjustRepUpdate - Handle Rep update
func (c *readerLiveMPDUpdateContext) adjustRepUpdate(reader readerBase, curMpd *MPDtype) error {
	var periodBaseURL url.URL
	periodBaseURL = reader.baseURL

	//Use PT as the base time to compute live point ref
	curWc := curMpd.PublishTime
	//log.Printf("PublishTime : %v", curWc.UTC())
	period, pSwc := c.getActivePeriod(reader, curMpd)
	if period == nil {
		return fmt.Errorf("Unable to find Active Period")
	}

	tURL, err := AdjustURLPath(periodBaseURL, period.BaseURL, "")
	if err != nil {
		return fmt.Errorf("Adjusting to Period(%v) BaseURL has error: %v", period.Id, err)
	}
	periodBaseURL = *tURL
	for _, adapt := range period.AdaptationSet {
		if adapt.ContentType != c.streamSelector.ContentType || adapt.Id != c.adaptSetID {
			continue
		}
		//Found matching AdaptationSet
		tURL, err := AdjustURLPath(periodBaseURL, adapt.BaseURL, "")
		if err != nil {
			return fmt.Errorf("Adjusting to Period(%v) BaseURL has error: %v", period.Id, err)
		}
		adaptBaseURL := *tURL
		for _, rp := range adapt.Representation {
			if rp.Id != c.repID {
				continue
			}
			//Found matching Representation
			tURL, err := AdjustURLPath(adaptBaseURL, rp.BaseURL, "")
			if err != nil {
				return fmt.Errorf("Adjusting to Period(%v) BaseURL has error: %v", period.Id, err)
			}
			rpBaseURL := *tURL
			baseWcTime := pSwc
			//Offset any PresentationTimeOffset
			if rp.SegmentBase.PresentationTimeOffset > 0 {
				baseWcTime = baseWcTime.Add(-1 * time.Duration(float64(rp.SegmentBase.PresentationTimeOffset)*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond)
			}
			if adapt.SegmentTemplate.PresentationTimeOffset > 0 {
				baseWcTime = baseWcTime.Add(-1 * time.Duration(float64(adapt.SegmentTemplate.PresentationTimeOffset)*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond)
			}
			//Offset any AvailabilityTimeOffset
			if adapt.SegmentTemplate.AvailabilityTimeOffset > 0 {
				baseWcTime = baseWcTime.Add(time.Duration(adapt.SegmentTemplate.AvailabilityTimeOffset*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond)
			}
			//Check if BaseWCTime is not modified
			if baseWcTime != c.baseWcTime {
				return fmt.Errorf("BaseTime mismatch (%v,%v,%v) C %v != Wc %v)", period.Id, adapt.Id, rp.Id, c.baseWcTime, baseWcTime)
			}
			//Nothing has changed
			//update only required field
			if len(adapt.SegmentTemplate.Initialization.SourceURL) > 0 {
				temp := strings.ReplaceAll(adapt.SegmentTemplate.Initialization.SourceURL, "$RepresentationID$", string(rp.Id))
				v, err := AdjustURLPath(rpBaseURL, []BaseURLType{}, temp)
				if err != nil {
					return fmt.Errorf("Adjusting to Representation(%v) BaseURL has error: %v", rp.Id, err)
				}
				c.initURL = *v
				c.initRange = adapt.SegmentTemplate.Initialization.Range
				c.binitURLServed = false //to be supplied
			} else {
				c.initURL = url.URL{}
				c.initRange = ""
				c.binitURLServed = true //mark already supplied so that it is not done
			}
			temp := strings.ReplaceAll(adapt.SegmentTemplate.Media, "$RepresentationID$", string(rp.Id))
			v, err := AdjustURLPath(rpBaseURL, []BaseURLType{}, temp)
			if err != nil {
				return fmt.Errorf("Adjusting to Representation(%v) BaseURL has error: %v", rp.Id, err)
			}
			entryStartTime := c.baseWcTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale)) * time.Microsecond)
			c.timeline = adapt.SegmentTemplate.SegmentTimeline
			c.baseURL = *v
			c.curSegTimeLineEntry = 0
			c.curEntry = 0
			c.chunkNumber = 0
			c.chunkTimeTicks = 0
			c.elapsedDurationTicks = 0
			c.startNumber = adapt.SegmentTemplate.StartNumber
			if entryStartTime.Before(curWc) {
				entryStartTime = entryStartTime.UTC().Add(1 * time.Microsecond)
				//log.Printf("Move (%v) To WallClock entryStartTime %v Begin %v", c.ID, curWc.UTC(), entryStartTime)
				curWc = entryStartTime
			}
			livePointErr := c.moveToNext(&curWc)
			//log.Printf("Update %v %v", livePointErr.errType, livePointErr.err)
			switch livePointErr.errType {
			case livePointNoEntry:
				//No Entry ... update without new Timeline addition
				livePointErr.err = nil
			case livePointFutureEntry:
				//Entry in future
				livePointErr.err = nil
			}
			return livePointErr.err
		}
	}
	return fmt.Errorf("Representation(%v:%v) not found", c.adaptSetID, c.repID)
}

//livePointLocate - Locate the Live Point in the Current MPD
// Set the context so that next URL fetch will return required values
func (c *readerLiveMPDUpdateContext) livePointLocate(reader readerBase, curMpd *MPDtype) error {
	var periodBaseURL url.URL
	periodBaseURL = reader.baseURL

	//Use PT as the base time to compute live point ref
	curWc := curMpd.PublishTime
	//log.Printf("PublishTime : %v", curWc.UTC())

	period, pSwc := c.getActivePeriod(reader, curMpd)
	if period == nil {
		return fmt.Errorf("Unable to find Active Period")
	}
	if err := c.Select(*period); err != nil {
		return fmt.Errorf("For Period(%v) No AdaptationSet selection found : %v", period.Id, err)
	}
	tURL, err := AdjustURLPath(periodBaseURL, period.BaseURL, "")
	if err != nil {
		return fmt.Errorf("Adjusting to Period(%v) BaseURL has error: %v", period.Id, err)
	}
	periodBaseURL = *tURL
	for _, adapt := range period.AdaptationSet {
		if adapt.ContentType != c.streamSelector.ContentType || adapt.Id != c.adaptSetID {
			continue
		}
		//Found matching AdaptationSet
		tURL, err := AdjustURLPath(periodBaseURL, adapt.BaseURL, "")
		if err != nil {
			return fmt.Errorf("Adjusting to Period(%v) BaseURL has error: %v", period.Id, err)
		}
		adaptBaseURL := *tURL
		for _, rp := range adapt.Representation {
			if rp.Id != c.repID {
				continue
			}
			//Found matching Representation
			tURL, err := AdjustURLPath(adaptBaseURL, rp.BaseURL, "")
			if err != nil {
				return fmt.Errorf("Adjusting to Period(%v) BaseURL has error: %v", period.Id, err)
			}
			rpBaseURL := *tURL

			//Initialize the values
			c.baseURL = rpBaseURL
			c.timescale = adapt.SegmentTemplate.Timescale

			c.baseWcTime = pSwc
			//log.Printf("baseWcTime : %v", c.baseWcTime.UTC())
			//Offset any PresentationTimeOffset
			if rp.SegmentBase.PresentationTimeOffset > 0 {
				d := time.Duration(float64(rp.SegmentBase.PresentationTimeOffset)*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond
				c.baseWcTime = c.baseWcTime.Add(-1 * d)
				//log.Printf("baseWcTime + SB.PresentationTimeOffset (%v): %v", d, c.baseWcTime.UTC())
			}
			if adapt.SegmentTemplate.PresentationTimeOffset > 0 {
				d := time.Duration(float64(adapt.SegmentTemplate.PresentationTimeOffset)*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond
				c.baseWcTime = c.baseWcTime.Add(-1 * d)
				//log.Printf("baseWcTime + ST.PresentationTimeOffset (%v): %v", d, c.baseWcTime.UTC())
			}
			//Offset any AvailabilityTimeOffset
			if adapt.SegmentTemplate.AvailabilityTimeOffset > 0 {
				d := time.Duration(adapt.SegmentTemplate.AvailabilityTimeOffset*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond
				c.baseWcTime = c.baseWcTime.Add(d)
				//log.Printf("baseWcTime + ST.PresentationTimeOffset (%v): %v", d, c.baseWcTime.UTC())
			}
			c.timeline = adapt.SegmentTemplate.SegmentTimeline
			c.isNumber = reader.isNumber
			c.isTime = reader.isTime
			if len(adapt.SegmentTemplate.Initialization.SourceURL) > 0 {
				temp := strings.ReplaceAll(adapt.SegmentTemplate.Initialization.SourceURL, "$RepresentationID$", string(rp.Id))
				v, err := AdjustURLPath(rpBaseURL, []BaseURLType{}, temp)
				if err != nil {
					return fmt.Errorf("Adjusting to Representation(%v) BaseURL has error: %v", rp.Id, err)
				}
				c.initURL = *v
				c.initRange = adapt.SegmentTemplate.Initialization.Range
				c.binitURLServed = false //to be supplied
			} else {
				c.initURL = url.URL{}
				c.initRange = ""
				c.binitURLServed = true //mark already supplied so that it is not done
			}
			temp := strings.ReplaceAll(adapt.SegmentTemplate.Media, "$RepresentationID$", string(rp.Id))
			v, err := AdjustURLPath(rpBaseURL, []BaseURLType{}, temp)
			if err != nil {
				return fmt.Errorf("Adjusting to Representation(%v) BaseURL has error: %v", rp.Id, err)
			}
			c.baseURL = *v
			c.curSegTimeLineEntry = 0
			c.curEntry = 0
			c.chunkNumber = 0
			c.chunkTimeTicks = 0
			c.elapsedDurationTicks = 0
			c.startNumber = adapt.SegmentTemplate.StartNumber
			livePointErr := c.moveToNext(&curWc)
			return livePointErr.err
		}
	}
	return fmt.Errorf("Representation(%v:%v) not found", c.adaptSetID, c.repID)
}

//getURL - Returns current URL
//Return:
//  1: Current available url
//  2: Error=io.EOF if not present
func (c readerLiveMPDUpdateContext) getURL() (ret *S, err error) {
	//Check if within given timeline
	if c.curSegTimeLineEntry >= len(c.timeline.S) {
		return nil, io.EOF
	}
	//If entry is nil... ignore move to next entry
	entry := c.timeline.S[c.curSegTimeLineEntry]
	//number of entry = given value + 1
	repeatCount := entry.R + 1
	//Check if we are done with all entries
	if c.curEntry >= repeatCount {
		return nil, io.EOF
	}
	return &c.timeline.S[c.curSegTimeLineEntry], nil
}

//NextURLs - Get URLs from Current MPD context
//-- Once end of this list is reached
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   context for cancellation
// Return:
//   1: Channel of URLs, can be read till closed
//   2: error
func (c *readerLiveMPDUpdateContext) NextURLs(ctx context.Context) (ret <-chan ChunkURL, err error) {
	return c.getURLs(ctx, ReaderContext(c))
}

//NextURL -
//-- Once end is reached (io.EOF)
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   None
// Return:
//   1: Next URL
//   2: error
func (c *readerLiveMPDUpdateContext) NextURL() (*ChunkURL, error) {
	return c.nextURL()
}

//NextURL -
//-- Once end is reached (io.EOF)
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   None
// Return:
//   1: Next URL
//   2: error
func (c *readerLiveMPDUpdateContext) nextURL() (ret *ChunkURL, err error) {
	var entry *S
	ret = nil
	err = nil
	if !c.binitURLServed {
		ret = &ChunkURL{}
		c.binitURLServed = true
		ret.ChunkURL = c.initURL
		ret.Range = c.initRange
		ret.Duration = 0
		ret.FetchAt = time.Now()
		return
	}
	entry, err = c.getURL()
	if err != nil {
		return
	}
	ret = &ChunkURL{}
	if c.isNumber {
		ret.ChunkURL = c.baseURL
		ret.Duration = time.Duration(float64(entry.D)*1000000/float64(c.timescale)) * time.Microsecond
		ret.ChunkURL.Path = strings.ReplaceAll(ret.ChunkURL.Path, "$Number$", strconv.FormatInt(int64(c.chunkNumber+c.startNumber), 10))
		ret.FetchAt = c.baseWcTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale)) * time.Microsecond)
	}
	if c.isTime {
		ret.ChunkURL = c.baseURL
		ret.Duration = time.Duration(float64(entry.D)*1000000/float64(c.timescale)) * time.Microsecond
		ret.ChunkURL.Path = strings.ReplaceAll(ret.ChunkURL.Path, "$Time$", strconv.FormatUint((c.elapsedDurationTicks+c.chunkTimeTicks), 10))
		ret.FetchAt = c.baseWcTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale)) * time.Microsecond)
	}
	c.moveToNext(nil)
	return
}
