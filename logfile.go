// vim: ft=go ts=4 sw=4 et ai:
package mlib

import (
    "sort"
    "path"
    "path/filepath"
    "os"
    "errors"
    "time"
    "fmt"
    "strings"
    "regexp"
    "io/ioutil"
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
    // creation time in UTC
    created time.Time
    // compress on rotation?
    compress bool
    // rotation size?
    maxbytes int64
    // rotation time?
    maxseconds int64
    // number of logs to keep
    nlogs int
}

func NewLogFile(path string, maxbytes, maxseconds int64, nlogs int) (*LogFile, error) {
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
        new_log.created = time.Now().UTC()
        new_log.compress = false
        new_log.maxbytes = maxbytes
        new_log.maxseconds = maxseconds
        new_log.nlogs = nlogs
        // The directory must exist
        if _, err := os.Stat(new_log.dir); os.IsNotExist(err) {
            return nil, errors.New("directory " + new_log.dir + " does not exist")
        }
        return &new_log, nil
    }
}

/*
 * Private methods **************************************************
 */

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

func (logfile LogFile) parse_creation() time.Time {
    var datetime = regexp.MustCompile(`(\d{14})\.log`)
    if datetime.MatchString(logfile.filename) {
        // Matched name.
        datetime_string := datetime.FindStringSubmatch(logfile.filename)[1]
        log.Debugf("parsed out datetime: %q", datetime_string)
        zone, _ := time.Now().Zone()
        log.Debugf("timezone is now %s", zone)
        t, err := time.Parse("20060102150405 MST", datetime_string + " " + zone)
        if err == nil {
            log.Debugf("time %q", t)
            return t.UTC()
        } else {
            log.Errorf("time parse error on %s, using now: %s", logfile.filename, err)
            return time.Now().UTC()
        }
    } else {
        log.Debug("failed to match time string, using now")
        return time.Now().UTC()
    }
}
/*
 * End Private methods **********************************************
 */

/*
 * Public methods ***************************************************
 */

func (logfile *LogFile) Open() error {
    var err error
    oldname := logfile.filename
    for i := 0; i < 3; i++ {
        logfile.filename = logfile.gen_newname()
        if oldname == logfile.filename {
            log.Errorf("newname matches oldname: %s", oldname)
            time.Sleep(time.Second*2)
            continue
        } else {
            break
        }
    }
    current_path := logfile.CurrentPath()

    if stat, err := os.Stat(current_path); os.IsNotExist(err) {
        log.Debugf("%s does not yet exist", current_path)
        logfile.created = time.Now().UTC()
    } else {
        log.Debugf("%s exists already, size %d", current_path, stat.Size())
        logfile.size = stat.Size()
        // What to do with created if the file already exists? We can't rely
        // on the filesystem storing created time.
        // Luckily we put the file creation date and time into the filename.
        logfile.created = logfile.parse_creation()
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
            log.Debugf("file %s needs rotation by size", logfile.CurrentPath())
            return true
        }
    }
    if logfile.maxseconds != 0 {
        now := time.Now().UTC()
        duration := now.Sub(logfile.created)
        log.Debugf("it has been %d seconds since file creation", duration.Seconds())
        if int64(duration.Seconds()) > logfile.maxseconds {
            log.Debugf("file %s needs rotation by time", logfile.CurrentPath())
            return true
        }
    }
    return false
}

func (logfile LogFile) GetPath() string {
    return logfile.path
}

func (logfile LogFile) RotateFile() (*LogFile, error) {
    // All we really need to do is close and open again
    oldfile := logfile.CurrentPath()
    log.Debugf("oldfile is %s", oldfile)
    logfile.Close()
    if err := logfile.Open(); err != nil {
        return nil, err
    }

    if logfile.compress {
        go CompressFile(oldfile)
    }

    logfile.DeleteOldFiles()

    // Reset metadata on the file.
    logfile.size = 0
    logfile.created = time.Now().UTC()

    return &logfile, nil
}

// For sorting FileInfo objects by Name
type ByName []os.FileInfo
func (a ByName) Len() int               { return len(a) }
func (a ByName) Swap(i, j int)          { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool     { return a[i].Name() < a[j].Name() }

func (logfile LogFile) DeleteOldFiles() {
    if files, err := ioutil.ReadDir(logfile.dir); err != nil {
        log.Errorf("unable to open directory %s: %s", logfile.dir, err)
        return
    } else {
        dirfiles := make([]os.FileInfo, 0, 100)
        for _, dirfile := range files {
            log.Debugf("file in log dir %s: %s", logfile.dir, dirfile.Name())
            // skip dotfiles
            if strings.HasPrefix(dirfile.Name(), ".") {
                continue
            } else if regular := dirfile.Mode().IsRegular(); ! regular {
                log.Debugf("%s/%s is not a regular file, skipping", logfile.dir, dirfile.Name())
                continue
            }
            // should end in log or log.gz FIXME: configurable
            if strings.HasSuffix(dirfile.Name(), ".log") ||
               strings.HasSuffix(dirfile.Name(), ".log.gz") {
                dirfiles = append(dirfiles, dirfile)
            }
        }
        sort.Reverse(ByName(dirfiles))
        log.Debugf("dirfiles is now %#v", dirfiles)
        if len(dirfiles) > logfile.nlogs {
            todelete := len(dirfiles) - logfile.nlogs
            log.Info("logdir", logfile.dir, "- need to delete", todelete)
            for _, file := range dirfiles[:todelete] {
                todelete_path := path.Join(logfile.dir, file.Name())
                if err := os.Remove(todelete_path); err != nil {
                    log.Errorf("unlink: %s: %s", todelete_path, err)
                }
            }
        }
    }
}

/*
 * End Public methods ***********************************************
 */
