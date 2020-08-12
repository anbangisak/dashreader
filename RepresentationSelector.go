package dashreader

//RepresentationSelector -
//Decides among the list of representations which one to use
type RepresentationSelector interface {
	SelectRepresentation([]*RepresentationType) *RepresentationType
}

//MinBWRepresentationSelector -
//Selects the minimum of the available BW
type MinBWRepresentationSelector struct {
}

//SelectRepresentation - Selects one of the available representationss
func (s MinBWRepresentationSelector) SelectRepresentation(reps []*RepresentationType) *RepresentationType {
	var ret *RepresentationType
	var bandwidth uint = 0xFFFFFFFF
	for _, r := range reps {
		if r.Bandwidth < bandwidth {
			bandwidth = r.Bandwidth
			ret = r
		}
	}
	return ret
}

//MaxBWRepresentationSelector -
//Selects the minimum of the available BW
type MaxBWRepresentationSelector struct {
}

//SelectRepresentation - Selects one of the available representationss
func (s MaxBWRepresentationSelector) SelectRepresentation(reps []*RepresentationType) *RepresentationType {
	var ret *RepresentationType
	var bandwidth uint = 0
	for _, r := range reps {
		if r.Bandwidth < bandwidth {
			bandwidth = r.Bandwidth
			ret = r
		}
	}
	return ret
}
