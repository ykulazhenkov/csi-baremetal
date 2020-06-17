package utilwrappers

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"eos2git.cec.lab.emc.com/ECS/baremetal-csi-plugin.git/pkg/base/command"
	"eos2git.cec.lab.emc.com/ECS/baremetal-csi-plugin.git/pkg/base/linuxutils/fs"
)

// FSOperations is holds methods that consists of WrapFS methods
type FSOperations interface {
	// PrepareAndPerformMount composite methods which is prepare source and destination directories
	// and performs mount operation from src to dst
	PrepareAndPerformMount(src, dst string, bindMount bool) error
	fs.WrapFS
}

// FSOperationsImpl is a base implementation for FSOperation interface
type FSOperationsImpl struct {
	fs.WrapFS
	log *logrus.Entry
}

// NewFSOperationsImpl constructor for FSOperationsImpl and returns pointer on it
func NewFSOperationsImpl(e command.CmdExecutor, log *logrus.Logger) *FSOperationsImpl {
	return &FSOperationsImpl{
		WrapFS: fs.NewFSImpl(e),
		log:    log.WithField("component", "FSOperations"),
	}
}

// PrepareAndPerformMount implementation of FSOperations method
// create (if isn't exist) dst folder on node and perform mount from src to dst
// if bindMount set to true - mount operation will contain "--bind" option
// if error occurs and dst has created during current method call then dst will be removed
func (fsOp *FSOperationsImpl) PrepareAndPerformMount(src, dst string, bindMount bool) error {
	ll := fsOp.log.WithFields(logrus.Fields{
		"method": "PrepareAndPerformMount",
	})
	ll.Infof("Processing for source %s, destination %s", src, dst)

	// check whether dst path exist or no, if yes - assume that it is not a first provision for volume
	wasCreated := false
	_, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err = fsOp.MkDir(dst); err != nil {
			return err
		}
		wasCreated = true // if something went wrong we will remove path that had created based on that flag
	}

	// dst folder is exist, check whether it is a mount point
	if !wasCreated {
		alreadyMounted, err := fsOp.IsMountPoint(dst)
		if err != nil {
			_ = fsOp.RmDir(dst)
			return fmt.Errorf("unable to determine whether %s is a mountpoint or no: %v", dst, err)
		}
		if alreadyMounted {
			ll.Infof("%s has already mounted to %s", src, dst)
			return nil
		}
	}

	var opts string
	if bindMount {
		opts = fs.BindOption
	}
	if err := fsOp.Mount(src, dst, opts); err != nil {
		if wasCreated {
			_ = fsOp.RmDir(dst)
		}
		return fmt.Errorf("unable to mount %s to %s: %v", src, dst, err)
	}
	return nil
}