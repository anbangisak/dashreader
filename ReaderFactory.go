package dashreader

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	relativeBasePathURL = "./"
)

//ReaderFactory - ReaderFactory of DASHReaders
type ReaderFactory struct {
	//IsLive - Live content?
	IsLive bool
	//AST - Availablity Start Time - WallClock reference
	AST time.Time
	//PT - Publish Time - WallClock when MPD wa updated
	PT time.Time
	//MUP - Minimum Update Period - Polling interval for Manifest update
	MUP time.Duration
	//MBT - Minimum Buffer Period - Client consideratino for cache level
	MBT time.Duration
	//MPD - Media Presentation Duration - Presentation Duration
	MPD time.Duration
	//TSB - Total available duration of content
	TSB time.Duration
	//Current Period Start
	curPeriodStart time.Time
	//Current Period
	curPeriod *PeriodType
	//Base URL
	baseURL url.URL
	//SegmentTimeLine Present
	isSegmentTimeline *bool
	//$Time$based ?
	isTimeBased *bool
}

//GetDASHReader - Depending on the MPD contents find the right reader
func (f *ReaderFactory) GetDASHReader(ID string, mpdURL string, mpd *MPDtype) (Reader, error) {
	baseURL, err := url.Parse(mpdURL)
	if err != nil {
		return nil, fmt.Errorf("Supplied mpdURL(%v) not correct: %w", mpdURL, err)
	}
	baseURL, err = AdjustURLPath(*baseURL, mpd.BaseURL, relativeBasePathURL)
	if err != nil {
		return nil, fmt.Errorf("MPD.%w", err)
	}
	f.baseURL = *baseURL
	//Validate and read the fields to understand the type of MPD
	err = f.validate(mpd)
	if err != nil {
		return nil, err
	}
	//Build a Reader and respond
	return f.makeDASHReader(ID, mpd)
}

