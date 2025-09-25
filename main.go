package main

import (
	"context"
	"fmt"

	"gomod.pri/golib/storage"
	"gomod.pri/golib/storage/types"
)

func main() {
	cli, _ := storage.NewStorage("Dialer", types.Config{
		Provider:  "obs",
		Endpoint:  "https://obs.ap-southeast-3.myhuaweicloud.com",
		AccessKey: "HPUA37GOWXZDK2QPVZYT",
		SecretKey: "zHJqLhYVLjFIthjtzgzkFxws3jAEa8Nt7nLYjXzn",
		Bucket:    "hprod-freeswitch-obs",
	})

	sign, err := cli.SignUrl(context.Background(), "Dialer/95/f84d/95f84d31-3ce6-402b-a70e-73826a340b59.wav", 3600)
	fmt.Println(sign, err)
}
