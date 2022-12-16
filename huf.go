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

func (l *Lzh) writePtLen(n, nbit, iSpecial int) error {
	var i, k int
	for n > 0 && l.ptLen[n-1] == 0 {
		n--
	}
	err := l.putbits(nbit, uint16(n))
	if err != nil {
		return err
	}
	i = 0
	for i < n {

		k = int(l.ptLen[i])
		i++
		if k <= 6 {
			err := l.putbits(3, uint16(k))
			if err != nil {
				return err
			}
		} else {
			err := l.putbits(k-3, (1<<(k-3))-2)
			if err != nil {
				return err
			}
		}
		if i == iSpecial {
			for i < 6 && l.ptLen[i] == 0 {
				i++
			}
			err := l.putbits(2, uint16((i-3)&3))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Lzh) writeCLen() error {
	var i, k, n, count int
	n = nc
	for n > 0 && l.cLen[n-1] == 0 {
		n--
	}

	err := l.putbits(cbit, uint16(n))
	if err != nil {
		return err
	}
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
					err = l.putbits(int(l.ptLen[0]), l.ptCode[0])
					if err != nil {
						return err
					}
				}
			} else {
				if count <= 18 {
					err := l.putbits(int(l.ptLen[1]), l.ptCode[1])
					if err != nil {
						return err
					}
					err = l.putbits(4, uint16(count-3))
					if err != nil {
						return err
					}
				} else {
					if count == 19 {
						err := l.putbits(int(l.ptLen[0]), l.ptCode[0])
						if err != nil {
							return err
						}
						err = l.putbits(int(l.ptLen[1]), l.ptCode[1])
						if err != nil {
							return err
						}
						err = l.putbits(4, 15)
						if err != nil {
							return err
						}
					} else {
						err := l.putbits(int(l.ptLen[2]), l.ptCode[2])
						if err != nil {
							return err
						}
						err = l.putbits(cbit, uint16(count)-20)
						if err != nil {
							return err
						}
					}
				}
			}
		} else {
			err := l.putbits(int(l.ptLen[k+2]), l.ptCode[k+2])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Lzh) encodeC(c int) error {
	return l.putbits(int(l.cLen[c]), l.cCode[c])
}

