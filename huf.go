package lzh

import "fmt"

var (
	codeBit  = 16
	np       = (discbit + 1)
	nt       = (codeBit + 3)
	pbit     = 4
	tbit     = 5
	npt      = nt
	cbit     = 9
	maxmatch = 256
	bufsiz   uint
	nc       = (int(ucharMax) + maxmatch + 2 - int(threshold))
)

func (l *Lzh) countTFreq() {
	var i, k, n, count int
	for i = 0; i < nt; i++ {
		l.tFreq[i] = 0
	}
	n = nc
	for n > 0 && l.cLen[n-1] == 0 {
		n--
	}
	i = 0
	for i < n {
		k = int(l.cLen[i])
		i++
		if k == 0 {
			count = 1
			for i < n && l.cLen[i] == 0 {
				i++
				count++
			}
			if count <= 2 {
				l.tFreq[0] += uint16(count)
			} else {
				if count <= 18 {
					l.tFreq[1]++
				} else {
					if count == 19 {
						l.tFreq[0]++
						l.tFreq[1]++
					} else {
						l.tFreq[2]++
					}
				}
			}
		} else {
			l.tFreq[k+2]++
		}
	}
}

func (l *Lzh) writePtLen(n, nbit, iSpecial int) {
	var i, k int
	for n > 0 && l.ptLen[n-1] == 0 {
		n--
	}
	l.putbits(nbit, uint(n))
	i = 0
	for i < n {

		k = int(l.ptLen[i])
		i++
		if k <= 6 {
			l.putbits(3, uint(k))
		} else {
			l.putbits(k-3, (1<<(k-3))-2)
		}
		if i == iSpecial {
			for i < 6 && l.ptLen[i] == 0 {
				i++
			}
			l.putbits(2, uint((i-3)&3))
		}
	}
}

func (l *Lzh) writeCLen() {
	var i, k, n, count int
	n = nc
	for n > 0 && l.cLen[n-1] == 0 {
		n--
	}

	l.putbits(cbit, uint(n))
	for i < n {
		k = int(l.cLen[i])
		i++
		if k == 0 {
			count = 1
			for i < n && l.cLen[i] == 0 {
				i++
				count++
			}
			if count <= 2 {
				for k = 0; k < count; k++ {
					l.putbits(int(l.ptLen[0]), uint(l.ptCode[0]))
				}
			} else {
				if count <= 18 {
					l.putbits(int(l.ptLen[1]), uint(l.ptCode[1]))
					l.putbits(4, uint(count-3))
				} else {
					if count == 19 {
						l.putbits(int(l.ptLen[0]), uint(l.ptCode[0]))
						l.putbits(int(l.ptLen[1]), uint(l.ptCode[1]))
						l.putbits(4, 15)
					} else {
						l.putbits(int(l.ptLen[2]), uint(l.ptCode[2]))
						l.putbits(cbit, uint(count-20))
					}
				}
			}
		} else {
			l.putbits(int(l.ptLen[k+2]), uint(l.ptCode[k+2]))
		}
	}
}

func (l *Lzh) encodeC(c int) {
	l.putbits(int(l.cLen[c]), uint(l.cCode[c]))
}

func (l *Lzh) encodeP(p uint16) {
	var c, q uint16
	q = p
	for q != 0 {
		q >>= 1
		c++
	}
	l.putbits(int(l.ptLen[c]), uint(l.ptCode[c]))
	if c > 1 {
		l.putbits(int(c-1), uint(p&(0xFFFF>>(17-c))))
	}
}

func (l *Lzh) hufEncodeStart() {
	var i int
	bufsiz = 16 * 1024
	l.buf = make([]byte, bufsiz)
	l.buf[0] = 0
	for i = 0; i < nc; i++ {
		l.cFreq[i] = 0
	}
	for i = 0; i < np; i++ {
		l.pFreq[i] = 0
	}
	l.outputPos = 0
	l.outputMask = 0
	l.initPutbits()
}

func (l *Lzh) hufEncodeEnd() {
	if !l.unpackable {
		l.sendBlock()
		l.putbits(charBit-1, 0) /* flush remaining bits */
	}
}

func (l *Lzh) hufDecodeStart() {
	l.initGetbits()
	l.blocksize = 0
}

func (l *Lzh) sendBlock() {
	var flags, root, pos, size uint
	var k uint16
	var i int
	root = uint(l.makeTree(nc, &l.cFreq, &l.cLen, &l.cCode))
	size = uint(l.cFreq[root])
	l.putbits(16, size)
	if int(root) >= nc {
		l.countTFreq()
		root = uint(l.makeTree(nt, &l.tFreq, &l.ptLen, &l.ptCode))
		if int(root) >= nt {
			l.writePtLen(nt, tbit, 3)
		} else {
			l.putbits(tbit, 0)
			l.putbits(tbit, root)
		}
		l.writeCLen()
	} else {
		l.putbits(tbit, 0)
		l.putbits(tbit, 0)
		l.putbits(cbit, 0)
		l.putbits(cbit, root)
	}
	root = uint(l.makeTree(np, &l.pFreq, &l.ptLen, &l.ptCode))
	if int(root) >= np {
		l.writePtLen(np, pbit, -1)
	} else {
		l.putbits(pbit, 0)
		l.putbits(pbit, root)
	}
	pos = 0
	for i = 0; i < int(size); i++ {
		if i%charBit == 0 {
			flags = uint(l.buf[pos])
			pos++
		} else {
			flags <<= 1
		}
		if flags&(uint(1)<<(charBit-1)) != 0 {
			l.encodeC(int(uint(l.buf[pos]) + (uint(1) << charBit)))
			pos++
			k = uint16(l.buf[pos]) << charBit
			pos++
			k += uint16(l.buf[pos])
			pos++
			l.encodeP(k)
		} else {

			l.encodeC(int(l.buf[pos]))
			pos++
		}
		if l.unpackable {
			return
		}
	}
	for i = 0; i < nc; i++ {
		l.cFreq[i] = 0
	}
	for i = 0; i < np; i++ {
		l.pFreq[i] = 0
	}
}

