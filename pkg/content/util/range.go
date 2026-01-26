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

	start, errStart := strconv.ParseInt(parts[0], 10, 64)
	end, errEnd := strconv.ParseInt(parts[1], 10, 64)

	if errStart != nil || errEnd != nil {
		return nil, ErrInvalidRange
	}

	if start < 0 || end < start {
		return nil, ErrInvalidRange
	}

	return &Range{
		Start:  start,
		Length: end - start + 1,
	}, nil
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