func (l *Lzh) encodeP(p uint16) error {
	var c, q uint16
	q = p
	for q != 0 {
		q >>= 1
		c++
	}
	err := l.putbits(int(l.ptLen[c]), uint16(l.ptCode[c]))
	if err != nil {
		return err
	}
	if c > 1 {
		err := l.putbits(int(c-1), uint16(p&(0xFFFF>>(17-c))))
		if err != nil {
			return err
		}
	}
	return nil
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

func (l *Lzh) hufEncodeEnd() error {
	if !l.unpackable {
		err := l.sendBlock()
		if err != nil {
			return err
		}
		err = l.putbits(charBit-1, 0) /* flush remaining bits */
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Lzh) hufDecodeStart() error {
	err := l.initGetbits()
	if err != nil {
		return err
	}
	l.blocksize = 0
	return nil
}

func (l *Lzh) sendBlock() error {
	var flags, root, pos, size uint16
	var k uint16
	var i int
	root = uint16(l.makeTree(nc, &l.cFreq, &l.cLen, &l.cCode))
	size = uint16(l.cFreq[root])
	err := l.putbits(16, size)
	if err != nil {
		return err
	}
	if int(root) >= nc {
		l.countTFreq()
		root = uint16(l.makeTree(nt, &l.tFreq, &l.ptLen, &l.ptCode))
		if int(root) >= nt {
			err := l.writePtLen(nt, tbit, 3)
			if err != nil {
				return err
			}
		} else {
			err := l.putbits(tbit, 0)
			if err != nil {
				return err
			}
			err = l.putbits(tbit, root)
			if err != nil {
				return err
			}
		}
		err := l.writeCLen()
		if err != nil {
			return err
		}
	} else {
		err := l.putbits(tbit, 0)
		if err != nil {
			return err
		}
		err = l.putbits(tbit, 0)
		if err != nil {
			return err
		}
		err = l.putbits(cbit, 0)
		if err != nil {
			return err
		}
		err = l.putbits(cbit, root)
		if err != nil {
			return err
		}
	}
	root = uint16(l.makeTree(np, &l.pFreq, &l.ptLen, &l.ptCode))
	if int(root) >= np {
		err := l.writePtLen(np, pbit, -1)
		if err != nil {
			return err
		}
	} else {
		err := l.putbits(pbit, 0)
		if err != nil {
			return err
		}
		err = l.putbits(pbit, root)
		if err != nil {
			return err
		}
	}
	pos = 0
	for i = 0; i < int(size); i++ {
		if i%charBit == 0 {
			flags = uint16(l.buf[pos])
			pos++
		} else {
			flags <<= 1
		}
		if flags&(uint16(1)<<(charBit-1)) != 0 {
			err := l.encodeC(int(uint(l.buf[pos]) + (uint(1) << charBit)))
			if err != nil {
				return err
			}
			pos++
			k = uint16(l.buf[pos]) << charBit
			pos++
			k += uint16(l.buf[pos])
			pos++
			err = l.encodeP(k)
			if err != nil {
				return err
			}
		} else {

			err := l.encodeC(int(l.buf[pos]))
			if err != nil {
				return err
			}
			pos++
		}
		if l.unpackable {
			return nil
		}
	}
	for i = 0; i < nc; i++ {
		l.cFreq[i] = 0
	}
	for i = 0; i < np; i++ {
		l.pFreq[i] = 0
	}
	return nil
}

/***** decoding *****/
func (l *Lzh) readPtLen(nn, nbit, iSpecial int) error {
	var i, c, n int
	var mask uint32
	v, err := l.getbits(nbit)
	if err != nil {
		return err
	}
	n = int(v)
	if n == 0 {
		v, err := l.getbits(nbit)
		if err != nil {
			return err
		}
		c = int(v)
		for i = 0; i < nn; i++ {
			l.ptLen[i] = 0
		}
		for i = 0; i < 256; i++ {
			l.ptTable[i] = uint16(c)
		}
	} else {
		i = 0
		for i < n {
			c = int(l.bitbuf >> (bitbufsiz - 3))
			if c == 7 {
				mask = uint32(1) << (bitbufsiz - 1 - 3)
				for (mask & l.bitbuf) != 0 {
					mask >>= 1
					c++
				}
			}
			if c < 7 {
				err := l.fillbuf(3)
				if err != nil {
					return err
				}
			} else {
				err := l.fillbuf(c - 3)
				if err != nil {
					return err
				}
			}

			l.ptLen[i] = byte(c)
			i++
			if i == iSpecial {
				v, err := l.getbits(2)
				if err != nil {
					return err
				}
				c = int(v)
				c--
				for c >= 0 {
					c--
					l.ptLen[i] = 0
					i++
				}
			}
		}
		for i < nn {
			l.ptLen[i] = 0
			i++
		}
		err := l.makeTable(nn, &l.ptLen, 8, &l.ptTable)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Lzh) readCLen() error {
	var i, c, n int
	var mask uint32

	v, err := l.getbits(cbit)
	if err != nil {
		return err
	}
	n = int(v)
	if n == 0 {
		v, err := l.getbits(cbit)
		if err != nil {
			return err
		}
		c = int(v)
		for i = 0; i < nc; i++ {
			l.cLen[i] = 0
		}
		for i = 0; i < 4096; i++ {
			l.cTable[i] = uint16(c)
		}
	} else {
		i = 0
		for i < n {
			c = int(l.ptTable[l.bitbuf>>(bitbufsiz-8)])
			if c >= nt {
				mask = uint32(1) << (bitbufsiz - 1 - 8)
				for {
					if (l.bitbuf & mask) != 0 {
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
			err := l.fillbuf(int(l.ptLen[c]))
			if err != nil {
				return err
			}
			if c <= 2 {
				if c == 0 {
					c = 1
				} else {
					if c == 1 {
						v, err := l.getbits(4)
						if err != nil {
							return err
						}
						c = int(v) + 3
					} else {
						v, err = l.getbits(cbit)
						if err != nil {
							return err
						}
						c = int(v) + 20
					}
				}
				c--
				for c >= 0 {

					l.cLen[i] = 0
					i++
					c--
				}
			} else {

				l.cLen[i] = byte(c - 2)
				i++
			}
		}
		for i < nc {

			l.cLen[i] = 0
			i++
		}
		err := l.makeTable(nc, &l.cLen, 12, &l.cTable)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Lzh) decodeC() (uint16, error) {
	var j, mask uint16
	if l.blocksize == 0 {
		var err error
		l.blocksize, err = l.getbits(16)
		if err != nil {
			return 0, err
		}
		err = l.readPtLen(nt, tbit, 3)
		if err != nil {
			return 0, err
		}
		err = l.readCLen()
		if err != nil {
			return 0, err
		}
		err = l.readPtLen(np, pbit, -1)
		if err != nil {
			return 0, err
		}
	}
	l.blocksize--
	j = uint16(l.cTable[l.bitbuf>>(bitbufsiz-12)])
	if int(j) >= nc {
		mask = uint16(1) << (bitbufsiz - 1 - 12)
		for {
			if (uint16(l.bitbuf) & mask) == 0 {
				j = uint16(l.right[j])
			} else {
				j = uint16(l.left[j])
			}
			mask >>= 1
			if int(j) < nc {
				break
			}
		}
	}
	return j, l.fillbuf(int(l.cLen[j]))
}

func (l *Lzh) decodeP() (uint16, error) {
	var j uint16
	var mask uint32
	j = uint16(l.ptTable[l.bitbuf>>(bitbufsiz-8)])
	if int(j) >= np {
		mask = uint32(1) << (bitbufsiz - 1 - 8)
		for int(j) >= np {
			if (l.bitbuf & mask) != 0 {
				j = uint16(l.right[j])
			} else {
				j = uint16(l.left[j])
			}
			mask >>= 1
		}
	}
	err := l.fillbuf(int(l.ptLen[j]))
	if err != nil {
		return j, err
	}
	if j != 0 {
		v, err := l.getbits(int(j - 1))
		if err != nil {
			return 0, err
		}
		j = (uint16(1) << (j - 1)) + v
	}
	return j, nil
}

var cpos uint

func (l *Lzh) output(c, p uint) error {

	l.outputMask >>= 1
	if l.outputMask == 0 {
		l.outputMask = uint(1) << (charBit - 1)
		if l.outputPos >= (l.bufsiz - 3*uint(charBit)) {
			err := l.sendBlock()
			if err != nil {
				return err
			}
			if l.unpackable {
				return nil
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
	return nil
}

func (l *Lzh) printCLen() {
	for i := 0; i < nc; i++ {
		fmt.Printf("l.c_len[%d]:%d\n", i, l.cLen[i])
	}
}
