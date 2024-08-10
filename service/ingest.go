package service

import (
	"encoding/json"
	"fmt"
	"github.com/TrollEyeSecurity/ccsyslogingester/config"
	"github.com/TrollEyeSecurity/ccsyslogingester/syslogs"
	"github.com/TrollEyeSecurity/ccsyslogingester/utilities/redis"
	"github.com/getsentry/sentry-go"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func IngestService() {
	appConfiguration, appConfigurationErr := config.LoadConfiguration("/etc/ccsyslog/config.json")
	if appConfigurationErr != nil {
		err := fmt.Errorf("app-configuration error %v", appConfigurationErr)
		log.Println(err)
		return
	}
	l, ListenTcpErr := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: appConfiguration.ListenPort})
	if ListenTcpErr != nil {
		err := fmt.Errorf("listen-tcp error %v", ListenTcpErr)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	defer l.Close()

	for {
		c, lErr := l.Accept()
		if lErr != nil {
			err := fmt.Errorf("listen-accept error %v", lErr)
			sentry.CaptureException(err)
			log.Println(err)
			return
		}
		go handleClient(c)
	}
}

func handleClient(c net.Conn) {
	defer c.Close()
	for {
		buffer := make([]byte, 1024)
		n, readErr := c.Read(buffer)
		if readErr != nil {
			err := fmt.Errorf("buffer-read error %v", readErr)
			sentry.CaptureException(err)
			log.Println(err)
			return
		}
		msg := buffer[:n]
		addr := c.RemoteAddr().String()
		go HandleMsg(&msg, &addr)
	}
}

func HandleMsg(msg *[]byte, remoteAddr *string) {
	splitMsg := strings.Split(string(*msg), "|")
	if len(splitMsg) < 8 {
		return
	}
	cef := strings.Split(strings.ToLower(splitMsg[0]), "cef:")
	if len(cef) < 2 {
		return
	}
	cefVersion := cef[1]
	productVendor := splitMsg[1]
	product := splitMsg[2]
	productVersion := splitMsg[3]
	eventClass := splitMsg[4]
	eventName := splitMsg[5]
	eventSeverity := splitMsg[6]
	syslogMsg := splitMsg[7]
	syslogMsgSplit := strings.Split(syslogMsg, " ")
	jsonMsg, makeJsonErr := MakeJson(&syslogMsgSplit)
	if makeJsonErr != nil {
		err := fmt.Errorf("make-json error %v", makeJsonErr)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}

	cefVersionInt, atoiErr := strconv.Atoi(cefVersion)
	if atoiErr != nil {
		err := fmt.Errorf("string-to-int error %v", atoiErr)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	cefMessage := syslogs.CefMessage{
		Time:           time.Now(),
		RemoteAddr:     *remoteAddr,
		CefVersion:     cefVersionInt,
		ProductVendor:  productVendor,
		Product:        product,
		ProductVersion: productVersion,
		EventClass:     eventClass,
		EventName:      eventName,
		EventSeverity:  eventSeverity,
		SyslogMsg:      syslogMsg,
		JsonMsg:        *jsonMsg,
	}

	cefMessageByte, jsonMarshalErr := json.Marshal(cefMessage)
	if jsonMarshalErr != nil {
		err := fmt.Errorf("json-marshal error %v", jsonMarshalErr)
		sentry.CaptureException(err)
		log.Println(err)
	}

	rdb, getRedisClientErr := redis.GetRedisClient()
	if getRedisClientErr != nil {
		err := fmt.Errorf("get-redis-client error - %v", *getRedisClientErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}

	rdb.LPush(redis.SIEMTasksQueue, cefMessageByte)

}

func MakeJson(array *[]string) (*[]byte, error) {
	jsonA := make(map[string]string)
	for _, str := range *array {
		s := strings.Split(str, "=")
		if len(s) != 2 {
			continue
		}
		jsonA[s[0]] = s[1]
	}
	jsonB, e := json.Marshal(jsonA)
	if e != nil {
		return nil, e
	}
	return &jsonB, nil
}
