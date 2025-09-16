package subscription

import (
	"strings"
	"time"
)

type YearMonth time.Time

func (ym *YearMonth) UnmarshalJSON(data []byte) error {
	// убираем кавычки
	str := strings.Trim(string(data), `"`)
	if str == "" {
		return nil
	}
	// парсим как "MM-2006"
	t, err := time.Parse("01-2006", str)
	if err != nil {
		return err
	}
	*ym = YearMonth(t)
	return nil
}

func (ym YearMonth) MarshalJSON() ([]byte, error) {
	t := time.Time(ym)
	return []byte(`"` + t.Format("01-2006") + `"`), nil
}

func (ym YearMonth) ToTime() time.Time {
	return time.Time(ym)
}
