// +build darwin

// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "syscall"
    "os"
    "fmt"
    "time"
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

func StatfileSize(filename string) (size int64, err error) {
    var stat syscall.Stat_t
    size = 0
    err = syscall.Stat(filename, &stat)
    // use os.IsNotExist(err) to test if it doesn't exist
    if err != nil {
        return
    } else {
        // The file exists. Update our globals.
        size = stat.Size
    }
    return size, err
}
