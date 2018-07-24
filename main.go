package main

import (
	cr "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/rtc"
	oh "github.com/ossrs/go-oryx-lib/http"
	ol "github.com/ossrs/go-oryx-lib/logger"
	"net/http"
	"os"
	"sync"
)

type ChannelAuth struct {
	AppId      string
	ChannelId  string
	Nonce      string
	Timestamp  int64
	ChannelKey string
}

func CreateChannel(appId, channelId,
	regionId, accessKeyId, accessKeySecret string,
) (*ChannelAuth, error) {
	client, err := rtc.NewClientWithAccessKey(
		regionId, accessKeyId, accessKeySecret)
	if err != nil {
		return nil, err
	}

	client.EnableAsync(5, 10)

	r := rtc.CreateCreateChannelRequest()
	r.AppId = appId
	r.ChannelId = channelId

	rrs, errs := client.CreateChannelWithChan(r)
	select {
	case err := <-errs:
		return nil, err
	case r0 := <-rrs:
		return &ChannelAuth{
			AppId:      appId,
			ChannelId:  channelId,
			Nonce:      r0.Nonce,
			Timestamp:  int64(r0.Timestamp),
			ChannelKey: r0.ChannelKey,
		}, nil
	}
}

func BuildToken(channel, channelkey,
	appid, uid, session, nonce string, timestamp int64,
) (token string, err error) {
	h := sha256.New()
	if _, err = h.Write([]byte(channel)); err != nil {
		return "", err
	}
	if _, err = h.Write([]byte(channelkey)); err != nil {
		return "", err
	}
	if _, err = h.Write([]byte(appid)); err != nil {
		return "", err
	}
	if _, err = h.Write([]byte(uid)); err != nil {
		return "", err
	}
	if _, err = h.Write([]byte(session)); err != nil {
		return "", err
	}
	if _, err = h.Write([]byte(nonce)); err != nil {
		return "", err
	}
	if _, err = h.Write([]byte(fmt.Sprint(timestamp))); err != nil {
		return "", err
	}
	s := h.Sum(nil)
	token = hex.EncodeToString(s)
	return
}

// generate a random string
func BuildRandom(length int) string {
	if length <= 0 {
		return ""
	}

	b := make([]byte, length/2+1)
	_, _ = cr.Read(b)
	s := hex.EncodeToString(b)
	return s[:length]
}

func main() {
	flag.Usage = func() {
		fmt.Println(fmt.Sprintf("Usage: %v <options>", os.Args[0]))
		fmt.Println(fmt.Sprintf("	--appid			the id of app"))
		fmt.Println(fmt.Sprintf("	--listen		listen port"))
		fmt.Println(fmt.Sprintf("	--access-key-id		the id of access key"))
		fmt.Println(fmt.Sprintf("	--access-key-secret	the secret of access key"))
		fmt.Println(fmt.Sprintf("	--gslb			the gslb url"))
		fmt.Println(fmt.Sprintf("Example:"))
		fmt.Println(fmt.Sprintf("	%v --listen=8080 --access-key-id=OGAEkdiL62AkwSgs --access-key-secret=4JaIs4SG4dLwPsQSwGAHzeOQKxO6iw --appid=iwo5l81k --gslb=https://rgslb.rtc.aliyuncs.com", os.Args[0]))
	}

	var appid, listen, accessKeyId, accessKeySecret, gslb string
	flag.StringVar(&appid, "appid", "", "app id")
	flag.StringVar(&listen, "listen", "", "listen port")
	flag.StringVar(&accessKeyId, "access-key-id", "", "access key id")
	flag.StringVar(&accessKeySecret, "access-key-secret", "", "access key secret")
	flag.StringVar(&gslb, "gslb", "", "gslb url")
	regionId := "cn-hangzhou"

	flag.Parse()
	if appid == "" || listen == "" || accessKeyId == "" || accessKeySecret == "" || gslb == "" {
		flag.Usage()
		os.Exit(-1)
	}

	ol.Tf(nil, "Server listen=%v, appid=%v, akId=%v, akSecret=%v, gslb=%v, region=%v",
		listen, appid, accessKeyId, accessKeySecret, gslb, regionId)

	channels := make(map[string]*ChannelAuth)
	var lock sync.Mutex

	pattern := "/app/v1/login"
	ol.Tf(nil, "Handle %v", pattern)
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers.
		if o := r.Header.Get("Origin"); o != "" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,HEAD,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Server,Range,Content-Length,Content-Range")
			w.Header().Set("Access-Control-Allow-Headers", "Origin,Range,Accept-Encoding,Referer,Cache-Control,X-Proxy-Authorization,X-Requested-With,Content-Type")
		}

		// For matched OPTIONS, should directly return without response.
		if r.Method == "OPTIONS" {
			return
		}

		q := r.URL.Query()
		channelId, user := q.Get("room"), q.Get("user")
		channelUrl := fmt.Sprintf("%v/%v", appid, channelId)
		ol.Tf(nil, "Request channelId=%v, user=%v, appid=%v", channelId, user, appid)

		var auth *ChannelAuth
		func() {
			lock.Lock()
			defer lock.Unlock()

			var ok bool
			if auth, ok = channels[channelUrl]; ok {
				return
			}

			var err error
			if auth, err = CreateChannel(appid, channelId, regionId, accessKeyId, accessKeySecret); err != nil {
				oh.WriteError(nil, w, r, err)
				return
			}

			channels[channelUrl] = auth
			ol.Tf(nil, "Create channelId=%v, nonce=%v, timestamp=%v, channelKey=%v",
				channelId, auth.Nonce, auth.Timestamp, auth.ChannelKey)
		}()
		if auth == nil {
			return
		}

		userId, session := BuildRandom(32), BuildRandom(32)
		token, err := BuildToken(channelId, auth.ChannelKey, appid, userId, session, auth.Nonce, auth.Timestamp)
		if err != nil {
			oh.WriteError(nil, w, r, err)
			return
		}

		username := fmt.Sprintf("%s?appid=%s&session=%s&channel=%s&nonce=%s&timestamp=%d",
			userId, appid, session, channelId, auth.Nonce, auth.Timestamp)

		type TURN struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		type Response struct {
			AppId     string   `json:"appid"`
			UserId    string   `json:"userid"`
			GSLB      []string `json:"gslb"`
			Session   string   `json:"session"`
			Token     string   `json:"token"`
			Nonce     string   `json:"nonce"`
			Timestamp int64    `json:"timestamp"`
			TURN      *TURN    `json:"turn"`
		}
		oh.WriteData(nil, w, r, &Response{
			appid, userId, []string{gslb}, session, token, auth.Nonce, auth.Timestamp,
			&TURN{username, token},
		})
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%v", listen), nil); err != nil {
		panic(err)
	}
}
