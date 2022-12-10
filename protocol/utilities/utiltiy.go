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

var ASCIIMap = map[byte]string{
	0:  "<NUL>",
	1:  "<SOH>",
	2:  "<STX>",
	3:  "<ETX>",
	4:  "<EOT>",
	5:  "<ENQ>",
	6:  "<ACK>",
	7:  "<BEL>",
	8:  "<BS>",
	9:  "<HT>",
	10: "<LF>",
	11: "<VT>",
	12: "<FF>",
	13: "<CR>",
	14: "<SO>",
	15: "<SI>",
	16: "<DLE>",
	17: "<DC1>",
	18: "<DC2>",
	19: "<DC3>",
	20: "<DC4>",
	21: "<NAK>",
	22: "<SYN>",
	23: "<ETB>",
	24: "<CAN>",
	25: "<EM>",
	26: "<SUB>",
	27: "<ESC>",
	28: "<FS>",
	29: "<GS>",
	30: "<RS>",
	31: "<US>",
}

func makeBytesReadable(in []byte) string {
	ret := ""
	for i := 0; i < len(in); i++ {
		if in[i] < 32 {
			ret = ret + ASCIIMap[in[i]]
		} else {
			ret = ret + string(in[i])
		}
	}
	return ret
}
