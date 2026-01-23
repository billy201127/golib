package aliyun

import "gomod.pri/golib/kmscred"

func init() {
	kmscred.Register(kmscred.VendorAliyun, func(cfg kmscred.Config) (kmscred.Client, error) {
		return NewKMSClientByMode(string(cfg.Mode), cfg.AccessKey, cfg.SecretKey, cfg.Region)
	})
}

