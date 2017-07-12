// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "time"
    "io"
    "sync"
)

var (
    mu sync.Mutex
)

func init() {
}

// An io.Writer that prefixes the output with a timestamp.
type TimeWriter struct {
    Writer io.Writer
    Utc bool
    Disable bool
}

// Our version of Write for the TimeWriter that prefixes the output
// with a timestamp, depending on some internal object properties.
// It is also thread-safe.
func (w TimeWriter) Write(b []byte) (n int, err error) {
    mu.Lock()
    defer mu.Unlock()
    if ! w.Disable {
        var now time.Time
        if w.Utc {
            now = time.Now().UTC()
        } else {
            now = time.Now()
        }
        io.WriteString(w.Writer, now.Format(time.StampMicro) + " ")
    }
    n, err = w.Writer.Write(b)
    return n, err
}
