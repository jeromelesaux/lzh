package lzh

import (
	"fmt"
	"os"
)

type node uint

var (
	maxHashVal = (3*discsiz + (discsiz/512+1)*int16(ucharMax))
	//perc_flag    int16 = 0x8000
)

func (l *Lzh) encode() error {
	var lastmatchlen int
	var lastmatchpos int16
	var err error
	l.allocateMemory()
	l.initSlide()
	l.hufEncodeStart()
	l.remainder, err = l.freadCrc(&l.text, int(discsiz), l.infilePtr, int(discsiz)+maxmatch, &l.infile)
	l.infilePtr += l.remainder
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, ".")
	l.matchlen = 0
	l.pos = discsiz
	l.insertNode()
	if l.matchlen > l.remainder {
		l.matchlen = l.remainder
	}
	for l.remainder > 0 && !l.unpackable {
		lastmatchlen = l.matchlen
		lastmatchpos = l.matchpos
		l.getNextMatch()
		if l.matchlen > l.remainder {
			l.matchlen = l.remainder
		}
		if l.matchlen > lastmatchlen || lastmatchlen < int(threshold) {
			err := l.output(uint(l.text[l.pos-1]), 0)
			if err != nil {
				return err
			}
		} else {
			err := l.output(uint(lastmatchlen)+(ucharMax+1-threshold), uint(l.pos-lastmatchpos-2)&uint(discsiz-1))
			if err != nil {
				return err
			}
			lastmatchlen--
			for lastmatchlen > 0 {
				l.getNextMatch()
				lastmatchlen--
			}
			if l.matchlen > l.remainder {
				l.matchlen = l.remainder
			}
		}
	}
	return l.hufEncodeEnd()
}

func (l *Lzh) allocateMemory() {
	l.text = make([]byte, int(discsiz)*2+maxmatch+1)
	l.level = make([]uint, uint(discsiz)+ucharMax+1)
	l.childcount = make([]uint, uint(discsiz)+ucharMax+1)
	l.position = make([]int16, uint(discsiz)+ucharMax+1)
	l.parent = make([]int16, discsiz*2)
	l.prev = make([]int16, discsiz*2)
	l.next = make([]int16, maxHashVal+1)
}

func (l *Lzh) initSlide() {
	var i int16
	for i = discsiz; i <= discsiz+int16(ucharMax); i++ {
		l.level[i] = 1
		l.position[i] = 0
	}
	for i = discsiz; i < discsiz*2; i++ {
		l.parent[i] = 0
	}
	l.avail = 1
	for i = 1; i < discsiz-1; i++ {
		l.next[i] = int16(i) + 1
	}
	l.next[discsiz-1] = 0
	for i = discsiz * 2; i <= maxHashVal; i++ {
		l.next[i] = 0
	}
}

func (l *Lzh) insertNode() {
	var q, r, j, t int16
	var c, t1, t2 byte
	if l.matchlen >= 4 {
		l.matchlen--
		r = (l.matchpos + 1) | discsiz
		q = l.parent[r]
		for q == 0 {
			r = l.next[r]
			q = l.parent[r]
		}
		for int(l.level[q]) >= l.matchlen {
			r = q
			q = l.parent[q]
		}
		t = q
		for l.position[t] < 0 {
			l.position[t] = l.pos
			t = l.parent[t]
		}
		if t < discsiz {
			l.position[t] = percflagOr(l.pos)
		}
	} else {
		q = int16(l.text[l.pos]) + discsiz
		c = l.text[l.pos+1]
		r = l.child(q, c)
		if r == 0 {
			l.makechild(q, c, l.pos)
			l.matchlen = 1
			return
		}
		l.matchlen = 2
	}
	for {
		if r >= discsiz {
			j = int16(maxmatch)
			l.matchpos = r
		} else {
			j = int16(l.level[r])
			l.matchpos = percflagNotand(l.position[r])
		}
		if l.matchpos >= l.pos {
			l.matchpos -= int16(discsiz)
		}

		t1 = l.text[int(l.pos)+l.matchlen]
		t2 = l.text[int(l.matchpos)+l.matchlen]
		t1pos := int(l.pos) + l.matchlen
		t2pos := int(l.matchpos) + l.matchlen
		for l.matchlen < int(j) {
			if t1 != t2 {
				l.split(r)
				return
			}
			l.matchlen++
			t1pos++
			t2pos++
			t1 = l.text[t1pos]
			t2 = l.text[t2pos]
		}
		if l.matchlen >= maxmatch {
			break
		}
		l.position[r] = l.pos
		q = r
		r = l.child(q, t1)
		if r == 0 {
			l.makechild(q, t1, l.pos)
			return
		}
		l.matchlen++
	}
	t = l.prev[r]
	l.prev[l.pos] = t
	l.next[t] = l.pos
	t = l.next[r]
	l.next[l.pos] = t
	l.prev[t] = l.pos
	l.parent[l.pos] = q
	l.parent[r] = 0
	l.next[r] = l.pos /* special use of next[] */
}

