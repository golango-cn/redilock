## redilock


Distributed locks and a guard dog

## install
go get github.com/ningzi1986/redilock

## Usage

```go

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

```

See test cases for more usage