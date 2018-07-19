package main

import (
	"flag"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rtc"
	"os"
)

func main() {
	flag.Usage = func() {
		fmt.Println(fmt.Sprintf("Usage: %v <options>", os.Args[0]))
		fmt.Println(fmt.Sprintf("	--listen		listen port"))
		fmt.Println(fmt.Sprintf("	--access-key-id		the id of access key"))
		fmt.Println(fmt.Sprintf("	--access-key-secret	the secret of access key"))
		fmt.Println(fmt.Sprintf("	--gslb			the gslb url"))
		fmt.Println(fmt.Sprintf("Example:"))
		fmt.Println(fmt.Sprintf("	%v --listen=8080 --access-key-id=OGAEkdiL62AkwSgs --access-key-secret=4JaIs4SG4dLwPsQSwGAHzeOQKxO6iw --appid=iwo5l81k --gslb=https://rgslb.rtc.aliyuncs.com", os.Args[0]))
	}

	listen := flag.String("listen", "", "listen port")
	accessKeyId := flag.String("access-key-id", "", "access key id")
	accessKeySecret := flag.String("access-key-secret", "", "access key secret")
	regionId := "cn-hangzhou"

	flag.Parse()
	if *listen == "" || *accessKeyId == "" || *accessKeySecret == "" {
		flag.Usage()
		os.Exit(-1)
	}

	client, err := rtc.NewClientWithAccessKey(regionId, *accessKeyId, *accessKeySecret)
	if err != nil {
		panic(err)
	}

	_ = client
}
