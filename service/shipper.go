package service

import (
	"fmt"
	"github.com/TrollEyeSecurity/ccsyslogingester/config"
	"github.com/TrollEyeSecurity/ccsyslogingester/utilities/redis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"log"
	"time"
)

func ShipperService() {
	timeout := 5 * time.Second
	rdb, getRedisClientErr := redis.GetRedisClient()
	if getRedisClientErr != nil {
		err := fmt.Errorf("get-redis-client error - %v", *getRedisClientErr)
		fmt.Println(err)
		sentry.CaptureException(err)
		log.Println(err)
		return
	}
	var sequenceToken string
	var logStreamName string
	appConfiguration, appConfigurationErr := config.LoadConfiguration("/etc/ccsyslog/config.json")
	if appConfigurationErr != nil {
		err := fmt.Errorf("app-configuration error %v", appConfigurationErr)
		log.Println(err)
		return
	}
	cloudWatchGroup := appConfiguration.CloudWatchGroupName
	awsRegion := appConfiguration.AwsRegion
	awsAccessKeyId := appConfiguration.AwsAccessKeyId
	awsAccessKeySecret := appConfiguration.AwsSecretAccessKey
	awsCredentials := credentials.NewStaticCredentials(awsAccessKeyId, awsAccessKeySecret, "")
	sess := session.Must(session.NewSession())
	awsConfig := aws.Config{Region: &awsRegion, Credentials: awsCredentials}
	cwl := cloudwatchlogs.New(sess, &awsConfig)
	for {
		result, bRPopErr := rdb.BRPop(timeout, redis.SIEMTasksQueue).Result()
		if bRPopErr != nil {
			if bRPopErr.Error() == "redis: nil" {
				time.Sleep(5 * time.Second)
				sequenceToken = ""
				continue
			}
			err := fmt.Errorf("redis brpop error - %v", bRPopErr)
			fmt.Println(err)
			sentry.CaptureException(err)
			log.Println(err)
			break
		}

		ts := aws.Int64(time.Now().UnixNano() / int64(time.Millisecond))
		cwLog := cloudwatchlogs.InputLogEvent{Message: &result[1], Timestamp: ts}
		var logQueue []*cloudwatchlogs.InputLogEvent
		logQueue = append(logQueue, &cwLog)
		input := cloudwatchlogs.PutLogEventsInput{
			LogEvents:    logQueue,
			LogGroupName: &cloudWatchGroup,
		}
		if sequenceToken == "" {
			name := uuid.New().String()
			_, createLogStreamErr := cwl.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
				LogGroupName:  &cloudWatchGroup,
				LogStreamName: &name,
			})
			if createLogStreamErr != nil {
				err := fmt.Errorf("create-log-stream error - %v", createLogStreamErr)
				fmt.Println(err)
				sentry.CaptureException(err)
				log.Println(err)
				break
			}
			logStreamName = name
		} else {
			input = *input.SetSequenceToken(sequenceToken)
		}
		input = *input.SetLogStreamName(logStreamName)
		resp, putLogEventsErr := cwl.PutLogEvents(&input)
		if putLogEventsErr != nil {
			err := fmt.Errorf("put-log-events error - %v", putLogEventsErr)
			fmt.Println(err)
			sentry.CaptureException(err)
			log.Println(err)
			break
		}
		if resp != nil {
			sequenceToken = *resp.NextSequenceToken
		}

	}
}
