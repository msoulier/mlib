// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "time"
    "io"
    "os"
    "sync"
    "fmt"
    "github.com/op/go-logging"
    "compress/gzip"
    "path/filepath"
    "runtime"
    "strconv"
    "bytes"
    "syscall"
)

var (
    mu sync.Mutex
    log *logging.Logger
)

// Set the logger for this module.
func SetLogger(newlog *logging.Logger) {
    log = newlog
}

func init() {
    format := logging.MustStringFormatter(
        `%{time:2006-01-02 15:04:05.000-0700} %{level} [%{shortfile}] %{message}`,
        )
    stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
    stderrFormatter := logging.NewBackendFormatter(stderrBackend, format)
    stderrBackendLevelled := logging.AddModuleLevel(stderrFormatter)
    logging.SetBackend(stderrBackendLevelled)
    stderrBackendLevelled.SetLevel(logging.WARNING, "mlib")
    log = logging.MustGetLogger("mlib")
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

// Format bytes in a human-readable format.
func Bytes2human(bytes int64) string {
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

// Given the path to a file on disk, perform a gzip on the file.
func CompressFile(path string) error {
    newfilename := fmt.Sprintf("%s.gz", path)
    log.Debugf("mlib.CompressFile: path %s, newfilename %s", path, newfilename)
    if oldfile, err := os.Open(path); err != nil {
        return err
    } else {
        defer oldfile.Close()
        if newfile, err := os.Create(newfilename); err != nil {
            return err
        } else {
            defer newfile.Close()
            zipwriter := gzip.NewWriter(newfile)
            // Note, this defer order is important or the zipwriter buffer
            // will not be flushed to the file.
            defer zipwriter.Close()
            zipwriter.Comment = "rotated logfile"
            zipwriter.Name = filepath.Base(path)
            zipwriter.ModTime = time.Now()

            if nbytes, err := io.Copy(zipwriter, oldfile); err != nil {
                log.Errorf("gzip: %s", err)
                return err
            } else {
                log.Debugf("gzip of %s succeeded, nbytes: %d", path, nbytes)
                log.Debugf("unlinking %s", path)
                os.Remove(path)
            }
        }
    }
    return nil
}

func GetGID() uint64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    b = bytes.TrimPrefix(b, []byte("goroutine "))
    b = b[:bytes.IndexByte(b, ' ')]
    n, _ := strconv.ParseUint(string(b), 10, 64)
    return n
}

// Takes a time.Duration and prints it in days, hours, minutes, seconds, rounded down
func Duration2Human(diff time.Duration) string {
    days := 0
    hours := 0
    minutes := 0
    seconds := int(diff.Seconds())
    if seconds > 60 {
        minutes = int(seconds / 60)
        seconds -= minutes*60
    }
    if minutes > 60 {
        hours = int(minutes / 60)
        minutes -= hours*60
    }
    if hours > 24 {
        days = int(hours / 24)
        hours -= days*24
    }
    s := fmt.Sprintf("%d days, %d hours, %d minutes and %d seconds", days, hours, minutes, seconds)
    return s
}

// Copy a file from source to destination, with an optional move boolean
// to delete the original. It also uses flock to advisory lock the destination
// file in case another thread is doing the same, controlled by the lock boolean.
// FIXME: if source and destination are on the same drive, just rename
// Returns the number of bytes copied, and an error object.
func CopyFile(source, dest string, move, lock bool) (int64, error) {
	destfile, err := os.Create(dest)
    if err != nil {
		return 0, err
	}
    defer destfile.Close()
    sourcefile, err := os.Open(source)
    if err != nil {
        return 0, err
    }
    defer sourcefile.Close()
    // flock the destination file
    if lock {
        if err := syscall.Flock(int(destfile.Fd()), syscall.LOCK_EX); err != nil {
            return 0, err
        }
        defer syscall.Flock(int(destfile.Fd()), syscall.LOCK_UN)
    }
    nBytes, err := io.Copy(destfile, sourcefile)
    if err != nil {
        return nBytes, err
    }
    if move {
        // Delete the original.
        err := os.Remove(source)
        if err != nil {
            return nBytes, err
        }
    }
	return nBytes, nil
}
