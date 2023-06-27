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
	_ "net/http/pprof"
	"sync"
	"time"
)

// An Iodup object maintining internal buffers and state
type Out struct {
	outBuf  []byte
	bufChan chan []byte
}

type Iodup struct {
	inBuf      []byte
	output     []*Out
	bufs       [][]byte
	inBufIndex uint
	src        io.ReadCloser
}

const (
	numOutputs uint = 2
	numBufs    uint = 128
	sizeBuf    uint = 8192
)

type IoDups struct {
	stashed []*Iodup   // Iodup cache
	mutex   sync.Mutex //protects Iodup cahce

	// the iodups defaults for this cache
	numOutputs uint
	numBufs    uint
	sizeBuf    uint
}

// NewIoDups creates iodups cache and set the iodups defaults
// The optional params may include 3 optional integers as parameters:
// 1. The number of outputs (default is 2)
// 2. The number of buffers which may be at least 3 (default is 1024)
// 3. The size of the buffers (default is 8192)
func NewIoDups(params ...uint) (iodups *IoDups) {
	iodups = new(IoDups)
	switch len(params) {
	case 0:
		iodups.numOutputs = 2
		iodups.numBufs = numBufs
		iodups.sizeBuf = sizeBuf
	case 1:
		iodups.numOutputs = params[0]
		if iodups.numOutputs < 2 {
			iodups.numOutputs = 2
		}
		iodups.numBufs = numBufs
		iodups.sizeBuf = sizeBuf
	case 2:
		iodups.numOutputs = params[0]
		if iodups.numOutputs < 2 {
			iodups.numOutputs = 2
		}
		iodups.numBufs = params[1]
		if iodups.numBufs < 3 {
			iodups.numBufs = numBufs
		}
		iodups.sizeBuf = sizeBuf
	case 3:
		iodups.numOutputs = params[0]
		if iodups.numOutputs < 2 {
			iodups.numOutputs = 2
		}
		iodups.numBufs = params[1]
		if iodups.numBufs < 3 {
			iodups.numBufs = numBufs
		}
		iodups.sizeBuf = params[2]
		if iodups.sizeBuf < 1 {
			iodups.sizeBuf = 1
		}
	default:
		panic("too many params in newStream")
	}
	return
}

// Create a New iodup to wrap an existing provider of an io.ReadCloser interface
// The new iodup.out[] will expose an io.ReadCloser interface
// A goroutine will be initiated to wait on the original provider Read interface
// and deliver the data to the Readwer using an internal channel
func (iodups *IoDups) NewIoDup(src io.ReadCloser) []*Out {
	var iod *Iodup
	output := make([]*Out, iodups.numOutputs)

	if src == nil {
		// all readers are nil
		return output
	}

	iodups.mutex.Lock()
	numIOds := len(iodups.stashed)
	if numIOds > 0 {
		iod = iodups.stashed[0]
		iodups.stashed = iodups.stashed[1:]
	} else {
		iod = new(Iodup)
		iod.bufs = make([][]byte, iodups.numBufs)
		for i := uint(0); i < iodups.numBufs; i++ {
			iod.bufs[i] = make([]byte, iodups.sizeBuf)
		}
	}
	iodups.mutex.Unlock()

	// create s.numOutputs outputs
	iod.output = output
	for j := uint(0); j < iodups.numOutputs; j++ {
		// we will maintain a maximum of s.numBufs-2 in s.bufChan + one buffer in s.inBuf + one buffer s.outBuf
		iod.output[j] = new(Out)
		iod.output[j].bufChan = make(chan []byte, iodups.numBufs-2)
	}
	iod.src = src
	iod.inBuf = iod.bufs[0]
	iod.inBufIndex = 0

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
				iod.inBufIndex = (iod.inBufIndex + 1) % iodups.numBufs
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

		for j := uint(0); j < iodups.numOutputs; j++ {
			iod.output[j].closeChannel()
		}

		iodups.stash(iod)
	}()

	return iod.output
}

func (iodups *IoDups) stash(iod *Iodup) {
	iodups.mutex.Lock()
	defer iodups.mutex.Unlock()
	iodups.stashed = append(iodups.stashed, iod)
}

func (iod *Iodup) forwardToOut(buf []byte) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("(iof *iodup) forwardToOut recovering from panic... %v\n", recovered)
		}

		// we never close bufChan from the receiver side, so we should never panic here!
		// closing the source is not a great idea...
	}()

	for j := 0; j < len(iod.output); j++ {
		iod.output[j].bufChan <- buf
	}
}
func (iod *Iodup) readFromSrc() (n int, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("(iof *iodup) readFromSrc recovering from panic... %v\n", recovered)

			// We close the internal channel to signal from the src to readers that we are done
			for j := 0; j < len(iod.output); j++ {
				close(iod.output[j].bufChan)
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
