package dashreader

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/PaesslerAG/gval"
)

//StreamSelector - Selection criteria for checking if AdaptationSet is good
//ContentType - "video","audio", etc...
//BitRates supported (bps) - 3000000 (for 3mpbs)
//        if empty anything is accepted
//        e.g. >3000000, <=3000000, ==3000000
//Codecs supported - regex
//        if empty anything is accepted
//        e.g. hvc1*, avc1.*
//Langs supported - regex of ISO 639-2 lang code
//        if empty anything is accepted
//        e.g. eng, spa
type StreamSelector struct {
	ID          string   `json:"id,omitempty"`
	ContentType string   `json:"contentType"`
	BitRates    []string `json:"bitratesexprs,omitempty"`
	Codecs      []string `json:"codecsregexs,omitempty"`
	Langs       []string `json:"langsregexs,omitempty"`
}

const (
	//MatchResultDontCare - don't care
	MatchResultDontCare = -1
	//MatchResultNotFound - Not found
	MatchResultNotFound = 0
	//MatchResultPartial - found partial
	MatchResultPartial = 1
	//MatchResultFound - found
	MatchResultFound = 2
)

//SelectAdaptationSets - select adaptationSets for the period
// period : The selected period
// streams : selection criteria
//Return:
// []adaptationSet : selected adaptationSets
// []error : specific error for each stream
func (sl *StreamSelectorList) SelectAdaptationSets(period PeriodType) ([]*AdaptationSetType, []error) {
	ret := make([]*AdaptationSetType, len(period.AdaptationSet))
	retErr := make([]error, len(period.AdaptationSet))
	for i, adaptSet := range period.AdaptationSet {
		//log.Printf("Evaluationg AdaptationSet %v %v ...", adaptSet.Id, adaptSet.ContentType)
		if len(*sl) <= 0 {
			//log.Printf("AdaptationSet %v %v included", adaptSet.Id, adaptSet.ContentType)
			ret[i] = &adaptSet
			continue
		}
		if len(adaptSet.ContentType) <= 0 {
			//ignore adaptatonset
			continue
		}
		for _, stream := range *sl {
			if adaptSet.ContentType == stream.ContentType {
				//ContentType matched
				match := stream.IsMatch(adaptSet)
				switch match {
				case MatchResultNotFound:
					continue //Check another AdaptationSet
				}
				//log.Printf("AdaptationSet %v %v included", adaptSet.ID, *adaptSet.ContentType)
				ret[i] = &adaptSet
				break
			}
		}
	}
	for i := range ret {
		retErr[i] = fmt.Errorf("Match not found")
	}
	return ret, retErr
}

//IsMatch - Finds if codecs present and required matches
// adaptSet : AdaptationSet
// return -
//   -1 - Don't Care
//    0 - Not Found
//    1 - Partial match
//    2 - Full match
func (s *StreamSelector) IsMatch(adaptSet AdaptationSetType) int {
	var ret int
	if adaptSet.ContentType != s.ContentType {
		return MatchResultNotFound
	}
	ret = s.matchCodec(adaptSet)
	switch ret {
	case MatchResultNotFound:
		return ret
	}
	ret1 := s.matchLang(adaptSet)
	if ret1 > ret {
		ret = ret1
	}
	return ret
}

//matchLang - finds if lang present and required matches
// adaptSet : AdaptationSet
// codecExpected : Decoder codec supported
// return -
//   -1 - Don't Care
//    0 - Not Found
//    1 - Partial match
//    2 - Full match
func (s *StreamSelector) matchLang(adaptSet AdaptationSetType) int {
	if len(s.Langs) == 0 {
		return MatchResultDontCare
	}
	if len(adaptSet.Lang) <= 0 {
		return MatchResultDontCare
	}
	for _, lang := range s.Langs {
		re := regexp.MustCompile(lang)
		if re == nil {
			continue //Regular expression is not good
		}
		if len(re.FindString(adaptSet.Lang)) > 0 {
			return MatchResultFound
		}
	}
	return 0
}

//matchCodec - finds if lang present and required matches
// adaptSet : AdaptationSet
// codecExpected : Decoder codec supported
// return -
//   -1 - Don't Care
//    0 - Not Found
//    1 - Partial match
//    2 - Full match
func (s *StreamSelector) matchCodec(adaptSet AdaptationSetType) int {
	//If no filters all is good
	if len(s.Codecs) == 0 {
		return MatchResultDontCare //Dont' Care
	}
	if len(adaptSet.Codecs) > 0 {
		//Input has Codecs at Adaptation Level
		for _, codec := range s.Codecs {
			re := regexp.MustCompile(codec)
			if re == nil {
				continue //Regular expression is not good
			}
			//If regex matches return success
			if len(re.FindString(adaptSet.Codecs)) > 0 {
				return MatchResultFound //Full match
			}
		}
	}
	var ret int
	ret = MatchResultDontCare //Don't Care
	//check for codecs in each of the Adaptation set
	for _, representation := range adaptSet.Representation {
		ret1 := s.matchCodecRep(representation)
		if ret1 > ret {
			ret = ret1
		}
	}
	return ret
}

//IsMatchRepresentation - Finds if codecs present and required matches
// adaptSet : AdaptationSet
// return -
//   -1 - Don't Care
//    0 - Not Found
//    1 - Partial match
//    2 - Full match
func (s *StreamSelector) IsMatchRepresentation(representation RepresentationType) int {
	var ret int
	ret = s.matchBitrateRep(representation)
	switch ret {
	case MatchResultNotFound:
		return ret
	}
	ret1 := s.matchCodecRep(representation)
	if ret1 > ret {
		ret = ret1
	}
	return ret
}

//matchBitrateRep - finds if bitrate present and required matches
// adaptSet : AdaptationSet
// codecExpected : Decoder codec supported
// return -
//   -1 - Don't Care
//    0 - Not Found
//    1 - Partial match
//    2 - Full match
func (s *StreamSelector) matchBitrateRep(representation RepresentationType) int {
	if len(s.BitRates) == 0 {
		return MatchResultDontCare
	}
	const bitrateStr = "br"
	rateValue := map[string]interface{}{
		bitrateStr: representation.Bandwidth,
	}
	//When more than one bw is given AND() is used
	for _, bitrateexpr := range s.BitRates {
		expr := bitrateStr + bitrateexpr
		value, err := gval.Evaluate(expr, rateValue)
		if err != nil {
			//the experession was wrong
			return MatchResultNotFound
		}
		switch v := reflect.ValueOf(value); v.Kind() {
		case reflect.Bool:
			if !v.Bool() {
				return MatchResultNotFound
			}
			//Found match
		default:
			//the experession was wrong
			return MatchResultNotFound
		}
	}
	return MatchResultFound
}

//matchCodecRep - finds if codec present and required matches
// adaptSet : AdaptationSet
// codecExpected : Decoder codec supported
// return -
//   -1 - Don't Care
//    0 - Not Found
//    1 - Partial match
//    2 - Full match
func (s *StreamSelector) matchCodecRep(representation RepresentationType) int {
	//If no filters all is good
	if len(s.Codecs) == 0 {
		return MatchResultDontCare //Dont' Care
	}
	for _, codec := range s.Codecs {
		re := regexp.MustCompile(codec)
		if re == nil {
			continue //Regular expression is not good
		}
		//If regex matches return success
		if len(re.FindString(representation.Codecs)) > 0 {
			return MatchResultFound //Full match
		}
	}
	//Regex present but doesn't match
	return MatchResultNotFound //Not Found
}
