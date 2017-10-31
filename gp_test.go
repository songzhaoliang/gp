package gp

import (
	"io/ioutil"
	"log"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func TestNewWorker(t *testing.T) {
	workerPool := make(chan *Worker)
	worker := NewWorker(workerPool)
	worker.Start()
	assert.NotNil(t, worker)

	worker = <-workerPool
	assert.NotNil(t, worker, "Worker should register itself to the pool")

	called := false
	done := make(chan bool)
	worker.jobChannel <- func() {
		called = true
		done <- true
	}
	<-done
	assert.Equal(t, true, called)
}

func TestNewPool(t *testing.T) {
	pool := NewPool(1000, 10000)
	defer pool.Release()

	iterations := 100000
	pool.WaitCount(iterations)
	var counter uint64
	for i := 0; i < iterations; i++ {
		arg := uint64(1)
		pool.Go(func() {
			defer pool.JobDone()

			atomic.AddUint64(&counter, arg)
			assert.Equal(t, uint64(1), arg)
		})
	}
	pool.WaitAll()

	assert.Equal(t, uint64(iterations), atomic.LoadUint64(&counter))
}

func TestPool_Release(t *testing.T) {
	num := runtime.NumGoroutine()
	defer func() {
		assert.Equal(t, num, runtime.NumGoroutine())
	}()

	pool := NewPool(5, 10)
	defer pool.Release()
	iterations := 1000
	pool.WaitCount(iterations)
	for i := 0; i < iterations; i++ {
		pool.Go(func() {
			defer pool.JobDone()
		})
	}
	pool.WaitAll()
}

func BenchmarkPool(b *testing.B) {
	pool := NewPool(1, 10)
	defer pool.Release()

	log.SetOutput(ioutil.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Go(func() {
			log.Printf("I am worker which number is %d\n", i)
		})
	}
}
