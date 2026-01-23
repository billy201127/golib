package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"gomod.pri/golib/kmscred"
	_ "gomod.pri/golib/kmscred/aliyun"
	_ "gomod.pri/golib/kmscred/aws"
	_ "gomod.pri/golib/kmscred/huawei"
)

type options struct {
	Vendor    string
	Mode      string
	AccessKey string
	SecretKey string
	Region    string
	Secret    string
}

func parseFlags(args []string) (options, error) {
	fs := flag.NewFlagSet("kmscred", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var o options
	fs.StringVar(&o.Vendor, "vendor", getenv("KMSCRED_VENDOR", string(kmscred.VendorAliyun)), "kms vendor: aliyun|aws|huaweicloud")
	fs.StringVar(&o.Mode, "mode", getenv("KMSCRED_MODE", string(kmscred.ModeRAM)), "auth mode: ram|aksk")
	fs.StringVar(&o.AccessKey, "ak", os.Getenv("KMSCRED_AK"), "access key (for aksk mode)")
	fs.StringVar(&o.SecretKey, "sk", os.Getenv("KMSCRED_SK"), "secret key (for aksk mode)")
	fs.StringVar(&o.Region, "region", os.Getenv("KMSCRED_REGION"), "region")
	fs.StringVar(&o.Secret, "secret", os.Getenv("KMSCRED_SECRET"), "secret name")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return o, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	o, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, "flag parse error:", err)
		fmt.Fprintln(stderr, "usage:")
		fmt.Fprintln(stderr, "  -vendor aliyun|aws|huaweicloud -mode ram|aksk -region <region> -secret <name> [-ak <ak> -sk <sk>]")
		return err
	}

	if o.Secret == "" {
		return fmt.Errorf("secret name is required (set -secret or KMSCRED_SECRET)")
	}

	cfg := kmscred.Config{
		Vendor:    kmscred.Vendor(o.Vendor),
		Mode:      kmscred.Mode(o.Mode),
		AccessKey: o.AccessKey,
		SecretKey: o.SecretKey,
		Region:    o.Region,
	}

	client, err := kmscred.New(cfg)
	if err != nil {
		return err
	}

	_ = ctx
	val, err := client.GetSecretValue(o.Secret)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, val)
	return nil
}

func main() {
	if err := Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
