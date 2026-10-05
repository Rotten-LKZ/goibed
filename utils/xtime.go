package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type XTime time.Time

func (t XTime) MarshalJSON() ([]byte, error) {
	tm := time.Time(t)
	if tm.IsZero() {
		return []byte("0"), nil
	}

	buf := make([]byte, 0, 16)
	return strconv.AppendInt(buf, tm.UnixMilli(), 10), nil
}

func (t *XTime) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	if timestamp == 0 {
		*t = XTime(time.Time{})
		return nil
	}
	*t = XTime(time.UnixMilli(timestamp))
	return nil
}

func (XTime) GormDataType() string { return "integer" }

func (t XTime) Value() (driver.Value, error) {
	tm := time.Time(t)
	if tm.IsZero() {
		return nil, nil
	}
	return tm.UnixMilli(), nil
}

func (t *XTime) Scan(v any) error {
	switch value := v.(type) {
	case int64:
		*t = XTime(time.UnixMilli(value))
		return nil
	case time.Time: // Legacy SQLite DATETIME values during migration.
		*t = XTime(value)
		return nil
	}
	return fmt.Errorf("failed to convert %v to XTime", v)
}
