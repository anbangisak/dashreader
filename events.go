package dashreader

//EVENT NAMES
const (
	EvtMPDTimelineGapFilled           = "MPD_ERR_TIMELINE_GAP"               //Filled duration gap
	EvtMPDTimelineInFuture            = "MPD_ERR_TIMELINE_FUTURE"            //TimeLine in Future - WC, StartTime
	EvtMPDTimelineNoLivePointEntries  = "MPD_ERR_NO_LIVEPOINT_ENTRIES"       //No Entries found
	EvtMPDPublishTimeOld              = "MPD_PUBLISH_TIME_OLD"               //Publish time is older than previous
	EvtMPDNoActivePeriod              = "MPD_NO_ACTIVE_PERIOD"               //Period active not found
	EvtMPDNoAdaptAfterFilter          = "MPD_NO_ADAPT_AFTER_FILTER"          //No AdaptationSets after filter
	EvtMPDNoRepresentationAfterFilter = "MPD_NO_REPRESENTATION_AFTER_FILTER" //No Representations after filter

)
