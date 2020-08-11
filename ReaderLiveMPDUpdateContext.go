package dashreader

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eswarantg/statzagg"
)

//readerLiveMPDUpdateContext - readerLiveMPDUpdate Context for
type readerLiveMPDUpdateContext struct {
	readerBaseContext

	baseWcTime time.Time           //BaseTime for SegmentTimeline
	timeline   SegmentTimelineType //SegmentTimeline - Active
	timescale  uint                //timescale - Ticks per sec
	isNumber   bool                //Number pattern
	isTime     bool                //Time pattern
	initURL    url.URL             //url for init
	baseURL    url.URL             //Base url for chunk

	binitURLServed       bool   //init URL pending to be returned
	curSegTimeLineEntry  int    //cur entry to generate next url from
	curEntry             int    //index within cur entry to generate next url from
	chunkNumber          uint   //Actual number for $Number$ usecase
	chunkTimeTicks       uint64 //Actual number for $Time$ usecase
	elapsedDurationTicks uint64 //Duration elapsed till now
	startNumber          uint   //Number elapsed
}

func (c *readerLiveMPDUpdateContext) adjustRepUpdate(reader readerBase, curMpd *MPDtype) error {
	//Can be better ... for now recompute livePointLocate
	c.livePointLocate(reader, curMpd)
	return nil
}

//moveToWallClock - Usred to adjust to wallClock
// Set the context so that next URL fetch will return required values
func (c *readerLiveMPDUpdateContext) moveToNext(wallClock *time.Time) error {
	//c.baseWcTime - Reference Time
	//c.elapsedDurationTicks is record base time from baseTime
	//c.chunkTimeTicks is entry from base time for record
	//current Time
	//c.baseTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale))*time.Microsecond)
	for {
		//Check if within given timeline
		if c.curSegTimeLineEntry >= len(c.timeline.S) {
			if wallClock != nil {
				entryStartTime := c.baseWcTime.Add(time.Duration(float64(c.elapsedDurationTicks+c.chunkTimeTicks)*1000000/float64(c.timescale)) * time.Microsecond)
				log.Printf("%v Start%v <= WC:%v", c.readerBaseContext.ID, entryStartTime.UTC(), wallClock.UTC())
			}
			return fmt.Errorf("EndOfSegmentTimeline")
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
					values := make([]interface{}, 2)
					values[0] = c.elapsedDurationTicks
					values[1] = entry.T
					if c.readerBaseContext.StatzAgg != nil {
						(*c.readerBaseContext.StatzAgg).PostEventStats(context.TODO(), &statzagg.EventStats{
							EventClock: time.Now(),
							ID:         c.readerBaseContext.ID,
							Name:       EvtMp4DurationGapFilled,
							Values:     values,
						})
					}
					//log.Printf("%v Break in segment", g.id)
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
				log.Printf("%v WC:%v == Start:%v", c.readerBaseContext.ID, wallClock.UTC(), entryStartTime.UTC())
				//Found record
				return nil
			}
			if wallClock.Before(entryStartTime) {
				values := make([]interface{}, 2)
				values[0] = wallClock
				values[1] = entryStartTime
				if c.readerBaseContext.StatzAgg != nil {
					(*c.readerBaseContext.StatzAgg).PostEventStats(context.TODO(), &statzagg.EventStats{
						EventClock: time.Now(),
						ID:         c.readerBaseContext.ID,
						Name:       EvtMPDTimelineInFuture,
						Values:     values,
					})
				}
				return fmt.Errorf("Entry_in_future by %v (WC:%v, entryStart:%v)", entryStartTime.Sub(*wallClock), wallClock.UTC(), entryStartTime.UTC())
			}
			//Found Entry with Past Starttime
			//EndTime in Future
			entryEndTime := entryStartTime.Add(time.Duration(float64(entry.D)*1000000/float64(c.timescale)) * time.Microsecond)
			if wallClock.Equal(entryEndTime) || wallClock.Before(entryEndTime) {
				log.Printf("%v Start%v <= WC:%v <= End:%v", c.readerBaseContext.ID, entryStartTime.UTC(), wallClock.UTC(), entryEndTime.UTC())
				//Found record
				return nil
			}
			//log.Printf("%v Start%v End:%v <= WC:%v", c.readerBaseContext.ID, entryStartTime.UTC(), entryEndTime.UTC(), wallClock.UTC())
			c.curEntry++
			c.chunkTimeTicks += entry.D
			continue //Check next record
		}
		//Just next record
		c.curEntry++
		c.chunkTimeTicks += entry.D
		//log.Printf("%v Current moved to %v", c.readerBaseContext.ID, entryStartTime.UTC())
		//wallClock is not nil
		return nil
	}
}

