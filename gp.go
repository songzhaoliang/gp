package gp

import (
	"sync"
)

type Job func()

type Worker struct {
	workerPool chan *Worker
	jobChannel chan Job
	stop       chan struct{}
}

func NewWorker(workerPool chan *Worker) *Worker {
	return &Worker{
		workerPool: workerPool,
		jobChannel: make(chan Job),
		stop:       make(chan struct{}),
	}
}

func (w *Worker) Start() {
	go func() {
		for {
			w.workerPool <- w

			select {
			case job := <-w.jobChannel:
				job()
			case <-w.stop:
				w.stop <- struct{}{}
				return
			}
		}
	}()
}

type Dispatcher struct {
	workerPool chan *Worker
	jobQueue   chan Job
	stop       chan struct{}
}

func NewDispatcher(workerPool chan *Worker, jobQueue chan Job) *Dispatcher {
	d := &Dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		stop:       make(chan struct{}),
	}

	for i := 0; i < cap(d.workerPool); i++ {
		worker := NewWorker(d.workerPool)
		worker.Start()
	}

	d.Dispatch()
	return d
}

func (d *Dispatcher) Dispatch() {
	go func() {
		for {
			select {
			case job := <-d.jobQueue:
				worker := <-d.workerPool
				worker.jobChannel <- job
			case <-d.stop:
				for i := 0; i < cap(d.workerPool); i++ {
					worker := <-d.workerPool
					worker.stop <- struct{}{}
					<-worker.stop
				}

				d.stop <- struct{}{}
				return
			}
		}
	}()
}

type Pool struct {
	jobQueue   chan Job
	dispatcher *Dispatcher
	wg         sync.WaitGroup
}

func NewPool(numWorkers, jobQueueLen int) *Pool {
	if numWorkers < 1 {
		numWorkers = 1
	}
	if jobQueueLen < 1 {
		jobQueueLen = 1
	}

	workerPool := make(chan *Worker, numWorkers)
	jobQueue := make(chan Job, jobQueueLen)

	return &Pool{
		jobQueue:   jobQueue,
		dispatcher: NewDispatcher(workerPool, jobQueue),
	}
}

func (p *Pool) Go(job Job) {
	if job != nil {
		p.jobQueue <- job
	}
}

func (p *Pool) Release() {
	p.dispatcher.stop <- struct{}{}
	<-p.dispatcher.stop
}

func (p *Pool) WaitCount(count int) {
	p.wg.Add(count)
}

func (p *Pool) JobDone() {
	p.wg.Done()
}

func (p *Pool) WaitAll() {
	p.wg.Wait()
}
