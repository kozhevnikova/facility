package main

import (
	"github.com/go-redis/redis"
)

func connectToRedis(config Config) (*redis.Client, error) {
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.Redis.Address,
		DB:   config.Redis.Database,
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		return nil, err
	}
	return redisClient, nil
}
