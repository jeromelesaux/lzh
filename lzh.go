package lzh

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
)

var (
	ulongMax uint   = (math.MaxInt64*2 + 1)
	fnameMax        = 255 - 25
	initCrc  uint16 = 0
)

type Lzh struct {
	j          int
	level      []uint
	position   []int16
	parent     []int16
	next       []int16
	text       []byte
	childcount []uint
	prev       []int16
	avail      int16
	remainder  int
	matchlen   int
	pos        int16
	matchpos   int16
	bufsiz     uint
	outputPos  uint
	outputMask uint
	blocksize  uint16

	buf        []byte
	ptLen      []byte
	cLen       []byte
	ptCode     []uint16
	cCode      []uint16
	cFreq      []uint16
	pFreq      []uint16
	tFreq      []uint16
	ptTable    []uint16
	cTable     []uint16
	headersize byte
	headersum  byte
	header     []byte
	fileCrc    uint16
	namelen    int
	filename   []byte

	subbitbuf uint16
	bitcount  int
	origsize  int
	compsize  int

	infile     []byte
	infilePtr  int
	outfile    []byte
	arcfile    []byte
	arcfilePtr int
	unpackable bool
	bitbuf     uint32
	crc        uint16
	crctable   []uint
	buffer     []byte

	n        int
	heapsize int
	heap     []int16
	lenCnt   []uint16
	freq     []uint16
	sortptr  byte
	len      []byte
	left     []int16
	right    []int16
}

func NewLzh() *Lzh {
	return &Lzh{
		buf:      make([]byte, 0),
		ptLen:    make([]byte, npt),
		cLen:     make([]byte, nc),
		ptCode:   make([]uint16, npt),
		cCode:    make([]uint16, nc),
		cFreq:    make([]uint16, 2*nc-1),
		pFreq:    make([]uint16, 2*np-1),
		tFreq:    make([]uint16, 2*nt-1),
		ptTable:  make([]uint16, 256),
		cTable:   make([]uint16, 4096),
		crctable: make([]uint, ucharMax+1),
		heap:     make([]int16, nc+1),
		lenCnt:   make([]uint16, 17),
		freq:     make([]uint16, 0),
		len:      make([]byte, 0),
		left:     make([]int16, 2*nc-1),
		right:    make([]int16, 2*nc-1),
		buffer:   make([]byte, 0),
		outfile:  make([]byte, 0),
		header:   make([]byte, 255),
	}
}

func percflagOr(v int16) int16 { // (short)v | PERC_FLAG
	var i int
	i = int(v) | 0x8000
	i |= 0xFFFF8000
	return int16(i) //
}

func percflagAnd(v int16) int16 { // (short)v & PERC_FLAG
	var i int
	i = int(v) & 0x8000
	//i |= 0xFFFF8000
	return int16(i) // a tester -16534 doit retourner 0
}

func percflagNotand(v int16) int16 { // (short)v & ~PERC_FLAG
	var i int
	i = int(v) & ^0x8000
	return int16(i)
}

func (l *Lzh) Decode(f io.Reader, createFile bool) (err error) {
	l.arcfile, err = ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	l.makeCrctable()
	l.readHeader()
	l.buffer = make([]byte, l.origsize)
	if err := l.extract(true); err != nil {
		return err
	}
	if createFile {
		return ioutil.WriteFile(string(l.filename), l.outfile, 0644)
	}
	return nil
}

func (l *Lzh) DecodedBuffer() []byte {
	return l.outfile
}

func (l *Lzh) Encode(f io.Writer, inputFilename string) (err error) {
	// a completer
	l.makeCrctable()
	l.filename = []byte(inputFilename)
	copy(l.header[20:], l.filename[:])
	l.add(true)
	_, err = f.Write(l.outfile)
	return err
}

func (l *Lzh) listStart() {
	fmt.Fprintf(os.Stdout, "Filename         Original Compressed Ratio CRC Method\n")
}

func (l *Lzh) List(filename string) (err error) {
	fmt.Fprintf(os.Stdout, "%-14s", l.filename)
	if l.namelen > 14 {
		fmt.Fprintf(os.Stdout, "\n              ")
	}
	r := l.ratio(uint(l.compsize), uint(l.origsize))
	fmt.Fprintf(os.Stdout, " %10d %10d %d.%03d %04X %5.5s\n",
		l.origsize, l.compsize, r/1000, r%1000, l.fileCrc, l.header)
	return nil
}

func (l *Lzh) ratio(a, b uint) uint {
	var i int
	for i = 0; i < 3; i++ {
		if a <= ulongMax/10 {
			a *= 10
		} else {
			b /= 10
		}
	}
	if (a + (b >> 1)) < a {
		a >>= 1
		b >>= 1
	}
	if b == 0 {
		return 0
	}
	return ((a + (b >> 1)) / b)

}

func (l *Lzh) readHeader() error {
	l.headersize = l.arcfile[0]
	l.headersum = l.arcfile[1]
	l.arcfilePtr += 2
	err, _ := l.freadCrc(&l.header, 0, l.arcfilePtr, int(l.headersize), &l.arcfile)
	l.arcfilePtr += int(l.headersize)
	if err != nil {
		return err
	}
	if l.calcHeadersum() != uint(l.headersum) {
		return errors.New("Header sum error")
	}
	l.compsize = int(l.getFromHeader(5, 4))
	l.origsize = int(l.getFromHeader(9, 4))
	l.fileCrc = l.getFromHeader(int(l.headersize)-5, 2)
	l.namelen = int(l.header[19])
	l.filename = append(l.filename, l.header[20:]...)

	return nil
}

func (l *Lzh) calcHeadersum() uint {
	var s uint
	for i := 0; i < int(l.headersize); i++ {
		s += uint(l.header[i])
	}
	return s & 0xff
}

