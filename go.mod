module github.com/appbricks/mycloudspace-common

go 1.17

replace github.com/appbricks/mycloudspace-common => ./

replace github.com/appbricks/cloud-builder => ../cloud-builder

replace github.com/mevansam/gocloud => ../../mevansam/gocloud

replace github.com/mevansam/goforms => ../../mevansam/goforms

replace github.com/mevansam/goutils => ../../mevansam/goutils

replace tailscale.com => ../tailscale

require (
	github.com/mevansam/goutils v0.0.0-00010101000000-000000000000
	golang.org/x/sys v0.0.0-20210816074244-15123e1e1f71
	golang.zx2c4.com/wireguard v0.0.0-20210905140043-2ef39d47540c
)

require (
	github.com/konsorten/go-windows-terminal-sequences v1.0.1 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/kr/text v0.1.0 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	golang.org/x/crypto v0.0.0-20210503195802-e9a32991a82e // indirect
	golang.org/x/net v0.0.0-20210504132125-bbd867fde50d // indirect
)
