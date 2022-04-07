package vpn

import (
	"time"

	"github.com/mevansam/goutils/logger"
)

func (w *wireguard) recordNetworkMetrics() (time.Duration, error) {

	var (
		err error
		
		sent, recd int64
	)

	if recd, sent, err = w.wgctrlClient.BytesTransmitted(); err != nil {
		logger.ErrorMessage(
			"wireguard.recordNetworkMetrics(): Failed to retrieve wireguard device information: %s", 
			err.Error(),
		)
		w.metricsError = err
		
	} else {
		if recd > 0 {
			w.recd.Set(recd)
		}
		if sent > 0 {
			w.sent.Set(sent)
		}
	}
	
	// record metrics every 500ms
	return 500, nil
}

func (w *wireguard) BytesTransmitted() (int64, int64, error) {
	return w.recd.Get(), w.sent.Get(), w.metricsError
}
