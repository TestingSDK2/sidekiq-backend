package cache

import (
	"fmt"

	"github.com/go-redis/redis/v7"
)

const oneHour int = 3600

const oneDay int = 3600 * 24

// Expire18HR - 18 hours
const Expire18HR int = oneHour * 18

// Expire24HR - 24 hours
const Expire24HR int = oneHour * 24

// Expire1WK - 7 days
const Expire1WK int = oneDay * 7

// Expire30D - 30 days
const Expire30D int = oneDay * 30

// Cache redis cache
type Cache struct {
	Client *redis.Client
}

// New create new cache
func New(config *Config) *Cache {
	cache := &Cache{}
	cache.Client = redis.NewClient(&redis.Options{
		Addr:     getCacheURL(config),
		Password: config.Password,
	})
	return cache

}

// Close cache
func (c *Cache) Close() error {
	return c.Client.Close()
}

func getCacheURL(config *Config) string {
	return fmt.Sprintf("%s:%s", config.Host, config.Port)
}

// GetValue - get value string by key
func (c *Cache) GetValue(key string) (string, error) {
	val, err := c.Client.Get(key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetValue - get value string by key
func (c *Cache) SetValue(key string, val string) error {
	return c.Client.Set(key, val, 0).Err()
}

// ExpireKey - set a redis key to expire
func (c *Cache) ExpireKey(key string, ttl int) {
	c.Client.Do("EXPIRE", key, fmt.Sprintf("%v", ttl))
}

func (c *Cache) Publish(key string, val string) error {
	return c.Client.Publish(key, val).Err()
}

func (c *Cache) Subscribe(key string) (<-chan *redis.Message, error) {
	sub := c.Client.Subscribe(key)
	_, err := sub.Receive()
	if err != nil {
		return nil, err
	}
	return sub.Channel(), nil
}

// DeleteValue - delete value string by key
func (c *Cache) DeleteValue(key string) error {
	return c.Client.Del(key).Err()
}
