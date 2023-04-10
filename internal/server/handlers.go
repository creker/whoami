package server

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/pprishchepa/whoami/internal/random"
	"github.com/valyala/fasthttp"
)

func whoamiHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.Header.Add("X-Whoami-Handler", "whoami")
	_, _ = fmt.Fprintf(ctx, "Request: %s %s\n", ctx.Method(), ctx.URI().String())
	_, _ = fmt.Fprintf(ctx, "Host: %q\n", ctx.Host())
	_, _ = fmt.Fprintf(ctx, "IP: %q\n\n", ctx.RemoteIP())
}

func benchHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.Header.Add("X-Whoami-Handler", "bench")
	_, _ = ctx.WriteString("OK")
}

func dataHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.Header.Add("X-Whoami-Handler", "data")

	var err error

	var delay DurationValue
	if err = ParseDurationValue(&delay, b2s(ctx.QueryArgs().Peek("delay")), MaxDelayTime); err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		_, _ = ctx.WriteString(fmt.Sprintf("parse delay: %v", err))
		return
	}

	var headerSize SizeValue
	err = ParseSizeValue(&headerSize, b2s(ctx.QueryArgs().Peek("header")), MaxHeaderSize)
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		_, _ = ctx.WriteString(fmt.Sprintf("parse header size: %v", err))
		return
	}

	var bodySize SizeValue
	err = ParseSizeValue(&bodySize, b2s(ctx.QueryArgs().Peek("body")), MaxBodySize)
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		_, _ = ctx.WriteString(fmt.Sprintf("parse body size: %v", err))
		return
	}

	if delay.Valid {
		if delay.Range {
			delay.Max = time.Duration(random.NormFloat64(float64(delay.Min), float64(delay.Max)))
		}
		timer := acquireTimer(delay.Max)
		defer releaseTimer(timer)
		select {
		case <-timer.C:
		case <-ctx.Done():
			return
		}
	}

	if headerSize.Valid && headerSize.Range {
		headerSize.Max = int64(random.NormFloat64(float64(headerSize.Min), float64(headerSize.Max)))
	}
	if bodySize.Valid && bodySize.Range {
		bodySize.Max = int64(random.NormFloat64(float64(bodySize.Min), float64(bodySize.Max)))
	}

	var buf *bytes.Buffer
	if headerSize.Valid || bodySize.Valid {
		buf = bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		if headerSize.Max > bodySize.Max {
			buf.Grow(int(headerSize.Max))
		} else {
			buf.Grow(int(bodySize.Max))
		}
	}

	if headerSize.Valid {
		random.Write(buf, int(headerSize.Max))
		c := cookiePool.Get().(*fasthttp.Cookie)
		c.SetValue(b2s(buf.Bytes()))
		ctx.Response.Header.SetCookie(c)
		cookiePool.Put(c)
	}

	if bodySize.Valid {
		buf.Reset()
		random.Write(buf, int(bodySize.Max))
		_, _ = ctx.Write(buf.Bytes())
	} else {
		_, _ = ctx.WriteString("OK")
	}

	if buf != nil {
		bufferPool.Put(buf)
	}
}