//livePointLocate - Locate the Live Point in the Current MPD
// Set the context so that next URL fetch will return required values
func (c *readerLiveMPDUpdateContext) livePointLocate(reader readerBase, curMpd *MPDtype) error {
	pSwc := reader.baseTime
	pEwc := time.Time{}
	//Use PT as the base time to compute live point ref
	curWc := curMpd.PublishTime
	for _, period := range curMpd.Period {
		var v time.Duration
		if IsPresentDuration(period.Start) {
			// PSwc = lastPSwc + PS
			v, _ = ParseDuration(period.Start)
			pSwc = pSwc.Add(v)
		}
		if curWc.Before(pSwc) {
			return fmt.Errorf("No LivePoint Period found")
		}
		if curWc.After(pSwc) {
			if !IsPresentDuration(period.Duration) {
				//Found a Period with Start valid and no Duration record
				break
			}
			v, _ = ParseDuration(period.Duration)
			pEwc = pSwc.Add(v)
			if pEwc.Before(curWc) {
				//The entire period is before curWc
				continue
			}
		}
		//curWc.Equal(pSwc)
		//pSwc >= curWc  < pEwc
		if err := c.Select(period); err != nil {
			return fmt.Errorf("For Period(%v) No AdaptationSet selection found : %v", period.Id, err)
		}
		for _, adapt := range period.AdaptationSet {
			if adapt.Id != c.adaptSetID {
				continue
			}
			//Found matching AdaptationSet
			for _, rp := range adapt.Representation {
				//Found matching Representation
				if rp.Id != c.repID {
					continue
				}
				//Initialize the values
				c.baseWcTime = pSwc
				//Offset any PresentationTimeOffset
				if rp.SegmentBase.PresentationTimeOffset > 0 {
					c.baseWcTime = c.baseWcTime.Add(-1 * time.Duration(float64(rp.SegmentBase.PresentationTimeOffset)*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond)
				}
				if adapt.SegmentTemplate.PresentationTimeOffset > 0 {
					c.baseWcTime = c.baseWcTime.Add(-1 * time.Duration(float64(adapt.SegmentTemplate.PresentationTimeOffset)*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond)
				}
				//Offset any AvailabilityTimeOffset
				if adapt.SegmentTemplate.AvailabilityTimeOffset > 0 {
					c.baseWcTime = c.baseWcTime.Add(time.Duration(adapt.SegmentTemplate.AvailabilityTimeOffset*1000000/float64(adapt.SegmentTemplate.Timescale)) * time.Microsecond)
				}
				c.timeline = adapt.SegmentTemplate.SegmentTimeline
				c.timescale = adapt.SegmentTemplate.Timescale

				c.isNumber = reader.isNumber
				c.isTime = reader.isTime

				if len(adapt.SegmentTemplate.Initialization.SourceURL) > 0 {
					c.initURL = url.URL{} //TBD
					c.binitURLServed = true
				} else {
					c.binitURLServed = false
				}
				c.baseURL = url.URL{} //TBD
				c.curSegTimeLineEntry = 0
				c.curEntry = 0
				c.chunkNumber = 0
				c.chunkTimeTicks = 0
				c.elapsedDurationTicks = 0
				c.startNumber = adapt.SegmentTemplate.StartNumber
				return c.moveToNext(&curWc)
			}
		}
		return fmt.Errorf("Representation(%v:%v) not found", c.adaptSetID, c.repID)
	}
	return nil
}

//getURL - Returns current URL
//Return:
//  1: Current available url
//  2: Error=io.EOF if not present
func (c *readerLiveMPDUpdateContext) getURL() (ret *S, err error) {
	//Check if within given timeline
	if c.curSegTimeLineEntry >= len(c.timeline.S) {
		return nil, io.EOF
	}
	//If entry is nil... ignore move to next entry
	entry := c.timeline.S[c.curSegTimeLineEntry]
	//Assume repeat = 1
	repeatCount := 1
	//number of entry = given value + 1
	repeatCount = entry.R + 1
	//Check if we are done with all entries
	if c.curEntry >= repeatCount {
		return nil, io.EOF
	}
	return &c.timeline.S[c.curSegTimeLineEntry], nil
}

//NextURL -
//-- Once end is reached (io.EOF)
//-- MakeDASHReaderContext has to be called again
// Parameters;
//   None
// Return:
//   1: Next URL
//   2: error
func (c readerLiveMPDUpdateContext) NextURL() (ret ChunkURL, err error) {
	var entry *S
	if !c.binitURLServed {
		c.binitURLServed = true
		ret.ChunkURL = c.initURL
		ret.Duration = 0
		ret.FetchAt = time.Now()
	}
	entry, err = c.getURL()
	if err != nil {
		return
	}
	if c.isNumber {
		ret.ChunkURL = c.baseURL
		ret.Duration = time.Duration(float64(entry.D)*1000000/float64(c.timescale)) * time.Microsecond
		ret.ChunkURL.Path = strings.ReplaceAll(ret.ChunkURL.Path, "$Number$", strconv.FormatInt(int64(c.chunkNumber+c.startNumber), 10))
		ret.ChunkURL.Path = strings.ReplaceAll(ret.ChunkURL.Path, "$Time$", strconv.FormatUint((c.elapsedDurationTicks+c.chunkTimeTicks), 10))
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
