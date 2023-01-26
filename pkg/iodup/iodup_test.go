// iodup can help filter data arriving from an io.ReadCloser
// It allows wraping an existing io.ReadCloser provider and filter
// the data before exposing it it up the chain.
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

func multiTestReader(iod *Iodup, msg string) error {
	errCh := make(chan error)
	for i := 0; i < int(iod.numOutputs); i++ {
		go func(r io.ReadCloser, msg string) {
			errCh <- iotest.TestReader(r, []byte(msg))
		}(iod.Output[i], msg)
	}
	for i := 0; i < int(iod.numOutputs); i++ {
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
		r1 := New(ur)

		if err := iotest.TestReader(r1.Output[0], []byte("")); err != nil {
			t.Fatal(err)
		}

		if err := iotest.TestReader(r1.Output[1], []byte("")); err != nil {
			t.Fatal(err)
		}

		if err := r1.Output[0].Close(); err != nil {
			t.Errorf("iodup.Output[0].Close() error = %v", err)
		}

		if err := r1.Output[1].Close(); err != nil {
			t.Errorf("iodup.Output[1].Close() error = %v", err)
		}

		ur.closePanic = true
		r2 := New(ur, 2, 7)
		if err := iotest.TestReader(r2.Output[0], []byte("")); err != nil {
			t.Fatal(err)
		}
		if err := iotest.TestReader(r2.Output[1], []byte("")); err != nil {
			t.Fatal(err)
		}
		if err := r2.Output[0].Close(); err != nil {
			t.Errorf("iodup.Close() error = %v", err)
		}
		if err := r2.Output[1].Close(); err != nil {
			t.Errorf("iodup.Close() error = %v", err)
		}
	})
}

func TestNewNil(t *testing.T) {
	t.Run("", func(t *testing.T) {
		r := New(nil)
		if r.Output[0] != nil || r.Output[1] != nil {
			fmt.Printf("%v\n", r.Output)
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
		r := New(io.NopCloser(strings.NewReader(msg1)), 2, 4, 4)
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
			r := New(io.NopCloser(strings.NewReader(msg)))
			err := multiTestReader(r, msg)
			if err != nil {
				t.Fatal(err)
			}
		})

		for _, numBuf := range numBufs {
			t.Run("", func(t *testing.T) {
				r := New(io.NopCloser(strings.NewReader(msg)), 2, numBuf)
				err := multiTestReader(r, msg)
				if err != nil {
					t.Fatal(err)
				}
			})
			for _, sizeBuf := range sizeBufs {
				t.Run("", func(t *testing.T) {
					r := New(io.NopCloser(strings.NewReader(msg)), 2, numBuf, sizeBuf)
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
		r := New(io.NopCloser(strings.NewReader(msg1)), 2, 1, 2, 3)
		err := multiTestReader(r, msg1)
		if err == nil {
			t.Fatal("Expected error, but returned without one")
		}
	})

	t.Run("", func(t *testing.T) {
		r := New(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("", func(t *testing.T) {
		r := New(io.NopCloser(strings.NewReader(msg1)), 256)
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("", func(t *testing.T) {
		r := New(io.NopCloser(strings.NewReader(msg1)), 256, 3, 10)
		err := multiTestReader(r, msg1)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("", func(t *testing.T) {
		r := New(io.NopCloser(strings.NewReader(msg1)))
		err := multiTestReader(r, msg0)
		if err == nil {
			t.Error("Expected error, but returned without one")
		}
	})
}
