// +build darwin

// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "syscall"
    "time"
    "os"
    "fmt"
)

func SelectStdin(timeout_secs int64) (bool) {
    var r_fdset syscall.FdSet
    var timeout syscall.Timeval
    timeout.Sec = timeout_secs
    timeout.Usec = 0
    for i := 0; i < 16; i++ {
        r_fdset.Bits[i] = 0
    }
    r_fdset.Bits[0] = 1
    selerr := syscall.Select(1, &r_fdset, nil, nil, &timeout)
    if selerr != nil {
        fmt.Fprintf(os.Stderr, "%s\n", selerr)
    }
    // Is it really ready to read or did we time out?
    if r_fdset.Bits[0] == 1 {
        return true
    } else {
        return false
    }
}
