package src

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"time"
)

// Redilock
type Redilock struct {
	Addr     string        // Redis address
	Db       int           // Redis database
	Password string        // Redis password
	client   *redis.Client // Redis client
	Key      string        // Lock key

	maxRetryCount     int          // Maximum number of retries. If set to -1, retry until the lock is acquired
	retryMilliseconds int          // Retry the number of milliseconds
	currentRetryCount int          // Current number of retries
	ticker            *time.Ticker // The timer for watch dog
}

// New Redilock
func NewRedilock(key, addr, password string, db int) (*Redilock, error) {

	redilock := &Redilock{
		Key:               key,
		Addr:              addr,
		Db:                db,
		Password:          password,
		maxRetryCount:     -1,
		retryMilliseconds: 1000,
	}
	redilock.client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	_, err := redilock.client.Ping().Result()
	if err != nil {
		return nil, err
	}
	return redilock, nil
}

// Set the maximum number of retries
func (r *Redilock) SetMaxRetryCount(max int) error {
	if max < -1 {
		return errors.New("The maximum number of retries cannot be less than -1")
	}
	r.maxRetryCount = max
	return nil
}

// Redis lua script
var script = `if (redis.call('exists', KEYS[1]) == 0) then
				 redis.call('hset', KEYS[1], ARGV[2], 1);
				 redis.call('pexpire', KEYS[1], ARGV[1]);
				 return -100;
				end;
				if (redis.call('hexists', KEYS[1], ARGV[2]) == 1) then
					 redis.call('hincrby', KEYS[1], ARGV[2], 1);
					 return -101;
				end;
				return redis.call('pttl', KEYS[1]);`
var newScript = redis.NewScript(script)

// Lock
func (r *Redilock) Lock(ttl time.Duration, clientNo string) error {
	return r.lock(ttl, clientNo)
}

// lock
func (r *Redilock) lock(ttl time.Duration, clientNo string) error {

	result, err := newScript.Run(r.client, []string{r.Key}, ttl.Milliseconds(), clientNo).Result()
	if err != nil {
		return err
	}

	ret := result.(int64)

	// Locking success
	if ret == -100 {
		//　Give Key a dog that is about to expire
		go r.watchDog(ttl, clientNo)
		return nil
	}

	if ret == -101 {
		log.Println(clientNo, "The same lock already exists for this client")
		return nil
	}

	if r.maxRetryCount > 0 && r.currentRetryCount >= r.maxRetryCount {
		return fmt.Errorf(clientNo, "Try to lock retry too many times, lock failed")
	}

	r.currentRetryCount++
	time.Sleep(time.Duration(r.retryMilliseconds) * time.Millisecond)

	return r.lock(ttl, clientNo)

}

// Unlock
func (r *Redilock) UnLock() error {

	r.reset()
	_, err := r.client.Del(r.Key).Result()
	if err != nil {
		return err
	}
	return nil

}

// Reset Config
func (r *Redilock) reset() {

	r.currentRetryCount = 0
	if r.ticker != nil {
		r.ticker.Stop()
	}

}

// A very loyal dog
func (r *Redilock) watchDog(ttl time.Duration, clientNo string) {

	if r.ticker != nil {
		r.ticker.Stop()
	}

	f := float64(ttl.Milliseconds()) - float64(ttl.Milliseconds())/3
	r.ticker = time.NewTicker(time.Duration(f) * time.Millisecond)

	log.Println(clientNo, fmt.Sprintf("Watchdog start, renew after %f milliseconds", f))

	for {
		<-r.ticker.C
		log.Println(clientNo, "Watchdog renew key", r.client, r.Key, ttl.Milliseconds())
		r.client.Expire(r.Key, ttl)
	}

}
