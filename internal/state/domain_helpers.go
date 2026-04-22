package state

func copyBoolMap(src map[string]bool) map[string]bool {
	if src == nil {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyRecordMap(src map[string]*RecordInfo) map[string]*RecordInfo {
	if src == nil {
		return make(map[string]*RecordInfo)
	}
	dst := make(map[string]*RecordInfo, len(src))
	for k, v := range src {
		if v == nil {
			dst[k] = nil
			continue
		}
		vCopy := *v
		dst[k] = &vCopy
	}
	return dst
}