func (l *Lzh) hash(p, c int16) int16 {
	return (p + (c << (discbit - 9))) + discsiz*2
}

func (l *Lzh) makechild(q int16, c byte, r int16) {
	var h, t int16
	h = l.hash(q, int16(c))
	t = l.next[h]
	l.next[h] = r
	l.next[r] = t
	l.prev[t] = r
	l.prev[r] = h
	l.parent[r] = q
	l.childcount[q]++
}

func (l *Lzh) child(q int16, c byte) int16 {
	/* q's child for character c (NIL if not found) */
	var r int16
	r = l.next[l.hash(q, int16(c))]
	l.parent[0] = q /* sentinel */
	for l.parent[r] != q {
		r = l.next[r]
	}
	return r
}

func (l *Lzh) split(old int16) {
	var newval, t int16

	newval = l.avail
	l.avail = l.next[newval]
	l.childcount[newval] = 0
	t = l.prev[old]
	l.prev[newval] = t
	l.next[t] = newval
	t = l.next[old]
	l.next[newval] = t
	l.prev[t] = newval
	l.parent[newval] = l.parent[old]
	l.level[newval] = uint(l.matchlen)
	l.position[newval] = l.pos

	l.makechild(newval, l.text[int(l.matchpos)+l.matchlen], old)
	l.makechild(newval, l.text[int(l.pos)+l.matchlen], l.pos)
}

func (l *Lzh) getNextMatch() {
	l.remainder--
	l.pos++

	if l.pos == int16(discsiz)*2 {
		l.text = append(l.text[:0], l.text[discsiz:]...)
		n, err := l.freadCrc(&l.text, int(discsiz)+maxmatch, l.infilePtr, int(discsiz), &l.infile)
		l.infilePtr += n
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading crc : %s\n", err.Error())
		}
		l.remainder += n
		l.pos = discsiz
		fmt.Fprintf(os.Stderr, ".")
	}
	l.deleteNode()
	l.insertNode()
}

func (l *Lzh) deleteNode() {
	var q, r, s, t, u int16
	if l.parent[l.pos] == 0 {
		return
	}
	r = l.prev[l.pos]
	s = l.next[l.pos]
	l.next[r] = s
	l.prev[s] = r
	r = l.parent[l.pos]
	l.parent[l.pos] = 0
	l.childcount[r]--
	if r >= discsiz || l.childcount[r] > 1 {
		return
	}
	t = percflagNotand(l.position[r])
	if t >= l.pos {
		t -= discsiz
	}
	s = t
	q = l.parent[r]
	u = l.position[q]
	for percflagAnd(u) < 0 {
		u = percflagNotand(u)
		if u >= l.pos {
			u -= discsiz
		}
		if u > s {
			s = u
		}
		l.position[q] = (s | discsiz)
		q = l.parent[q]
		u = l.position[q]
	}
	if q < discsiz {
		if u >= l.pos {
			u -= discsiz
		}
		if u > s {
			s = u
		}
		l.position[q] = percflagOr(s | discsiz)
	}
	s = l.child(r, l.text[t+int16(l.level[r])])
	t = l.prev[s]
	u = l.next[s]
	l.next[t] = u
	l.prev[u] = t
	t = l.prev[r]
	l.next[t] = s
	l.prev[s] = t
	t = l.next[r]
	l.prev[t] = s
	l.next[s] = t
	l.parent[s] = l.parent[r]
	l.parent[r] = 0
	l.next[r] = l.avail
	l.avail = r
}
