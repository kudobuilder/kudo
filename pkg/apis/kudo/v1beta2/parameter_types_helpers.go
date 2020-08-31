package v1beta2

func (p *Parameter) IsImmutable() bool {
	return p.Immutable != nil && *p.Immutable
}

func (p *Parameter) IsRequired() bool {
	return p.Required != nil && *p.Required
}

func (p *Parameter) HasDefault() bool {
	return p.Default != nil
}

// GetChangedParamDefs returns a list of parameters from ov2 that changed based on the given compare function between ov1 and ov2
func GetChangedParamDefs(p1, p2 []Parameter, isEqual func(p1, p2 Parameter) bool) []Parameter {
	changedParams := []Parameter{}

	for _, p1 := range p1 {
		for _, p2 := range p2 {
			if p1.Name == p2.Name {
				if !isEqual(p1, p2) {
					changedParams = append(changedParams, p2)
				}
			}
		}
	}

	return changedParams
}

// GetAddedParameters returns a list of parameters that are in oldOv but not in newOv
func GetRemovedParamDefs(old, new []Parameter) []Parameter {
	return GetAddedParameters(new, old)
}

// GetAddedParameters returns a list of parameters that are in newOv but not in oldOv
func GetAddedParameters(old, new []Parameter) []Parameter {
	addedParams := []Parameter{}

NewParams:
	for _, newParam := range new {
		for _, oldParam := range old {
			if newParam.Name == oldParam.Name {
				continue NewParams
			}
		}
		addedParams = append(addedParams, newParam)
	}
	return addedParams
}

// ParameterDiff returns map containing all parameters that were removed or changed between old and new
func ParameterDiff(old, new map[string]string) map[string]string {
	changed, removed := RichParameterDiff(old, new)

	// Join both maps
	for key, val := range removed {
		changed[key] = val
	}
	return changed
}

// RichParameterDiff compares new and old map and returns two maps: first containing all changed/added
// and second all removed parameters.
func RichParameterDiff(old, new map[string]string) (changed, removed map[string]string) {
	changed, removed = make(map[string]string), make(map[string]string)

	for key, val := range old {
		// If a parameter was removed in the new spec
		if _, ok := new[key]; !ok {
			removed[key] = val
		}
	}

	for key, val := range new {
		// If new spec parameter was added or changed
		if v, ok := old[key]; !ok || v != val {
			changed[key] = val
		}
	}
	return
}
