/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iodup

import (
	"fmt"
	"io"
	"time"
)

// An Iodup object maintining internal buffers and state
type Out struct {
	outBuf  []byte
	bufChan chan []byte
}

type Iodup struct {
	inBuf      []byte
	Output     []*Out
	bufs       [][]byte
	inBufIndex uint
	numBufs    uint
	numOutputs uint
	sizeBuf    uint
	src        io.ReadCloser
}

// Create a New iodup to wrap an existing provider of an io.ReadCloser interface
// The new iodup.out[] will expose an io.ReadCloser interface
// The optional params may include 3 optional integers as parameters:
// 1. The number of outputs (default is 2)
// 2. The number of buffers which may be at least 3 (default is 1024)
// 3. The size of the buffers (default is 8192)
// A goroutine will be initiated to wait on the original provider Read interface
// and deliver the data to the Readwer using an internal channel
func New(src io.ReadCloser, params ...uint) (iod *Iodup) {
	var numOutputs, numBufs, sizeBuf uint
	switch len(params) {
	case 0:
		numOutputs = 2
		numBufs = 1024
		sizeBuf = 8192
	case 1:
		numOutputs = params[0]
		if numOutputs < 2 {
			numOutputs = 2
		}
		numBufs = 1024
		sizeBuf = 8192
	case 2:
		numOutputs = params[0]
		if numOutputs < 2 {
			numOutputs = 2
		}
		numBufs = params[1]
		if numBufs < 3 {
			numBufs = 1024
		}
		sizeBuf = 8192
	case 3:
		numOutputs = params[0]
		if numOutputs < 2 {
			numOutputs = 2
		}
		numBufs = params[1]
		if numBufs < 3 {
			numBufs = 1024
		}
		sizeBuf = params[2]
		if sizeBuf < 1 {
			sizeBuf = 1
		}
	default:
		panic("too many params in newStream")
	}

	iod = new(Iodup)
	iod.numOutputs = numOutputs
	iod.numBufs = numBufs
	iod.sizeBuf = sizeBuf
	iod.src = src

	// create s.numOutputs outputs
	iod.Output = make([]*Out, iod.numOutputs)

	if iod.src == nil {
		// all readers are nil
		return
	}

	for j := uint(0); j < iod.numOutputs; j++ {
		// we will maintain a maximum of s.numBufs-2 in s.bufChan + one buffer in s.inBuf + one buffer s.outBuf
		iod.Output[j] = new(Out)
		iod.Output[j].bufChan = make(chan []byte, iod.numBufs-2)
	}
	iod.bufs = make([][]byte, iod.numBufs)
	for i := uint(0); i < iod.numBufs; i++ {
		iod.bufs[i] = make([]byte, iod.sizeBuf)
	}
	iod.inBuf = iod.bufs[0]

	// start serving the io
	go func() {
		var n int
		var err error
		for err == nil {
			n, err = iod.readFromSrc()
			if n > 0 { // we have data
				iod.forwardToOut(iod.inBuf[:n])

				// ok, we now have a maximum of s.numBufs-2 in s.bufChan + one buffer s.outBuf
				// this means we have one free buffer to give to s.inBuf
				iod.inBufIndex = (iod.inBufIndex + 1) % iod.numBufs
				iod.inBuf = iod.bufs[iod.inBufIndex]
			} else { // no data
				if err == nil { // no data and no err.... bad, bad writer!!
					// hey, this io.Read interface is not doing as recommended!
					// "Implementations of Read are discouraged from returning a zero byte count with a nil error"
					// "Callers should treat a return of 0 and nil as indicating that nothing happened"
					// But even if nothing happened, we should not just abuse the CPU with an endless loop..
					time.Sleep(100 * time.Millisecond)
				}
			}
		}

		if err.Error() != "EOF" {
			fmt.Printf("(iof *iodup) Gorutine err %v\n", err)
		}

		for j := uint(0); j < iod.numOutputs; j++ {
			iod.Output[j].closeChannel()
		}
	}()

	return
}

func (iod *Iodup) forwardToOut(buf []byte) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("(iof *iodup) forwardToOut recovering from panic... %v\n", recovered)
		}

		// we never close bufChan from the receiver side, so we should never panic here!
		// closing the source is not a great idea...
	}()

	for j := uint(0); j < iod.numOutputs; j++ {
		iod.Output[j].bufChan <- buf
	}
}
func (iod *Iodup) readFromSrc() (n int, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("(iof *iodup) readFromSrc recovering from panic... %v\n", recovered)

			// We close the internal channel to signal from the src to readers that we are done
			for j := uint(0); j < iod.numOutputs; j++ {
				close(iod.Output[j].bufChan)
			}
			n = 0
			err = io.EOF
		}
	}()
	n, err = iod.src.Read(iod.inBuf)
	return n, err
}

// The io.Read interface of the iodup
func (out *Out) Read(dest []byte) (n int, err error) {
	var opened bool
	err = nil
	// Do we have bytes in our current buffer?
	if len(out.outBuf) == 0 {
		// Block until data arrives
		if out.outBuf, opened = <-out.bufChan; !opened && out.outBuf == nil {
			err = io.EOF
			n = 0
			return
		}
	}

	n = copy(dest, out.outBuf)
	// We copied n bytes, lets skip them for next time
	out.outBuf = out.outBuf[n:]
	return
}

// The io.Close interface of the iodup
func (out *Out) Close() error {
	// We ignore close from any of the readers - we close when the source closes
	return nil
}
func (out *Out) closeChannel() {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("(out *Out) closeChannel recovering from panic... %v\n", recovered)
		}
	}()
	close(out.bufChan)
}
