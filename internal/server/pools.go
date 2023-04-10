package server

import (
	"bytes"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var timerPool sync.Pool

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

var cookieExpire = time.Now().Add(365 * 24 * time.Hour)

var cookiePool = sync.Pool{
	New: func() any {
		c := fasthttp.Cookie{}
		c.SetKey("whoami")
		c.SetExpire(cookieExpire)
		c.SetPath("/")
		return &c
	},
}

func acquireTimer(d time.Duration) *time.Timer {
	v := timerPool.Get()
	if v == nil {
		return time.NewTimer(d)
	}
	tm := v.(*time.Timer)
	if tm.Reset(d) {
		// active timer?
		return time.NewTimer(d)
	}
	return tm
}

func releaseTimer(tm *time.Timer) {
	if !tm.Stop() {
		// tm.Stop() returns false if the timer has already expired or been stopped.
		// We can't be sure that timer.C will not be filled after timer.Stop(),
		// see https://groups.google.com/forum/#!topic/golang-nuts/-8O3AknKpwk
		//
		// The tip from manual to read from timer.C possibly blocks caller if caller
		// has already done <-timer.C. Non-blocking read from timer.C with select does
		// not help either because send is done concurrently from another goroutine.
		return
	}
	timerPool.Put(tm)
}
