//go:build linux || darwin
// +build linux darwin

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
		logger.ErrorMessage("WGCtrlService.startUAPI(): UAPI listen error: %s", err.Error())
		return err
	}
	// listen for UAPI IPC
	if wgcs.uapi, err = ipc.UAPIListen(wgcs.ifaceName, fileUAPI); err != nil {
		logger.ErrorMessage("WGCtrlService.startUAPI(): Failed to listen on UAPI socket: %v", err)
		return err
	}
	logger.DebugMessage("WGCtrlService.startUAPI(): UAPI listener started")

	// listen for control data on UAPI IPC socket
	go func() {		
		for {
			conn, err := wgcs.uapi.Accept()
			if err != nil {
				logger.DebugMessage("WGCtrlService.startUAPI(): UAPI listener stopped")
				if err == unix.EBADF {
					wgcs.err<-nil
				} else {
					logger.ErrorMessage("WGCtrlService.startUAPI(): UAPI error when stopped: %s", err.Error())
					wgcs.err<-err
				}
				return
			}
			go wgcs.device.IpcHandle(conn)
		}
	}()

	return nil
}
