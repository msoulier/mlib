// +build linux,amd64

// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "syscall"
    "time"
    "os"
    "fmt"
)

func SelectStdin(timeout_secs time.Duration) (bool) {
    var r_fdset syscall.FdSet
    var timeout syscall.Timeval
    timeout.Sec = int64(timeout_secs)
    timeout.Usec = 0
    for i := 0; i < 16; i++ {
        r_fdset.Bits[i] = 0
    }
    r_fdset.Bits[0] = 1
    _, selerr := syscall.Select(1, &r_fdset, nil, nil, &timeout)
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

func Statfile(outfileName string) (logfileSize int64, err error) {
    var stat syscall.Stat_t
    err = syscall.Stat(outfileName, &stat)
    if os.IsNotExist(err) {
        logfileSize = 0
    } else if err != nil {
        panic(err)
    } else {
        // The file exists. Update our globals.
        logfileSize = stat.Size
    }
    return logfileSize, err
}