/***** decoding *****/
func (l *Lzh) readPtLen(nn, nbit, iSpecial int) {
	var i, c, n int
	var mask uint
	n = int(l.getbits(nbit))
	if n == 0 {
		c = int(l.getbits(nbit))
		for i = 0; i < nn; i++ {
			l.ptLen[i] = 0
		}
		for i = 0; i < 256; i++ {
			l.ptTable[i] = byte(c)
		}
	} else {
		i = 0
		for i < n {
			c = int(l.bitbuf >> (bitbufsiz - 3))
			if c == 7 {
				mask = uint(1) << (bitbufsiz - 1 - 3)
				for mask&l.bitbuf == 0 {
					mask >>= 1
					c++
				}
			}
			if c < 7 {
				l.fillbuf(3)
			} else {
				l.fillbuf(c - 3)
			}
			i++
			l.ptLen[i] = byte(c)
			if i == iSpecial {
				c = int(l.getbits(2))
				c--
				for c >= 0 {
					c--
					i++
					l.ptLen[i] = 0
				}
			}
		}
		for i < nn {
			i++
			l.ptLen[i] = 0
		}
		l.makeTable(nn, l.ptLen, 8, l.ptTable)
	}
}

func (l *Lzh) readCLen() {
	var i, c, n int
	var mask uint

	n = int(l.getbits(cbit))
	if n == 0 {
		c = int(l.getbits(cbit))
		for i = 0; i < nc; i++ {
			l.cLen[i] = 0
		}
		for i = 0; i < 4096; i++ {
			l.cTable[i] = byte(c)
		}
	} else {
		i = 0
		for i < n {
			c = int(l.ptTable[l.bitbuf>>(bitbufsiz-8)])
			if c >= nt {
				mask = uint(1) << (bitbufsiz - 1 - 8)
				for {
					if (l.bitbuf & mask) == 0 {
						c = int(l.right[c])
					} else {
						c = int(l.left[c])
					}
					mask >>= 1
					if c < nt {
						break
					}
				}
			}
			l.fillbuf(int(l.ptLen[c]))
			if c <= 2 {
				if c == 0 {
					c = 1
				} else {
					if c == 1 {
						c = int(l.getbits(4)) + 3
					} else {
						c = int(l.getbits(cbit)) + 20
					}
				}
				c--
				for c >= 0 {
					i++
					l.cLen[i] = 0
				}
			} else {
				i++
				l.cLen[i] = byte(c - 2)
			}
		}
		for i < nc {
			i++
			l.cLen[i] = 0
		}
		l.makeTable(nc, l.cLen, 12, l.cTable)
	}
}

func (l *Lzh) decodeC() uint {
	var j, mask uint
	if l.blocksize == 0 {
		l.blocksize = l.getbits(16)
		l.readPtLen(nt, tbit, 3)
		l.readCLen()
		l.readPtLen(np, pbit, -1)
	}
	l.blocksize--
	j = uint(l.cTable[l.bitbuf>>(bitbufsiz-12)])
	if int(j) >= nc {
		mask = uint(1) << (bitbufsiz - 1 - 12)
		for {
			if (l.bitbuf & mask) == 0 {
				j = uint(l.right[j])
			} else {
				j = uint(l.left[j])
			}
			mask >>= 1
			if int(j) < nc {
				break
			}
		}
	}
	l.fillbuf(int(l.cLen[j]))
	return j
}

func (l *Lzh) decodeP() uint {
	var j, mask uint
	j = uint(l.ptTable[l.bitbuf>>(bitbufsiz-8)])
	if int(j) >= np {
		mask = uint(1) << (bitbufsiz - 1 - 8)
		for {
			if (l.bitbuf & mask) == 0 {
				j = uint(l.right[j])
			} else {
				j = uint(l.left[j])
			}
			mask >>= 1
			if int(j) < np {
				break
			}
		}
	}
	l.fillbuf(int(l.ptLen[j]))
	if j != 0 {
		j = (uint(1) << (j - 1)) + l.getbits(int(j-1))
	}
	return j
}

var cpos uint

func (l *Lzh) output(c, p uint) {

	l.outputMask >>= 1
	if l.outputMask == 0 {
		l.outputMask = uint(1) << (charBit - 1)
		if l.outputPos >= (l.bufsiz - 3*uint(charBit)) {
			l.sendBlock()
			if l.unpackable {
				return
			}
			l.outputPos = 0
		}
		cpos = l.outputPos
		l.outputPos++
		l.buf[cpos] = 0
	}
	l.buf[l.outputPos] = byte(c)
	l.outputPos++
	l.cFreq[c]++
	if c >= (uint(1) << charBit) {
		l.buf[cpos] |= byte(l.outputMask)
		l.buf[l.outputPos] = byte(p >> charBit)
		l.outputPos++
		l.buf[l.outputPos] = byte(p)
		l.outputPos++
		c = 0
		for p != 0 { // @TODO test
			p >>= 1
			c++
		}
		l.pFreq[c]++
	}
}

func (l *Lzh) printCLen() {
	for i := 0; i < nc; i++ {
		fmt.Printf("l.c_len[%d]:%d\n", i, l.cLen[i])
	}
}
