package diecast

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
)

func loadStandardFunctionsTime(funcs FuncMap) funcGroup {
	return funcGroup{
		Name: `Time Functions`,
		Description: `Used for working with time and duration values. Among this collection are ` +
			`functions for converting values to times, formatting time values, and performing ` +
			`time-oriented calculations on those values.`,
		Functions: []funcDef{
			{
				Name:    `time`,
				Summary: `Return the given time formatted using a given format.  See [Time Formats](#time-formats) for acceptable formats.`,
				Arguments: []funcArg{
					{
						Name:        `time`,
						Type:        `string, integer`,
						Description: `The time you want to format.  Parsing is extremely flexible, and can handle dates represented as RFC3339,  RFC822, RFC1123, epoch, or epoch nanoseconds.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `time "01 May 10 13:04 -0500" "rfc3339"`,
						Return: `2010-05-01T13:04:00-05:00`,
					}, {
						Code:   `time 1136239445 "ansic"`,
						Return: `Mon Jan  2 22:04:05 2006`,
					},
				},
				Function: tmFmt,
			}, {
				Name:    `now`,
				Summary: `Return the current time, optionally formatted using the given format.`,
				Arguments: []funcArg{
					{
						Name:        `format`,
						Type:        `string`,
						Optional:    true,
						Description: `How to format the time output. See [Time Formats](#time-formats) for how to use format strings.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `now`,
						Return: `2010-05-01T13:04:00-05:00`,
					}, {
						Code:   `now "ansic"`,
						Return: `Mon Jan  2 22:04:05 2006`,
					},
				},
				Function: func(format ...string) (string, error) {
					return tmFmt(time.Now(), format...)
				},
			}, {
				Name: `addTime`,
				Summary: `Return a time with with given duration added to it.  Can specify time at to apply the change to. ` +
					`Defaults to the current time.`,
				Arguments: []funcArg{
					{
						Name:        `duration`,
						Type:        `string`,
						Description: `The duration to add to the time (can be negative to subtract a duration). See [Time Durations](#time-durations) for how to specify durations.`,
					}, {
						Name:        `from`,
						Type:        `string`,
						Optional:    true,
						Description: `If specified, this time will be parsed and modified instead of the current time.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `addTime "2h30m"`,
						Return: `2010-05-01T15:34:00-05:00`,
					}, {
						Code:   `addTime "-14d" "2011-10-21T12:00:00-08:00"`,
						Return: `2011-10-07T12:00:00-08:00`,
					},
				},
				Function: func(durationString string, atI ...interface{}) (time.Time, error) {
					at := time.Now()

					if len(atI) > 0 {
						if tm, err := stringutil.ConvertToTime(atI[0]); err == nil {
							at = tm
						} else {
							return time.Time{}, err
						}
					}

					if duration, err := timeutil.ParseDuration(durationString); err == nil {
						return at.Add(duration), nil
					} else {
						return time.Time{}, err
					}
				},
			}, {
				Name:    `ago`,
				Summary: `Return a new time subtracted by the given duration.`,
				Function: func(durationString string, fromTime ...time.Time) (time.Time, error) {
					from := time.Now()

					if len(fromTime) > 0 {
						from = fromTime[0]
					}

					if duration, err := timeutil.ParseDuration(durationString); err == nil {
						return from.Add(-1 * duration), nil
					} else {
						return time.Time{}, err
					}
				},
			}, {
				Name: `since`,
				Summary: `Return the amount of time that has elapsed since the given time, ` +
					`optionally rounded to the nearest time interval.`,
				Arguments: []funcArg{
					{
						Name:        `from`,
						Type:        `string`,
						Description: `The time to use when determining the duration that time has elapsed from.`,
					}, {
						Name:     `interval`,
						Type:     `string`,
						Optional: true,
						Description: `If specified, the resulting time duration will be rounded to the nearest ` +
							`interval of this unit.  Can be one of: "second" (nearest second), "minute" ` +
							`(nearest minute), "hour" (nearest hour), or "day" (nearest day).`,
						Valid: []funcArg{
							{
								Name: `second`,
							}, {
								Name: `minute`,
							}, {
								Name: `hour`,
							}, {
								Name: `day`,
							},
						},
					},
				},
				Examples: []funcExample{
					{
						Code:   `since "2010-05-01T13:04:15-05:00`,
						Return: ``,
					}, {
						Code:   `since "-14d" "2011-10-21T12:00:00-08:00"`,
						Return: `2011-10-07T12:00:00-08:00`,
					},
				},
				Function: func(at interface{}, interval ...string) (time.Duration, error) {
					if tm, err := stringutil.ConvertToTime(at); err == nil {
						since := time.Since(tm)

						if len(interval) > 0 {
							switch strings.ToLower(interval[0]) {
							case `s`, `sec`, `second`:
								since = since.Round(time.Second)
							case `m`, `min`, `minute`:
								since = since.Round(time.Minute)
							case `h`, `hr`, `hour`:
								since = since.Round(time.Hour)
							case `d`, `day`:
								since = since.Round(24 * time.Hour)
							}
						}

						return since, nil
					} else {
						return 0, err
					}
				},
			}, {
				Name:    `duration`,
				Summary: `Convert the given value from a duration (specified with the given unit) into the given time format.`,
				Function: func(value interface{}, unit string, formats ...string) (string, error) {
					if v, err := stringutil.ConvertToInteger(value); err == nil {
						duration := time.Duration(v)
						format := `timer`

						if len(formats) > 0 {
							format = formats[0]
						}

						switch unit {
						case `ns`, ``:
							break
						case `us`:
							duration = duration * time.Microsecond
						case `ms`:
							duration = duration * time.Millisecond
						case `s`:
							duration = duration * time.Second
						case `m`:
							duration = duration * time.Minute
						case `h`:
							duration = duration * time.Hour
						case `d`:
							duration = duration * time.Hour * 24
						case `y`:
							duration = duration * time.Hour * 24 * 365
						default:
							return ``, fmt.Errorf("Unrecognized unit %q", unit)
						}

						basetime := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
						basetime = basetime.Add(duration)

						return tmFmt(basetime, format)
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `isBefore`,
				Summary: `Return whether the first time occurs before the second one.`,
				Function: func(first interface{}, secondI ...interface{}) (bool, error) {
					return timeCmp(true, first, secondI...)
				},
			}, {
				Name:    `isAfter`,
				Summary: `Return whether the first time occurs after the second one.`,
				Function: func(first interface{}, secondI ...interface{}) (bool, error) {
					return timeCmp(false, first, secondI...)
				},
			}, {
				Name:    `isBetweenTimes`,
				Summary: `Return whether the current time is between two times [first, second).`,
				Function: func(firstI interface{}, secondI interface{}, tm ...interface{}) (bool, error) {
					now := time.Now()

					if len(tm) > 0 && tm[0] != nil {
						if t, err := stringutil.ConvertToTime(tm[0]); err == nil {
							now = t
						} else {
							return false, err
						}
					}

					if firstT, err := stringutil.ConvertToTime(firstI); err == nil {
						if secondT, err := stringutil.ConvertToTime(secondI); err == nil {
							if now.Equal(firstT) || now.After(firstT) {
								if now.Before(secondT) {
									return true, nil
								}
							}
						} else {
							return false, err
						}
					} else {
						return false, err
					}

					return false, nil
				},
			}, {
				Name:    `extractTime`,
				Summary: `Attempt to extract a time value from the given string.`,
				Function: func(baseI interface{}) (time.Time, error) {
					if base, err := stringutil.ToString(baseI); err == nil {
						if tm, err := stringutil.ConvertToTime(base); err == nil {
							return tm, nil
						}

						parts := strings.FieldsFunc(base, func(c rune) bool {
							switch c {
							case '/':
								return true
							default:
								return false
							}
						})

						for i := (len(parts) - 1); i >= 0; i-- {
							part := parts[i]

							split := strings.FieldsFunc(part, func(c rune) bool {
								return !unicode.IsLetter(c) && !unicode.IsNumber(c)
							})

							// try working backward...
							for j := len(split); j >= 0; j-- {
								try := strings.Join(split[0:j], `-`)

								if tm, err := stringutil.ConvertToTime(try); err == nil {
									return tm, nil
								}

								try = strings.Join(split[j:], `-`)

								if tm, err := stringutil.ConvertToTime(try); err == nil {
									return tm, nil
								}
							}

							// ...then forward
							for j := 0; j < len(split); j++ {
								try := strings.Join(split[0:j], `-`)

								if tm, err := stringutil.ConvertToTime(try); err == nil {
									return tm, nil
								}

								try = strings.Join(split[j:], `-`)

								if tm, err := stringutil.ConvertToTime(try); err == nil {
									return tm, nil
								}
							}
						}

						return time.Time{}, nil
					} else {
						return time.Time{}, err
					}
				},
			}, {
				Name:    `sunrise`,
				Summary: `Return the time of apparent sunrise at the given coordinates, optionally for a given time.`,
				Function: func(latitude float64, longitude float64, atTime ...interface{}) (time.Time, error) {
					sr, _, err := getSunriseSunset(latitude, longitude, atTime...)
					return sr, err
				},
			}, {
				Name:    `sunset`,
				Summary: `Return the time of apparent sunset at the given coordinates, optionally for a given time.`,
				Function: func(latitude float64, longitude float64, atTime ...interface{}) (time.Time, error) {
					_, ss, err := getSunriseSunset(latitude, longitude, atTime...)
					return ss, err
				},
			},
		},
	}
}
