package main

import (
	"testing"
)

func TestSCardReadersListA(t *testing.T) {
	pcsc := NewPCSCInterface()
	err := pcsc.SCardEstablishContext()
	if err != nil {
		println("establish context: ", err.Error())
	}
	readers, err1 := pcsc.SCardListReadersA()
	if err1 != nil {
		println("list readers: ", err1.Error())
	}
	for _, reader := range readers {
		println(reader)
	}
	err2 := pcsc.SCardReleaseContext()
	if err2 != nil {
		println("release context: ", err2.Error())
	}
}
