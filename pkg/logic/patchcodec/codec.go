package patchcodec

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// encoding:
// a uint64 header.
// if header <= max_uint32, uint32(header)==block ID
// if header >  max_uint32, header-max_uint32 == len(data), followed by data

type Encoder struct {
	w      io.Writer
	header []byte
	err    error
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w, header: make([]byte, 8)}
}

func (enc *Encoder) WriteBlockID(id uint32) (int, error) {
	binary.LittleEndian.PutUint64(enc.header, uint64(id))
	_, enc.err = enc.w.Write(enc.header) // n <- uint64 == 8 bytes
	return 8, enc.err
}

func (enc *Encoder) WriteBlock(data []byte) (int, error) {
	if len(data) > math.MaxUint32 {
		return -1, fmt.Errorf("data block too large: %d", len(data))
	}
	n := uint64(len(data)) + math.MaxUint32
	binary.LittleEndian.PutUint64(enc.header, n)
	_, enc.err = enc.w.Write(enc.header) // n <- uint64 == 8 bytes
	if enc.err != nil {
		return -1, enc.err
	}
	_, enc.err = enc.w.Write(data) // n <- len(data)
	return 8 + len(data), enc.err
}

type Decoder struct {
	r         io.Reader
	headerBuf []byte
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r, headerBuf: make([]byte, 8)}
}

func (dec *Decoder) Decode(
	onBlockID func(uint32) (int, error),
	onData func(io.Reader) (int, error),
) (int, error) {
	var (
		header   uint64
		blocklen int64
		written  int
		n        int
		err      error
	)
	for {
		_, err = io.ReadFull(dec.r, dec.headerBuf)
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
		header = binary.LittleEndian.Uint64(dec.headerBuf)
		if header <= uint64(math.MaxUint32) {
			n, err = onBlockID(uint32(header))
			written += n
			if err != nil {
				return written, err
			}
			continue
		}
		blocklen = int64(header - uint64(math.MaxUint32))
		n, err = onData(io.LimitReader(dec.r, blocklen))
		written += n
		if err != nil {
			return written, err
		}
	}
}
