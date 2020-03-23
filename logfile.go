// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "path/filepath"
    "os"
    "errors"
    "time"
    "fmt"
    "strings"
)

type LogFile struct {
    // The original requested path, which is a symlink.
    path string
    // The current filename in dir that is active.
    filename string
    // The directory that we are writing to.
    dir string
    // The file object associated with this logfile.
    file *os.File
    // The current size of the current logfile.
    size int64
    // file age?
    // compress on rotation?
    compress bool
    // rotation size?
    maxbytes int64
    // rotation time?
    maxseconds int64
}

func NewLogFile(path string, maxbytes, maxseconds int64) (*LogFile, error) {
    var new_log LogFile
    var err error
    // Clean it, and keep absolute
    path, err = filepath.Abs(path)
    if err != nil {
        return nil, err
    } else if path == "" {
        return nil, errors.New("path to logfile is required")
    } else {
        // Absolute paths only
        new_log.path = path
        new_log.filename = ""
        new_log.dir = filepath.Dir(path)
        new_log.file = nil
        new_log.size = 0
        new_log.compress = false
        new_log.maxbytes = maxbytes
        new_log.maxseconds = maxseconds
        // The directory must exist
        if _, err := os.Stat(new_log.dir); os.IsNotExist(err) {
            return nil, errors.New("directory " + new_log.dir + " does not exist")
        }
        return &new_log, nil
    }
}

func (logfile *LogFile) Open() error {
    var err error
    // FIXME: If the file already exists, stat it and check the size
    logfile.filename = logfile.gen_newname()
    current_path := logfile.CurrentPath()

    if stat, err := os.Stat(current_path); os.IsNotExist(err) {
        log.Debugf("%s does not yet exist", current_path)
    } else {
        log.Debugf("%s exists already, size %d", current_path, stat.Size())
        logfile.size = stat.Size()
    }

    log.Debugf("opening %s", current_path)
    logfile.file, err = os.OpenFile(current_path,
                                    os.O_WRONLY | os.O_CREATE | os.O_APPEND,
                                    0600)
    if err != nil {
        log.Errorf("open: %s", err)
        return err
    }

    if logfile.file == nil {
        panic("file is nil")
    }

    // Delete the symlink if it is present.
    log.Debugf("deleting %s", logfile.path)
    os.Remove(logfile.path)
    // Recreate it.
    log.Debugf("symlink from %s to %s", logfile.path, logfile.filename)
    if err := os.Symlink(logfile.filename, logfile.path); err != nil {
        log.Fatal(err)
        return err
    }
    return nil
}

func (logfile LogFile) Close() {
    logfile.file.Close()
}

func (logfile *LogFile) Write(b []byte) (int, error) {
    log.Debugf("Write: buffer is %d bytes", len(b))
    nbytes, err := logfile.file.Write(b)
    if err != nil {
        log.Errorf("Write error: %s", err)
        log.Errorf("file: %v", logfile.file)
    }
    logfile.size += int64(nbytes)
    return nbytes, err
}

func gettimesuffix(now time.Time) string {
    log.Debugf("gettimesuffix: now is %s", now)
    // http://fuckinggodateformat.com/
    // %Y%m%e%H%M%S
    // rfc 3339 - seriously??
    rv := now.Format("20060102150405")
    log.Debugf("returning format %s", rv)
    // The timesuffix returned should never have spaces in it
    if strings.Contains(rv, " ") {
        panic(rv)
    }
    return rv
}

func (logfile LogFile) gen_newname() string {
    filename := filepath.Base(logfile.path)
    newname := fmt.Sprintf("%s-%s.log",
                           strings.TrimSuffix(filename, ".log"),
                           gettimesuffix(time.Now()))
    log.Debugf("gen_newname: newname = %s", newname)
    return newname
}

// Return the current path to the current open file.
func (logfile LogFile) CurrentPath() string {
    return filepath.Join(logfile.dir, logfile.filename)
}

// Flag the file for compression on rotation, or not.
func (logfile *LogFile) SetCompression(compress bool) {
    logfile.compress = compress
}

func (logfile LogFile) NeedsRotation() bool {
    if logfile.maxbytes != 0 {
        if logfile.size >= logfile.maxbytes {
            log.Debugf("file %s needs rotation by size", logfile.path)
            return true
        }
    }
    return false
}

func (logfile LogFile) GetPath() string {
    return logfile.path
}

func (logfile LogFile) RotateFile() error {
    // All we really need to do is close and open again
    oldfile := logfile.CurrentPath()
    log.Debugf("oldfile is %s", oldfile)
    logfile.Close()
    if err := logfile.Open(); err != nil {
        return err
    }

    if logfile.compress {
        go CompressFile(oldfile)
    }

    return nil
}
