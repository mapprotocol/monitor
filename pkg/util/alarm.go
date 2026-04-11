package util

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mapprotocol/mapo-lib/alarm"
)

var Env = ""

func init() {
	Env = os.Getenv("compass")
	alarm.Init(Env, os.Getenv("hooks"))
}

func Alarm(ctx context.Context, msg string) {
	fmt.Printf("%s [ALARM] %s\n", time.Now().Format("2006-01-02T15:04:05"), msg)
	_ = alarm.Send(ctx, msg)
}
