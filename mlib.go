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
    log.Debugf("CompressFile: path %s, newfilename %s", path, newfilename)
    if oldfile, err := os.Open(path); err != nil {
        return err
    } else {
        defer oldfile.Close()
        if newfile, err := os.Create(newfilename); err != nil {
            return err
        } else {
            defer newfile.Close()
            zipwriter := gzip.NewWriter(newfile)
            zipwriter.Comment = "rotated logfile"
            zipwriter.Name = filepath.Base(path)
            zipwriter.ModTime = time.Now()

            if nbytes, err := io.Copy(zipwriter, oldfile); err != nil {
                log.Errorf("gzip: %s", err)
                return err
            } else {
                log.Debugf("gzip of %s succeeded, nbytes: %d", path, nbytes)
            }
        }
    }
    return nil
}
