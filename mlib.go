// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "time"
    "io"
    "sync"
    "fmt"
)

var (
    mu sync.Mutex
)

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

// Format bytes in a human-readable format.
func Bytes2human(bytes int64) (string) {
    unit := "B"
    number := float64(bytes)
    if number > 1024.0 {
        number /= 1024.0
        unit = "kB"
    }
    if number > 1024.0 {
        number /= 1024.0
        unit = "MB"
    }
    if number > 1024.0 {
        number /= 1024.0
        unit = "GB"
    }
    if number > 1024.0 {
        number /= 1024.0
        unit = "TB"
    }
    return fmt.Sprintf("%.02f%s", number, unit)
}
