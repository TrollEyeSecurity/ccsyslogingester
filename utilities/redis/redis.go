package redis

import (
	"github.com/go-redis/redis"
	"os"
)

func GetRedisClient() (*redis.Client, *error) {
	redisUrl := os.Getenv("BROKER_URL")
	opts, parseUrlErr := redis.ParseURL(redisUrl)
	if parseUrlErr != nil {
		return nil, &parseUrlErr
	}
	client := redis.NewClient(opts)
	_, pingErr := client.Ping().Result()
	if pingErr != nil {
		return nil, &pingErr
	}
	return client, nil
}
