package lzh

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestCompress(t *testing.T) {
	b, _ := ioutil.ReadFile("compr_origin.txt")
	err := ioutil.WriteFile("compr.txt", b, 0644)
	l := NewLzh()
	if err != nil {
		t.Fatal(err)
	}
	f, _ := os.Create("archive.lha")
	err = l.Encode(f, "compr.txt")
	defer f.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecode(t *testing.T) {
	l := NewLzh()
	f, _ := os.Open("archive.lha")
	defer f.Close()
	err := l.Decode(f, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBin(t *testing.T) {

	var v2 int16 = 8236
	fmt.Printf("%b\n", v2)
	ov2 := percflagOr(v2) // doit fournir 0
	fmt.Printf("%d:%b\n", ov2, ov2)
	vv2 := percflagNotand(ov2)
	fmt.Printf("%d:%b\n", vv2, vv2)
	oo2 := percflagAnd(v2)
	fmt.Printf("%d:%b\n", oo2, oo2)

	var v int16 = -16534
	ov := percflagAnd(v)
	fmt.Printf("%d:%b\n", ov, ov)
}

/*
func percflag_or(v int16) int16 { // (short)v | PERC_FLAG
	var i int
	i = int(v) | 0x8000
	i |= 0xFFFF8000
	i += 1
	return int16(i)
}

func percflag_and(v int16) int16 { // (short)v & PERC_FLAG
	var i int
	i = int(v) & 0x8000
	//i |= 0xFFFF8000
	return int16(i) // a tester
}

func percflag_notand(v int16) int16 { // (short)v & ~PERC_FLAG
	var i int
	i = int(v) & ^0x8000
	return int16(i)
}
*/
