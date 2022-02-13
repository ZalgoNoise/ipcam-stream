package ipcam

import (
	"math"
	"math/rand"
	"time"
)

func ExpBackoff(max time.Duration, call func() error) (n int, err error) {

	var i float64
	var t time.Duration

	for i = float64(1); t <= max; i++ {
		err = call()

		if err != nil {
			t = increment(i)
			time.Sleep(t)
		} else {
			return int(i), nil
		}
	}
	return int(i), err
}

func increment(n float64) time.Duration {
	return time.Millisecond * time.Duration(
		int64(math.Pow(2, n))+
			rand.New(rand.NewSource(time.Now().UnixNano())).Int63n(1000))
}
