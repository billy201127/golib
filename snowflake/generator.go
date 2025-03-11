package snowflake

import (
	"bytes"
	"fmt"
	"net"
	"os"

	"github.com/bwmarrin/snowflake"
)

const workerIdBits = 5
const datacenterIdBits = 5
const maxWorkerId = -1 ^ (-1 << workerIdBits)
const maxDatacenterId = -1 ^ (-1 << datacenterIdBits)
const maxMachineId = -1 ^ (-1 << (workerIdBits + datacenterIdBits))

var generator *snowflake.Node

func getMacAddr() (mac []byte, err error) {
	interfaces, err := net.Interfaces()

	if err != nil {
		return
	}

	for _, inter := range interfaces {
		if addresses, e := inter.Addrs(); e == nil {
			for _, address := range addresses {
				if ip, ok := address.(*net.IPNet); ok && !ip.IP.IsLoopback() {
					if v := ip.IP.To4(); v != nil {
						mac = inter.HardwareAddr
						return
					}
				}
			}
		}
	}

	return
}

// see com.baomidou.mybatisplus.core.toolkit.Sequence.getDatacenterId
func getDatacenterId() int64 {
	mac, err := getMacAddr()

	if err != nil || len(mac) == 0 {
		return 1
	}

	mac = bytes.ReplaceAll(mac, []byte(":"), []byte(""))
	id := ((0x000000FF & int64(mac[len(mac)-2])) | (0x0000FF00 & (int64(mac[len(mac)-1]) << 8))) >> 6
	return id % (maxDatacenterId + 1)
}

// see com.baomidou.mybatisplus.core.toolkit.Sequence.getMaxWorkerId
func getWorkerId(datacenterId int64) int64 {
	bs := []byte(fmt.Sprintf("%d%d", datacenterId, os.Getpid()))
	code := uint32(0)

	for _, b := range bs {
		code = 31*code + uint32(b)
	}

	return (int64(code) & 0xffff) % (maxWorkerId + 1)
}

func getMachineId() int64 {
	datacenterId := getDatacenterId()
	workerId := getWorkerId(datacenterId)
	return (datacenterId << datacenterIdBits) | workerId
}

func init() {
	machineId := max(0, min(maxMachineId, getMachineId()))
	generator, _ = snowflake.NewNode(machineId)
}

func Generate() int64 {
	return generator.Generate().Int64()
}

func GenerateString() string {
	return generator.Generate().String()
}
