package lzh

var (
	threshold uint = 3
	discbit        = 13
	discsiz        = int16(1) << discbit
	ucharMax  uint = (0x7F*2 + 1)
)

func (l *Lzh) decodeStart() {
	l.hufDecodeStart()
	l.j = 0
}

/* The calling function must keep the number of
   bytes to be processed.  This function decodes
   either 'count' bytes or 'DICSIZ' bytes, whichever
   is smaller, into the array 'buffer[]' of size
   'DICSIZ' or more.
   Call decode_start() once for each new file
   before calling this function. */
func (l *Lzh) decode(count uint, buffer []byte) {
	var i, r, c uint
	l.j--
	for l.j >= 0 {
		buffer[r] = buffer[i]
		i = (i + 1) & uint(discsiz-1)
		r++
		if r == count {
			return
		}
		l.j--
	}

	for {
		c = l.decodeC()
		if c <= ucharMax {
			buffer[r] = byte(c)
			r++
			if r == count {
				return
			}
		} else {
			l.j = int(c - (ucharMax + 1 - threshold))
			i = (r - l.decodeP() - 1) & uint(discsiz-1)
			l.j--
			for l.j >= 0 {
				buffer[r] = buffer[i]
				i = (i + 1) & uint(discsiz-1)
				l.j--
			}
		}
	}

}
