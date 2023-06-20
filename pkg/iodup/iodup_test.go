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
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func multiTestReader(dup []*Out, msg string) error {
	errCh := make(chan error)
	for j := 0; j < len(dup); j++ {
		go func(r io.ReadCloser, msg string) {
			errCh <- iotest.TestReader(r, []byte(msg))
		}(dup[j], msg)
	}
	for j := 0; j < len(dup); j++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}
	return nil
}

type unothodoxReader struct {
	state      int
	closePanic bool
}

func (r *unothodoxReader) Read(buf []byte) (int, error) {
	if r.state == 0 {
		r.state = 1
		return 0, nil
	} else {
		r.state = 0
		return 0, errors.New("Aha!")
	}
}

func (r *unothodoxReader) Close() error {
	if r.closePanic {
		panic("I am already closed!")

	}
	return nil
}

func TestNewBadReader(t *testing.T) {
	ur := new(unothodoxReader)
	t.Run("unothodoxReader", func(t *testing.T) {
		ur.closePanic = false
		iodups := NewIoDups()
		r1 := iodups.NewIoDup(ur)

		if err := iotest.TestReader(r1[0], []byte("")); err != nil {
			t.Fatal(err)
		}

		if err := iotest.TestReader(r1[1], []byte("")); err != nil {
			t.Fatal(err)
		}

		if err := r1[0].Close(); err != nil {
			t.Errorf("iodup.Output[0].Close() error = %v", err)
		}

		if err := r1[1].Close(); err != nil {
			t.Errorf("iodup.Output[1].Close() error = %v", err)
		}

		ur.closePanic = true
		iodups = NewIoDups(2, 7)
		r2 := iodups.NewIoDup(ur)
		if err := iotest.TestReader(r2[0], []byte("")); err != nil {
			t.Fatal(err)
		}
		if err := iotest.TestReader(r2[1], []byte("")); err != nil {
			t.Fatal(err)
		}
		if err := r2[0].Close(); err != nil {
			t.Errorf("iodup.Close() error = %v", err)
		}
		if err := r2[1].Close(); err != nil {
			t.Errorf("iodup.Close() error = %v", err)
		}
	})
}

func TestNewNil(t *testing.T) {
	t.Run("", func(t *testing.T) {
		iodups := NewIoDups()
		r := iodups.NewIoDup(nil)
		if r[0] != nil || r[1] != nil {
			fmt.Printf("%v\n", r)
			t.Fatal("Expected nil in nil out")
		}
	})
}

func TestNew(t *testing.T) {
	const msg0 = ""
	const msg1 = "Now is the time for all good gophers."
	msg2Bytes := make([]byte, 256)
	rand.Read(msg2Bytes[:])

	msg2 := string(msg2Bytes[:])
	msgs := []string{msg0, msg1, msg2}

	numBufs := []uint{0, 1, 2, 3, 4, 8192}
	sizeBufs := []uint{0, 1, 2, 3, 4, 8192}

	t.Run("", func(t *testing.T) {
		iodups := NewIoDups(2, 4, 4)
		r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})

	for _, msg := range msgs {
		t.Run("", func(t *testing.T) {
			r := io.NopCloser(strings.NewReader(msg))
			err := iotest.TestReader(r, []byte(msg))
			if err != nil {
				t.Fatal(err)
			}
		})
		t.Run("", func(t *testing.T) {
			fmt.Printf("Test with msg len %d\n", len(msg))
			r := io.NopCloser(strings.NewReader(msg))
			err := iotest.TestReader(r, []byte(msg))
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("", func(t *testing.T) {
			iodups := NewIoDups()
			r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg)))
			err := multiTestReader(r, msg)
			if err != nil {
				t.Fatal(err)
			}
		})

		for _, numBuf := range numBufs {
			t.Run("", func(t *testing.T) {
				iodups := NewIoDups(2, numBuf)
				r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg)))
				err := multiTestReader(r, msg)
				if err != nil {
					t.Fatal(err)
				}
			})
			for _, sizeBuf := range sizeBufs {
				t.Run("", func(t *testing.T) {
					iodups := NewIoDups(2, numBuf, sizeBuf)
					r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg)))
					err := multiTestReader(r, msg)
					if err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}

	t.Run("", func(t *testing.T) {
		defer func() {
			r := recover()
			fmt.Printf("r is %v \n", r)
			if r == nil {
				t.Errorf("The code did not panic")
			}
		}()
		iodups := NewIoDups(2, 1, 2, 3)
		r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg1)
		if err == nil {
			t.Fatal("Expected error, but returned without one")
		}
	})

	t.Run("", func(t *testing.T) {
		iodups := NewIoDups()
		r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("", func(t *testing.T) {
		iodups := NewIoDups(256)
		r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("", func(t *testing.T) {
		iodups := NewIoDups(256, 3, 10)
		r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("", func(t *testing.T) {
		iodups := NewIoDups()
		r := iodups.NewIoDup(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg0)
		if err == nil {
			t.Error("Expected error, but returned without one")
		}
	})
}