func (l *Lzh) getFromHeader(i, n int) uint16 {
	var s uint16
	n--
	for n >= 0 {
		s = (s << 8) + uint16(l.header[i+n]) /* little endian */
		n--
	}
	return s
}

func (l *Lzh) extract(toFile bool) (err error) {
	var method int
	var n uint16
	var extHeadersize uint16

	fmt.Printf("Extracting %s ", l.filename)
	l.crc = initCrc
	method = int(l.header[3])
	l.header[3] = byte(' ')
	if method != 45 && string(l.header[0:5]) != "-lh -" {
		fmt.Fprintf(os.Stderr, "Unknown method: %d\n", method)
		return errors.New("Unknown method")
	}
	extHeadersize = l.getFromHeader(int(l.headersize)-2, 2)
	for extHeadersize != 0 {
		fmt.Fprintf(os.Stdout, "There's an extended header of size %d.\n",
			extHeadersize)
		l.compsize -= int(extHeadersize)
		if len(l.arcfile[l.arcfilePtr:]) < int(extHeadersize)-2 {
			return errors.New("Can't read")
		}
		extHeadersize = uint16(l.arcfile[l.arcfilePtr])
		l.arcfilePtr++
		extHeadersize += uint16(l.arcfile[l.arcfilePtr] << 8)
		l.arcfilePtr++
	}
	l.crc = initCrc
	if method != 0 {
		l.decodeStart()
	}
	for l.origsize != 0 {
		n = uint16(l.origsize)
		if l.origsize > int(discsiz) {
			n = uint16(discsiz)
		}
		if method != '0' {
			l.decode(n, &l.buffer)
		} else {
			l.buffer = append(l.buffer, l.arcfile[l.arcfilePtr:n]...)
		}
		l.fwriteCrc(&l.buffer, len(l.outfile), 0, int(n), &l.outfile)
		fmt.Fprintf(os.Stdout, ".")
		l.origsize -= int(n)

	}
	if l.crc^initCrc != l.fileCrc {
		return errors.New("CRC error")
	}
	return nil
}

func (l *Lzh) getLine(s []byte, n int) int {
	var i int
	var c byte

	for j := 0; j <= len(s); j++ {
		c = s[j]
		if c != byte('\n') {
			if i < n {
				i++
				s[i] = c
			}
		}
	}

	return i
}

func (l *Lzh) putToHeader(i, n, x int) {
	n--
	for n >= 0 {
		l.header[i] = byte(x & 0xff)
		x >>= 8
		i++
		n--
	}
}

func (l *Lzh) writeHeader() {
	if len(l.outfile) < int(l.outputPos)+1 {
		l.outfile = append(l.outfile, 0)
	}
	l.outfile[int(l.outputPos)] = l.headersize
	l.outputPos++
	/* We've destroyed file_crc by null-terminating filename. */
	l.putToHeader(int(l.headersize)-5, 2, int(l.fileCrc))
	if len(l.outfile) < int(l.outputPos)+1 {
		l.outfile = append(l.outfile, 0)
	}
	l.outfile[int(l.outputPos)] = byte(l.calcHeadersum())
	l.outputPos++
	l.fwriteCrc(&l.header, int(l.outputPos), 0, int(l.headersize), &l.outfile) /* CRC not used */
}

func (l *Lzh) copy() error {

	var n int
	l.writeHeader()
	for l.compsize != 0 {
		n = int(discsiz)
		if l.compsize > int(discsiz) {
			n = l.compsize
		}
		copy(l.buffer, l.arcfile[l.arcfilePtr:n])
		l.arcfilePtr += n
		l.outfile = append(l.outfile, l.buffer...)
		l.compsize -= n
	}
	return nil
}

func (l *Lzh) skip() {
	l.arcfilePtr += l.compsize
}

func (l *Lzh) add(replaceFlag bool) (err error) {
	var r uint

	f, err := os.Open(string(l.filename))
	if err != nil {
		return err
	}
	defer f.Close()
	l.infile, err = ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	if replaceFlag {
		fmt.Fprintf(os.Stderr, "Replacing %s ", l.filename)
		l.skip()
	} else {
		fmt.Fprintf(os.Stderr, "Adding %s ", l.filename)
	}

	l.outputPos = 0
	l.namelen = len(l.filename)
	l.header[19] = byte(l.namelen)
	l.headersize = byte(25 + l.namelen)
	copy(l.header[0:], []byte("-lh5-")) /* compress */
	l.writeHeader()                     /* temporarily */

	l.origsize = 0
	l.compsize = 0
	l.unpackable = false
	l.crc = initCrc
	l.encode()
	if l.unpackable {
		l.header[3] = '0' /* store */
		l.store()
	}
	l.fileCrc = l.crc ^ initCrc
	l.putToHeader(5, 4, l.compsize)
	l.putToHeader(9, 4, l.origsize)
	copy(l.header[13:], []byte{0x0, 0x0, 0x0, 0x0, 0x20, 0x01})
	copy(l.header[l.headersize-3:], []byte{0x20, 0x0, 0x0})
	l.outputPos = 0
	l.writeHeader() /* true header */
	ENDODFILE := []byte{0x0}
	l.outfile = append(l.outfile, ENDODFILE...)
	r = l.ratio(uint(l.compsize), uint(l.origsize))
	fmt.Fprintf(os.Stderr, " %d.%d%%\n", r/10, r%10)
	return err /* success */
}

func (l *Lzh) store() {
	var n uint
	l.origsize = 0
	l.crc = initCrc
	copy(l.infile[:], l.buffer[discsiz:])
	l.fwriteCrc(&l.buffer, len(l.outfile), 0, int(n), &l.outfile)
	l.origsize += int(n)
	l.compsize = l.origsize
}
