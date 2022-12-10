package utilities

func (s *fsm) containsSymbol(symbols []byte, symbol byte) bool {
	for _, x := range symbols {
		if x == symbol {
			return true
		}
	}
	return false
}

func substr(input string, start int, length int) string {
	asRunes := []rune(input)

	if start >= len(asRunes) {
		return ""
	}

	if start+length > len(asRunes) {
		length = len(asRunes) - start
	}

	return string(asRunes[start : start+length])
}

func makeBytesReadable(in []byte) string {
	ret := ""
	for i := 0; i < len(in); i++ {
		if in[i] < 32 {
			ret = ret + ASCIIMapNotPrintable[in[i]]
		} else {
			ret = ret + string(in[i])
		}
	}
	return ret
}
