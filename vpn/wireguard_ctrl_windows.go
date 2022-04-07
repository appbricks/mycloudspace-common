// +build windows

package vpn

import (
	"golang.zx2c4.com/wireguard/ipc"

	"github.com/mevansam/goutils/logger"
)

func (wgcs *WGCtrlService) startUAPI() error {

	var (
		err error
	)

	// listen for UAPI IPC
	if wgcs.uapi, err = ipc.UAPIListen(wgcs.ifaceName); err != nil {
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
				logger.DebugMessage("WGCtrlService.startUAPI(): UAPI error when stopped: %s", err.Error())

				wgcs.err<-nil
				return
			}
			go wgcs.device.IpcHandle(conn)
		}
	}()
	
	return nil
}