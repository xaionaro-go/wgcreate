package wgcreate

import (
	"github.com/xaionaro-go/errors"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
	"syscall"
)

func createUserspace(ifaceName string, mtu uint32, logger *device.Logger) (resultIfaceName string, err error) {
	defer func() { err = errors.Wrap(err, ifaceName, mtu) }()

	nofileLimit := &syscall.Rlimit{}
	err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, nofileLimit)
	if err != nil {
		return
	}
	if nofileLimit.Cur < 65536 {
		nofileLimit.Cur = 65536
		if nofileLimit.Max < 65536 {
			nofileLimit.Max = 65536
		}
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, nofileLimit)
	}

	tunDev, err := tun.CreateTUN(ifaceName, int(mtu))
	if err != nil {
		return
	}

	resultIfaceName, err = tunDev.Name()
	if err != nil {
		return
	}

	wgDev := device.NewDevice(tunDev, logger)

	logger.Info.Print("userspace device started")

	uapiFile, err := ipc.UAPIOpen(resultIfaceName)
	if err != nil {
		return
	}

	uapi, err := ipc.UAPIListen(resultIfaceName, uapiFile)
	if err != nil {
		return
	}

	go func() {
		for {
			conn, err := uapi.Accept()
			if err != nil {
				logger.Info.Print("unable to accept UAPI connection", err)
				return
			}
			go wgDev.IpcHandle(conn)
		}
	}()

	logger.Info.Println("UAPI started")

	return
}
