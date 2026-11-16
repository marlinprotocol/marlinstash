package service

import (
	"errors"
	"fmt"
	"marlinstash/tail"
	"marlinstash/types"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	lf "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const REFRESH time.Duration = 30

func beginTail(service string, host string, filepath string, offset uint64, datachan chan *types.EntryLine, killSignal chan uint64, inode uint64, log *lf.Entry) {
	log.WithField("offset", offset).Info("Tailer started")
	t, err := tail.TailFile(filepath, tail.Config{
		Location: &tail.SeekInfo{Offset: int64(offset)},
		Follow:   true,
	})
	if err != nil {
		log.Error("Tailer could not be started: ", err)
		killSignal <- inode
		return
	}
	for line := range t.Lines {
		if line.Err != nil {
			log.Info("Tailer stopped due to ", line.Err)
			killSignal <- inode
			return
		}
		datachan <- &types.EntryLine{
			Service: service,
			Host:    host,
			Inode:   inode,
			Offset:  line.Offset,
			Message: line.Text,
		}
	}
	// Lines is closed without an error line when the file is removed
	log.Info("Tailer stopped")
	killSignal <- inode
}

type logFile struct {
	path  string
	inode uint64
}

// findLogFiles returns the files under root whose name matches fileRegex.
// A missing or unreadable root is returned as an error. Files that can't be
// read or are removed while the directory is walked are skipped.
func findLogFiles(root string, fileRegex string, log *lf.Entry) ([]logFile, error) {
	re, err := regexp.Compile(fileRegex)
	if err != nil {
		return nil, fmt.Errorf("fileregex %q is not valid: %v", fileRegex, err)
	}

	rootInfo, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("logrootdir %q can't be read: %v", root, err)
	}
	if !rootInfo.IsDir() {
		return nil, fmt.Errorf("logrootdir %q is not a directory", root)
	}

	files := []logFile{}
	_ = filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if err != nil || f == nil {
			// f is nil when lstat fails, for example a file removed during the walk
			log.Info("Skipping ", path, ": ", err)
			return nil
		}
		if f.IsDir() || !re.MatchString(f.Name()) {
			return nil
		}
		// stat again to follow symlinks, tailer opens the file the link points to
		fileinfo, err := os.Stat(path)
		if err != nil {
			log.Info("Skipping ", path, ": ", err)
			return nil
		}
		if fileinfo.IsDir() {
			return nil
		}
		stat, ok := fileinfo.Sys().(*syscall.Stat_t)
		if !ok {
			log.Info("Skipping ", path, ": Not a syscall.Stat_t")
			return nil
		}
		files = append(files, logFile{path, stat.Ino})
		return nil
	})
	return files, nil
}

func Run(t types.Service, datachan chan *types.EntryLine, inodeOffsetReqChan, resetReqChan chan *types.InodeOffsetReq) {
	log := lf.WithField("service", t.Service)
	log.WithField("regex", t.FileRegex).WithField("refresh", REFRESH*time.Second).Info("Service started")

	invokedRoutines := make(map[uint64]bool)
	host := viper.GetString("host")
	killSignal := make(chan uint64)

	for {
		log.Info("Checking for new files to tail")
		files, err := findLogFiles(t.LogRootDir, t.FileRegex, log)
		if err != nil {
			log.Error("No files are read for this service, check logrootdir and fileregex in the config file: ", err)
		}
		for _, file := range files {
			err := startTailer(t, host, file, invokedRoutines, datachan, inodeOffsetReqChan, resetReqChan, killSignal, log)
			if err != nil {
				log.Error("Could not start tailer for ", file.path, ": ", err)
			}
		}

	WAIT:
		for {
			select {
			case inodeTailerKilled := <-killSignal:
				log.Info("Tailer has been killed for: ", inodeTailerKilled)
				invokedRoutines[inodeTailerKilled] = false
			case <-time.After(REFRESH * time.Second):
				// REDO FILEWALK HERE
				break WAIT
			}
		}
	}
}

func startTailer(t types.Service, host string, file logFile, invokedRoutines map[uint64]bool, datachan chan *types.EntryLine, inodeOffsetReqChan, resetReqChan chan *types.InodeOffsetReq, killSignal chan uint64, log *lf.Entry) error {
	inode := file.inode
	isCurrentlyRunning, ok := invokedRoutines[inode]
	if ok && isCurrentlyRunning {
		return nil
	}
	if !ok {
		log.Info("Running inode tailer for the first time for inode: ", inode)
	} else {
		log.Info("Seems like inode tailer is reinvoked for same inode again: ", inode)
	}
	respChan := make(chan *types.InodeOffset)
	inodeOffsetReqChan <- &types.InodeOffsetReq{
		Service: t.Service,
		Host:    host,
		Inode:   inode,
		Resp:    respChan,
	}
	log.Info("Waiting for DB to return offset for ", inode)
	offsetResponse, ok := <-respChan
	if !ok {
		// DB worker closes the channel when the query fails
		return errors.New("Offset query failure")
	}
	log.Info("DB responded with offset", offsetResponse, " for inode ", inode)

	// Mitigate inode reuse
	fi, err := os.Stat(file.path)
	if err != nil {
		return err
	}

	// Check size
	size := fi.Size()
	if uint64(size) < offsetResponse.Offset/10 {
		log.Info("Found large size difference between current file size and offset stored in DB for inode: ", inode)
		respChan := make(chan *types.InodeOffset)
		resetReqChan <- &types.InodeOffsetReq{
			Service: t.Service,
			Host:    host,
			Inode:   inode,
			Resp:    respChan,
		}
		_, ok := <-respChan
		if !ok {
			return errors.New("Reset failure")
		}
		offsetResponse.Offset = 0
	}

	go beginTail(t.Service, host, file.path, offsetResponse.Offset, datachan, killSignal, inode, log.WithField("file", file.path))
	invokedRoutines[inode] = true
	return nil
}
