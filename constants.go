package dashreader

const (
	//LiveProfile - String for Live Profile, Field: MPD@Profiles
	LiveProfile = "urn:mpeg:dash:profile:isoff-live:2011"
	//OnDemandProfile - String for OnDemandProfile Profile, Field: MPD@Profiles
	OnDemandProfile = "urn:mpeg:dash:profile:isoff-ondemand:2011"
	//RepresentationIDToken - Token part of SegmentTemplate@Media or SegmentTemplate@Index
	RepresentationIDToken = "$RepresentationID$"
	//TimeToken - Token part of SegmentTemplate@Media
	TimeToken = "$Time$"
	//NumberToken - Token part of SegmentTemplate@Media
	NumberToken = "$Number$"
)
