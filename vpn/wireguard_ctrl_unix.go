// +build darwin

package vpn

import (
	"os"

	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/ipc"

	"github.com/mevansam/goutils/logger"
)

func (wgcs *WGCtrlService) startUAPI() error {

	var (
		err error

		fileUAPI *os.File
	)

	// open UAPI file
	if fileUAPI, err = ipc.UAPIOpen(wgcs.ifaceName); err != nil {
		logger.DebugMessage("UAPI listen error: %s", err.Error())
		return err
	}
	// listen for UAPI IPC
	if wgcs.uapi, err = ipc.UAPIListen(wgcs.ifaceName, fileUAPI); err != nil {
		wgcs.deviceLogger.Errorf("Failed to listen on UAPI socket: %v", err)
		return err
	}
	wgcs.deviceLogger.Verbosef("UAPI listener started")

	// listen for control data on UAPI IPC socket
	go func() {		
		for {
			conn, err := wgcs.uapi.Accept()
			if err != nil {
				wgcs.deviceLogger.Verbosef("UAPI listener stopped")
				if err == unix.EBADF {
					wgcs.err<-nil
				} else {
					wgcs.err<-err
				}
				return
			}
			go wgcs.device.IpcHandle(conn)
		}
	}()

	return nil
}
