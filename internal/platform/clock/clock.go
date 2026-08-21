package clock

import "time"

type System struct{}

func (System) Now() time.Time { return time.Now().UTC() }

type Fixed struct{ Value time.Time }

func (f Fixed) Now() time.Time { return f.Value.UTC() }

func (f *Fixed) Advance(duration time.Duration) { f.Value = f.Value.Add(duration) }
