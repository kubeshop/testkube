package types

import "time"

const DefaultDurationFormat = "15:04:05"
const InvalidDurationString = "invalid"

func FormatDuration(in string) string {
	if in == "" {
		return ""
	}
	duration, err := time.ParseDuration(in)
	if err != nil {
		return InvalidDurationString + " " + err.Error()
	}

	return FormattedDuration(duration).Format()
}

func FormatDurationMs(in string) int32 {
	if in == "" {
		return 0
	}
	duration, err := time.ParseDuration(in)
	if err != nil {
		return 0
	}

	return int32(duration / time.Millisecond)
}

type FormattedDuration time.Duration

func (t FormattedDuration) Format(formats ...string) string {
	format := DefaultDurationFormat
	if len(formats) > 0 {
		format = formats[0]
	}

	z := time.Unix(0, 0).UTC()
	return z.Add(time.Duration(t)).Format(format)
}
