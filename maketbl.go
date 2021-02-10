package lzh

import (
	"errors"
)

func (l *Lzh) makeTable(nchar int, bitlen *[]byte, tablebits int, table *[]uint16) error {
	var (
		count  [17]uint16
		weight [17]uint16
		start  [18]uint16
		p      int
	)
	var i, k, len, ch, jutbits, avail, nextcode, mask uint
	for i = 1; i <= 16; i++ {
		start[i+1] = start[i] + (count[i] << (16 - i))
	}
	for i = 0; i < uint(nchar); i++ {
		count[(*bitlen)[i]]++
	}
	start[1] = 0
	for i = 1; i <= 16; i++ {
		start[i+1] = start[i] + (count[i] << (16 - i))
	}
	if start[17] != 0 {
		return errors.New("Bad table")
	}
	jutbits = 16 - uint(tablebits)
	for i = 1; i <= uint(tablebits); i++ {
		start[i] >>= jutbits
		weight[i] = (uint16(1) << (uint(tablebits) - i))
	}
	for i <= 16 {
		weight[i] = (uint16(1) << (16 - i))
		i++
	}
	/* Note: in the 1990 version of ar002, the above three lines
	       were:
	           while (i <= 16) weight[i++] = 1U << (16 - i);
		   but that was awfully compiler-dependent. */
	i = uint(start[tablebits+1] >> jutbits)
	if i != 0 {
		k = uint(1) << tablebits
		for i != k {
			(*table)[i] = 0
			i++
		}
	}
	avail = uint(nchar)
	mask = uint(1) << (15 - tablebits)
	for ch = 0; ch < uint(nchar); ch++ {
		len = uint((*bitlen)[ch])
		if len == 0 {
			continue
		}
		nextcode = uint(start[len]) + uint(weight[len])
		if int(len) <= tablebits {
			for i = uint(start[len]); i < nextcode; i++ {
				(*table)[i] = uint16(ch)
			}
		} else {
			k = uint(start[len])
			p = int(k >> jutbits)
			i = len - uint(tablebits)
			var setRight bool
			for i != 0 {
				if (*table)[p] == 0 {
					l.right[avail] = 0
					l.left[avail] = 0
					(*table)[p] = uint16(avail)
					avail++
				}
				p = int((*table)[p])
				if (k & mask) != 0 {
					setRight = true

				} else {
					setRight = false
				}
				k <<= 1
				i--
			}
			if setRight {
				l.right[p] = int16(ch)
			} else {
				l.left[p] = int16(ch)
			}
		}
		start[len] = uint16(nextcode)
	}
	return nil
}
