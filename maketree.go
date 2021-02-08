package lzh

import (
	"fmt"
	"os"
)

var depth int

func (l *Lzh) countLen(i int) {
	if i < l.n {
		if depth < 16 {
			l.lenCnt[depth]++
		} else {
			l.lenCnt[16]++
		}
	} else {
		depth++
		l.countLen(int(l.left[i]))
		l.countLen(int(l.right[i]))
		depth--
	}
}

func (l *Lzh) makeLen(root int, codeparm *[]uint16) {
	var i, k int
	var cum uint
	for i = 0; i <= 16; i++ {
		l.lenCnt[i] = 0
	}
	//depth = 0
	l.countLen(root)
	for i = 16; i > 0; i-- {
		cum += uint(l.lenCnt[i]) << (16 - i)
	}
	for cum != (uint(1) << 16) {
		fmt.Fprintf(os.Stderr, "17")
		l.lenCnt[16]--
		for i = 15; i > 0; i-- {
			if l.lenCnt[i] != 0 {
				l.lenCnt[i]--
				l.lenCnt[i+1] += 2
				break
			}
		}
		cum--
	}
	for i = 16; i > 0; i-- {
		k = int(l.lenCnt[i])
		k--
		for k >= 0 {
			l.len[(*codeparm)[l.sortptr]] = byte(i)
			l.sortptr++
			k--
		}
	}
}

func (l *Lzh) downheap(i int) { /* priority queue; send i-th entry down heap */
	var j int
	var k int16
	k = l.heap[i]
	j = 2 * i
	for j <= l.heapsize {
		if j < l.heapsize && l.freq[l.heap[j]] > l.freq[l.heap[j+1]] {
			j++
		}
		if l.freq[k] <= l.freq[l.heap[j]] {
			break
		}
		l.heap[i] = l.heap[j]
		i = j
		j = 2 * i
	}
	l.heap[i] = k
}

func (l *Lzh) makeCode(n int, len *[]byte, code *[]uint16) {
	var i int
	var start [18]uint16

	start[1] = 0
	for i = 1; i <= 16; i++ {
		start[i+1] = (start[i] + uint16(l.lenCnt[i])) << 1
	}
	for i = 0; i < n; i++ {
		(*code)[i] = start[(*len)[i]]
		start[(*len)[i]]++
	}
}

func (l *Lzh) makeTree(nparm int, freqparm *[]uint16, lenparm *[]byte, codeparm *[]uint16) int {
	var i, j, k, avail int
	l.n = nparm
	l.freq = (*freqparm)
	l.len = (*lenparm)
	avail = l.n
	l.heapsize = 0
	l.heap[1] = 0
	for i = 0; i < l.n; i++ {
		l.len[i] = 0
		if l.freq[i] != 0 {
			l.heapsize++
			l.heap[l.heapsize] = int16(i)
		}
	}
	if l.heapsize < 2 {
		(*codeparm)[l.heap[1]] = 0
		return int(l.heap[1])
	}
	for i = l.heapsize / 2; i >= 1; i-- {
		l.downheap(i) /* make priority queue */
	}
	l.sortptr = 0
	for { /* while queue has at least two entries */
		i = int(l.heap[1]) /* take out least-freq entry */
		if i < l.n {
			(*codeparm)[l.sortptr] = uint16(i)
			l.sortptr++
		}
		l.heap[1] = l.heap[l.heapsize]
		l.heapsize--
		l.downheap(1)
		j = int(l.heap[1]) /* next least-freq entry */
		if j < l.n {
			(*codeparm)[l.sortptr] = uint16(j)
			l.sortptr++
		}

		k = avail /* generate new node */
		avail++
		l.freq[k] = l.freq[i] + l.freq[j]
		l.heap[1] = int16(k)
		l.downheap(1) /* put into queue */
		l.left[k] = int16(i)
		l.right[k] = int16(j)
		if l.heapsize <= 1 {
			break
		}
	}
	l.sortptr = 0
	l.makeLen(k, codeparm)
	l.makeCode(nparm, lenparm, codeparm)
	return k
}
