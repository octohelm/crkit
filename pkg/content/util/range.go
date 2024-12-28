package util

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidRange = errors.New("invalid range")

func ParseRange(s string) (*Range, error) {
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidRange
	}

	r := &Range{}

	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, ErrInvalidRange
	}

	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, ErrInvalidRange
	}

	r.Start = start
	r.Length = end + 1 - r.Start

	return r, nil
}

type Range struct {
	Start  int64
	Length int64
}

func (r Range) IsZero() bool {
	return r.Length == 0
}

func (r Range) String() string {
	return fmt.Sprintf("%d-%d", r.Start, r.Start+r.Length-1)
}

func (r Range) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r *Range) UnmarshalText(d []byte) error {
	rr, err := ParseRange(string(d))
	if err != nil {
		return err
	}
	*r = *rr
	return nil
}
