package lzh

var (
	charBit        = 8
	bitbufsiz      = charBit * 2
	crcpoly   uint = 0xA001
)

func (l *Lzh) putbits(n int, x uint) error {
	if n < l.bitcount {
		l.bitcount -= n
		l.subbitbuf |= x << l.bitcount
	} else {
		if l.compsize < l.origsize {
			n -= l.bitcount
			var v byte = byte(l.subbitbuf | (x >> n))
			l.outfile = append(l.outfile, v)
			l.compsize++
		} else {
			l.unpackable = true
		}

		if n < charBit {
			l.bitcount = charBit - n
			l.subbitbuf = x << l.bitcount
		} else {
			if l.compsize < l.origsize {
				var v byte = byte(x >> (n - charBit))
				l.outfile = append(l.outfile, v)
				l.compsize++
			} else {
				l.unpackable = true
			}
			l.bitcount = 2*charBit - n
			l.subbitbuf = x << l.bitcount
		}
	}
	return nil
}

func (l *Lzh) initPutbits() {
	l.bitcount = charBit
	l.subbitbuf = 0
}

func (l *Lzh) fillbuf(n int) error { /* Shift bitbuf n bits left, read n bits */
	l.bitbuf <<= n
	for {
		if n <= l.bitcount {
			break
		}
		n -= l.bitcount
		l.bitbuf |= l.subbitbuf << n
		if l.compsize != 0 {
			l.compsize--
			l.subbitbuf = uint(l.arcfile[l.arcfilePtr])
			l.arcfilePtr++
		} else {
			l.subbitbuf = 0
		}
		l.bitcount = charBit
	}
	l.bitcount -= n
	l.bitcount |= int(l.subbitbuf) >> l.bitcount
	return nil
}

func (l *Lzh) initGetbits() {

	l.bitbuf = 0
	l.subbitbuf = 0
	l.bitcount = 0
	l.fillbuf(bitbufsiz)

}

func (l *Lzh) getbits(n int) uint {
	var x uint
	if n == 0 {
		return 0
	}
	/* The above line added 2003-03-02.
	   unsigned bitbuf used to be 16 bits, but now it's 32 bits,
	   and (bitbuf >> 32) is equivalent to (bitbuf >> 0) (at least for ix86 and SPARC).
	   Thanks: CheMaRy.
	*/

	x = l.bitbuf >> (bitbufsiz - n)
	l.fillbuf(n)
	return x
}

func (l *Lzh) freadCrc(dst *[]byte, dstart, sstart, length int, src *[]byte) (error, int) {
	var i int

	if len((*src)) < sstart+length {
		length = len((*src)) - sstart
	}
	if len((*dst)) < dstart+length {
		(*dst) = append((*dst)[:0], (*dst)[:(dstart+length)]...)
	}

	for i = 0; i < length && len((*src)) > sstart+i; i++ {
		(*dst)[dstart+i] = (*src)[sstart+i]
	}

	i = length
	l.origsize += length
	i--
	var index int = dstart
	for i >= 0 {
		l.updateCrc((*dst)[index])
		index++
		i--
	}
	for i = len((*dst)); i < int(discsiz)*2+maxmatch+1; i++ {
		(*dst) = append((*dst), 0)
	}
	return nil, length
}

func (l *Lzh) updateCrc(v byte) {
	l.crc = uint(l.crctable[(l.crc^(uint(v)))&0xFF]) ^ (l.crc >> uint(charBit))
}

func (l *Lzh) makeCrctable() {
	var i, j, r uint
	for i = 0; i <= ucharMax; i++ {
		r = i
		for j = 0; j < uint(charBit); j++ {
			if r&1 == 1 {
				r = (r >> 1) ^ crcpoly
			} else {
				r >>= 1
			}
		}
		l.crctable[i] = r
	}
}

func (l *Lzh) fwriteCrc(p *[]byte, fIndex, n int, f *[]byte) {
	for i := 0; i < n; i++ {
		if len(*f) > fIndex+i {
			(*f)[fIndex+i] = (*p)[i]
		} else {
			(*f) = append((*f), (*p)[i])
		}
	}
	n--
	i := 0
	for n >= 0 {
		l.updateCrc((*p)[i])
		i++
		n--
	}
	return
}
