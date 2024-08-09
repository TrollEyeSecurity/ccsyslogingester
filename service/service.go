package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TrollEyeSecurity/ccsyslogingester/config"
	"github.com/TrollEyeSecurity/ccsyslogingester/syslogs"
	"github.com/TrollEyeSecurity/ccsyslogingester/utilities/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/getsentry/sentry-go"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func IngestService() {
	appConfiguration := config.LoadConfiguration("/etc/ccsyslog/config.json")
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
		go ParseMsg(&msg, &addr)
	}
}

func ParseMsg(msg *[]byte, remoteAddr *string) {
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
		panic(err)
	}

	cefMessage := syslogs.CefMessage{
		CefVersion:     cefVersionInt,
		ProductVendor:  productVendor,
		Product:        product,
		ProductVersion: productVersion,
		EventClass:     eventClass,
		EventName:      eventName,
		EventSeverity:  eventSeverity,
		SyslogMsg:      syslogMsg,
		JsonMsg:        string(*jsonMsg),
	}
	rdb, getRedisClientErr := redis.GetRedisClient()
	if getRedisClientErr != nil {
		err := fmt.Errorf("get-redis-client error - %v", *getRedisClientErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	a, jsonMarshalErr := json.Marshal(cefMessage)
	if jsonMarshalErr != nil {
		err := fmt.Errorf("json-marshal error %v", jsonMarshalErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	rdb.LPush(redis.SIEMTasksQueue, a)
}

func ShipperService() {
	var wg sync.WaitGroup
	timeout := 5 * time.Second
	rdb, getRedisClientErr := redis.GetRedisClient()
	if getRedisClientErr != nil {
		err := fmt.Errorf("get-redis-client error - %v", *getRedisClientErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	for {
		var num int64
		if rdb.LLen(redis.SIEMTasksQueue).Val() > 20 {
			num = int64(100)
		} else {
			num = rdb.LLen(redis.SIEMTasksQueue).Val()
		}
		for i := int64(0); i < num; i++ {
			result, bRPopErr := rdb.BRPop(timeout, redis.SIEMTasksQueue).Result()
			if bRPopErr != nil {
				if bRPopErr.Error() == "redis: nil" {
					break
				}
				err := fmt.Errorf("redis brpop error - %v", bRPopErr)
				fmt.Println(err)
				sentry.CaptureException(err)
				log.Println(err)
				break
			}
			wg.Add(1)
			go ExecuteTask(&result[1], &wg)
		}
		wg.Wait()
	}
}

func ExecuteTask(result *string, wg *sync.WaitGroup) {
	defer wg.Done()
	appConfiguration := config.LoadConfiguration("/etc/ccsyslog/config.json")
	ts := time.Now().Unix()
	var cefMessage syslogs.CefMessage
	unmarshalErr := json.Unmarshal([]byte(*result), &cefMessage)
	if unmarshalErr != nil {
		err := fmt.Errorf("exectue task unmarshalErr error - %v", unmarshalErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	awsRegion := appConfiguration.AwsRegion
	bucketName := appConfiguration.BucketName
	fileName := fmt.Sprintf("%s-%s-%s-%d", cefMessage.EventName, cefMessage.EventSeverity, cefMessage.EventClass, ts)
	awsAccessKeyId := appConfiguration.AwsAccessKeyId
	awsAccessKeySecret := appConfiguration.AwsSecretAccessKey
	awsCredentials := credentials.NewStaticCredentials(awsAccessKeyId, awsAccessKeySecret, "")
	sess := session.Must(session.NewSession())
	awsConfig := aws.Config{Region: &awsRegion, Credentials: awsCredentials}
	svc := s3.New(sess, &awsConfig)
	_, putObjectErr := svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
		Body:   bytes.NewReader([]byte(*result)),
	})
	if putObjectErr != nil {
		err := fmt.Errorf("s3-put error - %v", putObjectErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
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
