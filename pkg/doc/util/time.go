/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"time"
)

// TimeWithTrailingZeroMsec overrides marshalling of time.Time. It keeps a format of initial unmarshalling
// in case when date has zero a fractional second (e.g. ".000").
// For example, time.Time marshals 2018-03-15T00:00:00.000Z to 2018-03-15T00:00:00Z
// while TimeWithTrailingZeroMsec marshals to the initial 2018-03-15T00:00:00.000Z value.
type TimeWithTrailingZeroMsec struct {
	time.Time

	trailingZerosMsecCount int
}

// NewTime creates TimeWithTrailingZeroMsec without zero sub-second precision.
// It functions as a normal time.Time object.
func NewTime(t time.Time) *TimeWithTrailingZeroMsec {
	return &TimeWithTrailingZeroMsec{Time: t}
}

// NewTimeWithTrailingZeroMsec creates TimeWithTrailingZeroMsec with certain zero sub-second precision.
func NewTimeWithTrailingZeroMsec(t time.Time, trailingZerosMsecCount int) *TimeWithTrailingZeroMsec {
	return &TimeWithTrailingZeroMsec{
		Time:                   t,
		trailingZerosMsecCount: trailingZerosMsecCount,
	}
}

// MarshalJSON implements the json.Marshaler interface.
// The time is a quoted string in RFC 3339 format, with sub-second precision added if present.
// In case of zero sub-second precision presence, trailing zeros are included.
func (tm TimeWithTrailingZeroMsec) MarshalJSON() ([]byte, error) {
	timeBytes, err := tm.Time.MarshalJSON()
	if err != nil {
		return nil, err
	}

	if tm.trailingZerosMsecCount == 0 {
		return timeBytes, nil
	}

	return tm.marshalJSONWithTrailingZeroMsec()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in RFC 3339 format.
// In case of zero sub-second precision, it's kept and applied when e.g. unmarshal the time to JSON.
func (tm *TimeWithTrailingZeroMsec) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	err := (&tm.Time).UnmarshalJSON(data)
	if err != nil {
		return err
	}

	tm.keepTrailingZerosMsecFormat(string(data))

	return nil
}

// GetFormat returns customized time.RFC3339Nano with trailing zeros included in case of
// zero sub-second precision presence. Otherwise it returns time.RFC3339Nano.
func (tm TimeWithTrailingZeroMsec) GetFormat() string {
	if tm.trailingZerosMsecCount > 0 {
		return tm.getTrailingZeroIncludedFormat()
	}

	return time.RFC3339Nano
}

// ParseTimeWithTrailingZeroMsec parses a formatted string and returns the time value it represents.
// In case of zero sub-second precision, it's kept and applied when e.g. unmarshal the time to JSON.
func ParseTimeWithTrailingZeroMsec(timeStr string) (*TimeWithTrailingZeroMsec, error) {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, err
	}

	tWithTrailingZeroMsec := &TimeWithTrailingZeroMsec{Time: t}
	tWithTrailingZeroMsec.keepTrailingZerosMsecFormat(timeStr)

	return tWithTrailingZeroMsec, nil
}

func (tm TimeWithTrailingZeroMsec) marshalJSONWithTrailingZeroMsec() ([]byte, error) {
	format := tm.getTrailingZeroIncludedFormat()

	b := make([]byte, 0, len(format)+len(`""`))
	b = append(b, '"')
	b = tm.AppendFormat(b, format)
	b = append(b, '"')

	return b, nil
}

func (tm TimeWithTrailingZeroMsec) getTrailingZeroIncludedFormat() string {
	format := "2006-01-02T15:04:05."

	for i := 0; i < tm.trailingZerosMsecCount; i++ {
		format += "0"
	}

	format += "Z07:00"

	return format
}

func (tm *TimeWithTrailingZeroMsec) keepTrailingZerosMsecFormat(timeStr string) {
	msecFraction := false
	zerosCount := 0

	for i := 0; i < len(timeStr); i++ {
		c := int(timeStr[i])
		if !msecFraction {
			if c == '.' {
				msecFraction = true
			}

			continue
		}

		if c == 'Z' {
			if zerosCount > 0 {
				tm.trailingZerosMsecCount = zerosCount
			}

			break
		}

		if c != '0' {
			break
		}

		zerosCount++
	}
}
