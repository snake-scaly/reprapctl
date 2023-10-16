package yall_test

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

var someTime = time.Date(2020, 11, 22, 12, 34, 56, 789, time.UTC)
var someCtx = context.Background()

func rec(args ...any) slog.Record {
	pc, _, _, _ := runtime.Caller(0)
	r := slog.NewRecord(someTime, slog.LevelInfo, "msg", pc)
	r.Time = someTime
	r.Add(args...)
	return r
}
