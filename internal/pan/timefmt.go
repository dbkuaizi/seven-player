package pan

import "strconv"

const timeLayoutRFC3339 = "2006-01-02T15:04:05Z07:00"

func toDecimalString(value int) string {
	return strconv.FormatInt(int64(value), 10)
}
