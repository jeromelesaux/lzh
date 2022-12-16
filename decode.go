package lzh

var (
	threshold uint = 3
	discbit        = 13
	discsiz        = int16(1) << discbit
	ucharMax  uint = (0x7F*2 + 1)
)

func (l *Lzh) decodeStart() error {
	err := l.hufDecodeStart()
	if err != nil {
		return err
	}
	l.j = 0
	return nil
}

/* The calling function must keep the number of
   bytes to be processed.  This function decodes
   either 'count' bytes or 'DICSIZ' bytes, whichever
   is smaller, into the array 'buffer[]' of size
   'DICSIZ' or more.
   Call decode_start() once for each new file
   before calling this function. */
var decodeIndex uint32

func (l *Lzh) decode(count uint16, buffer *[]byte) error {
	var r, c uint16
	l.j--
	for l.j >= 0 {
		(*buffer)[r] = (*buffer)[decodeIndex]
		decodeIndex = (decodeIndex + 1) & uint32(discsiz-1)
		r++
		if r == count {
			return nil
		}
		l.j--
	}

	for {
		var err error
		c, err = l.decodeC()
		if err != nil {
			return err
		}
		if c <= uint16(ucharMax) {
			(*buffer)[r] = byte(c)
			r++
			if r == count {
				return nil
			}
		} else {
			l.j = int(c - uint16(ucharMax+1-threshold))
			v, err := l.decodeP()
			if err != nil {
				return err
			}
			decodeIndex = uint32(r-v-1) & uint32(discsiz-1)
			l.j--
			for l.j >= 0 {
				(*buffer)[r] = (*buffer)[decodeIndex]
				decodeIndex = (decodeIndex + 1) & uint32(discsiz-1)
				r++
				if r == count {
					return nil
				}
				l.j--
			}
		}
	}

}