func (f *ReaderFactory) validateMpdDurationFields(mpd *MPDtype) error {
	var err error
	var v time.Duration
	if len(mpd.MinBufferTime) > 0 {
		if v, err = ParseDuration(mpd.MinBufferTime); err != nil {
			return fmt.Errorf("MPD.MinBufferTime (\"%v\") MUST be valid : %w", mpd.MinBufferTime, err)
		}
		f.MBT = v
	}
	if len(mpd.TimeShiftBufferDepth) > 0 {
		if v, err = ParseDuration(mpd.TimeShiftBufferDepth); err != nil {
			return fmt.Errorf("MPD.TimeShiftBufferDepth (\"%v\") MUST be valid : %w", mpd.TimeShiftBufferDepth, err)
		}
		f.TSB = v
	}
	if len(mpd.SuggestedPresentationDelay) > 0 {
		if _, err = ParseDuration(mpd.SuggestedPresentationDelay); err != nil {
			return fmt.Errorf("MPD.SuggestedPresentationDelay (\"%v\") MUST be valid : %w", mpd.SuggestedPresentationDelay, err)
		}
	}
	if len(mpd.MaxSegmentDuration) > 0 {
		if _, err = ParseDuration(mpd.MaxSegmentDuration); err != nil {
			return fmt.Errorf("MPD.MaxSegmentDuration (\"%v\") MUST be valid : %w", mpd.MaxSegmentDuration, err)
		}
	}
	if len(mpd.MaxSubsegmentDuration) > 0 {
		if _, err = ParseDuration(mpd.MaxSubsegmentDuration); err != nil {
			return fmt.Errorf("MPD.MaxSubsegmentDuration (\"%v\") MUST be valid : %w", mpd.MaxSubsegmentDuration, err)
		}
	}
	if len(mpd.MediaPresentationDuration) > 0 {
		if v, err = ParseDuration(mpd.MediaPresentationDuration); err != nil {
			return fmt.Errorf("MPD.MediaPresentationDuration (\"%v\") MUST be valid : %w", mpd.MediaPresentationDuration, err)
		}
		f.MPD = v
	}
	if len(mpd.MinimumUpdatePeriod) > 0 {
		if v, err = ParseDuration(mpd.MinimumUpdatePeriod); err != nil {
			return fmt.Errorf("MPD.MinimumUpdatePeriod (\"%v\") MUST be valid : %w", mpd.MinimumUpdatePeriod, err)
		}
		f.MUP = v
	}
	for _, period := range mpd.Period {
		err = f.validatePeriodDurationFields(&period)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *ReaderFactory) validatePeriodDurationFields(period *PeriodType) error {
	var err error
	if len(period.Start) > 0 {
		if _, err = ParseDuration(period.Start); err != nil {
			return fmt.Errorf("period.Start (\"%v\") MUST be valid : %w", period.Start, err)
		}
	}
	if len(period.Duration) > 0 {
		if _, err = ParseDuration(period.Duration); err != nil {
			return fmt.Errorf("period.Duration (\"%v\") MUST be valid : %w", period.Duration, err)
		}
	}
	return nil
}

//Validate - Validates the MPD for various fields
func (f *ReaderFactory) validate(mpd *MPDtype) error {
	var err error
	//Validate the Duration Fields
	err = f.validateMpdDurationFields(mpd)
	if err != nil {
		return err
	}
	if !IsPresentTime(mpd.PublishTime) {
		return fmt.Errorf("MPD.PublishTime MUST be present")
	}
	f.PT = mpd.PublishTime
	switch mpd.Type {
	case "static":
		err = f.validateStatic(mpd)
	case "dynamic":
		err = f.validateDynamicMpd(mpd)
	default:
		err = fmt.Errorf("MPD.Type MUST be (static or dynamic)")
	}
	return err
}

func (f *ReaderFactory) validateDynamicMpd(mpd *MPDtype) error {
	mpdUpdateMode := true
	// Value of these fields won't change
	if !strings.Contains(mpd.Profiles, LiveProfile) {
		return fmt.Errorf("MPD.Profile (\"%v\") MUST include \"%v\" for MPD.Type=\"dynamic\"", mpd.Profiles, LiveProfile)
	}
	if !IsPresentTime(mpd.AvailabilityStartTime) {
		return fmt.Errorf("MPD.AvailabilityStartTime (\"%v\") MUST be present", mpd.AvailabilityStartTime)
	}
	f.AST = mpd.AvailabilityStartTime
	if len(mpd.Period) <= 0 {
		return fmt.Errorf("MPD.Period atleast ONE is required")
	}
	if !IsPresentDuration(mpd.MinimumUpdatePeriod) {
		f.MUP, _ = ParseDuration(mpd.MinimumUpdatePeriod)
		mpdUpdateMode = false
	}
	lastPeriodStart := f.AST
	var lastPeriodDuration *time.Duration
	for i, period := range mpd.Period {
		var pSwc time.Time
		if !IsPresentDuration(period.Start) {
			//Period.Start is MUST for Period 1
			if i == 1 {
				return fmt.Errorf("Period.Start MUST be present for first period")
			}
			if lastPeriodDuration == nil {
				panic("lastPeriodDuration not known in second period")
			}
			pSwc = lastPeriodStart.Add(*lastPeriodDuration)
		} else {
			// PSwc = lastPSwc + PS
			v, _ := ParseDuration(period.Start)
			pSwc = lastPeriodStart.Add(v)
		}
		if !IsPresentDuration(period.Duration) {
			//Without MPD update
			if !mpdUpdateMode {
				if i == len(mpd.Period)-1 {
					//Either Period.Duration or mpd.MediaPresentationDuration is MUST
					if len(mpd.MediaPresentationDuration) <= 0 {
						return fmt.Errorf("Period.Duration for Last Period or MPD.MediaPresentationDuration MUST be present")
					}
				}
			}
			lastPeriodDuration = nil
		} else {
			v, _ := ParseDuration(period.Duration)
			lastPeriodDuration = &v

		}
		for _, adaptSet := range period.AdaptationSet {
			err := f.validateDynamicAdaptSet(pSwc, &lastPeriodDuration, &adaptSet)
			if err != nil {
				return err
			}
		}
	}
	f.IsLive = true
	return nil
}

func (f *ReaderFactory) validateDynamicAdaptSet(periodStart time.Time, periodDuration **time.Duration, adaptSet *AdaptationSetType) error {
	timeScalePresent := false
	segTimelinePresent := false
	durationPresent := false
	numberBased := false
	timeBased := false
	if GetBoolFromConditionalUintType(adaptSet.SegmentAlignment) == false {
		return fmt.Errorf("AdapatationSet (%v) SegmentAlignment MUST be \"true\"", adaptSet.Id)
	}
	segTemplate := adaptSet.SegmentTemplate
	if len(segTemplate.Media) <= 0 {
		return fmt.Errorf("AdapatationSet (%v) SegmentTemplate.Media MUST be present", adaptSet.Id)
	}
	if len(segTemplate.Media) > 0 {
		if strings.Contains(segTemplate.Media, TimeToken) {
			timeBased = true
		}
		if strings.Contains(segTemplate.Media, NumberToken) {
			numberBased = true
		}
		if timeBased == false && numberBased == false {
			return fmt.Errorf("AdapatationSet (%v) SegmentTemplate.Media (%v) MUST be either %v or %v", adaptSet.Id, segTemplate.Media, TimeToken, NumberToken)
		}
		if timeBased == true && numberBased == true {
			return fmt.Errorf("AdapatationSet (%v) SegmentTemplate.Media (%v) MUST be either %v or %v", adaptSet.Id, segTemplate.Media, TimeToken, NumberToken)
		}
		if f.isTimeBased == nil {
			f.isTimeBased = new(bool)
			*f.isTimeBased = timeBased
		}
		if *f.isTimeBased != timeBased {
			return fmt.Errorf("Different AdaptationSets are using different URL patterns, Not Supported")
		}
	}
	if adaptSet.SegmentTemplate.Duration != 0 {
		durationPresent = true
	}
	if adaptSet.SegmentTemplate.SegmentTimeline.S != nil {
		if len(adaptSet.SegmentTemplate.SegmentTimeline.S) > 0 {
			segTimelinePresent = true
		}
	}
	if segTemplate.Timescale != 0 {
		timeScalePresent = true
	}
	if segTimelinePresent && durationPresent {
		return fmt.Errorf("For AdaptationSet %v only ONE of SegmentTemplate.Duration(\"%v\") or SegmentTemplate.SegmentTimeline(%v items) MUST be present. ", adaptSet.Id, adaptSet.SegmentTemplate.Duration, len(adaptSet.SegmentList.SegmentTimeline.S))
	}
	if segTimelinePresent && !timeScalePresent {
		return fmt.Errorf("AdapatationSet (%v) SegmentTemplate.TimeScale (%v) MUST be present with SegmentTemplate.SegmentTimeline", adaptSet.Id, segTemplate.Timescale)
	}
	if f.isSegmentTimeline == nil {
		f.isSegmentTimeline = new(bool)
		*f.isSegmentTimeline = segTimelinePresent
	}
	if segTimelinePresent != *f.isSegmentTimeline {
		return fmt.Errorf("Different AdaptationSets are using different GEN patterns, Not Supported")
	}
	for _, rep := range adaptSet.Representation {
		if rep.Id == "" {
			return fmt.Errorf("Representation without ID(\"%v:%v\") found", adaptSet.Id, rep.Id)
		}
		if rep.Bandwidth <= 0 {
			return fmt.Errorf("Representation(\"%v:%v\") with invalid Bandwidth(%v) found", adaptSet.Id, rep.Id, rep.Bandwidth)
		}
	}
	return nil
}

func (f *ReaderFactory) validateStatic(mpd *MPDtype) error {
	if !strings.Contains(mpd.Profiles, OnDemandProfile) {
		return fmt.Errorf("MPD.Profile MUST include \"%v\" for MPD.Type=\"static\"", OnDemandProfile)
	}
	return nil
}

//makeDASHReader - depending on the read type, return DASH Reader
func (f *ReaderFactory) makeDASHReader(ID string, mpd *MPDtype) (Reader, error) {
	if f.IsLive {
		if f.isSegmentTimeline != nil {
			if *f.isSegmentTimeline {
				if f.isTimeBased != nil {
					ret := &readerLiveMPDUpdate{
						readerBaseExtn: readerBaseExtn{
							updCounter: 0,
							readerBase: readerBase{
								ID:       ID,
								baseURL:  f.baseURL,
								baseTime: f.AST,
								isNumber: !*f.isTimeBased,
								isTime:   *f.isTimeBased,
							},
						},
					}
					_, _, err := ret.Update(mpd)
					if err != nil {
						return nil, err
					}
					return ret, nil
				}
			}

		}
	}
	return nil, fmt.Errorf("Reader not found")
}
