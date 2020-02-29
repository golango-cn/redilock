package redilock

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func Test01(t *testing.T) {

	redilock, err := NewRedilock("user:1", "192.168.1.200:6379", "", 0)
	if err != nil {
		panic(err)
	}

	err = redilock.Lock(time.Duration(30000)*time.Millisecond, "client001")
	if err != nil {
		panic(err)
	}
	defer redilock.UnLock()

	//Simulate execution for 50 seconds
	time.Sleep(50 * time.Second)

}

func Test02(t *testing.T) {

	var wg sync.WaitGroup

	redilock, err := NewRedilock("user:1", "192.168.1.200:6379", "", 0)
	if err != nil {
		panic(err)
	}

	clients := []struct {
		clientNo string
		ttl      time.Duration
	}{
		{"clientNo001", time.Second * 30},
		{"clientNo002", time.Second * 10},
		{"clientNo003", time.Second * 20},
		{"clientNo004", time.Second * 15},
		{"clientNo005", time.Second * 11},
	}

	wg.Add(len(clients))

	for _, v := range clients {

		go func(ttl time.Duration, clientNo string) {

			defer wg.Done()
			err2 := redilock.Lock(ttl, clientNo)
			if err2 != nil {
				fmt.Errorf("Locking failure %s：%s\n", clientNo, err2.Error())
				return
			}
			defer redilock.UnLock()

			doSomething(clientNo)

		}(v.ttl, v.clientNo)
	}

	wg.Wait()

}

func doSomething(clientNo string) {

	intn := rand.Intn(50000)

	fmt.Printf("%s Client startup，execute for %d millisecond\n", clientNo, intn)
	time.Sleep(time.Duration(intn) * time.Millisecond)

	fmt.Println("Client execution is complete", clientNo)
}
