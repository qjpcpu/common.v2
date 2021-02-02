package json

import (
	ejson "encoding/json"
	"time"
)

var tzBeijing *time.Location

func init() {
	tzBeijing, _ = time.LoadLocation("Asia/Shanghai")
}

const (
	DateTimeLayout = "2006-01-02 15:04:05"
	DateLayout     = "2006-01-02"
	TimeLayout     = "15:04:05"
)

type DateTime time.Time
type Date time.Time
type Time time.Time

func (t DateTime) MarshalJSON() ([]byte, error) {
	str := time.Time(t).Format(DateTimeLayout)
	return ejson.Marshal(str)
}

func (t *DateTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := ejson.Unmarshal(b, &s); err != nil {
		return err
	}
	tm, err := time.ParseInLocation(DateTimeLayout, s, time.Now().Location())
	if err != nil {
		return err
	}
	*t = DateTime(tm)
	return nil
}

func (t Date) MarshalJSON() ([]byte, error) {
	str := time.Time(t).Format(DateLayout)
	return ejson.Marshal(str)
}

func (t *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := ejson.Unmarshal(b, &s); err != nil {
		return err
	}
	tm, err := time.ParseInLocation(DateLayout, s, time.Now().Location())
	if err != nil {
		return err
	}
	*t = Date(tm)
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	str := time.Time(t).Format(TimeLayout)
	return ejson.Marshal(str)
}

func (t *Time) UnmarshalJSON(b []byte) error {
	var s string
	if err := ejson.Unmarshal(b, &s); err != nil {
		return err
	}
	tm, err := time.ParseInLocation(TimeLayout, s, time.Now().Location())
	if err != nil {
		return err
	}
	*t = Time(tm)
	return nil
}
