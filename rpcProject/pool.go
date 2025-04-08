package rpcProject

import (
	"sync"
)

type Task func()

type Pool struct {
	TaskQueue chan Task
	wg        sync.WaitGroup
}

func NewPool(size int) *Pool {
	return &Pool{
		TaskQueue: make(chan Task, size),
	}
}

func (p *Pool) Run() {
	for i := 0; i < cap(p.TaskQueue); i++ {
		go func() {
			defer p.wg.Done()
			for task := range p.TaskQueue {
				task()
			}
		}()
		p.wg.Add(1)
	}
}

func (p *Pool) AddTask(task Task) {
	p.TaskQueue <- task
}

func (p *Pool) Stop() {
	close(p.TaskQueue)
	p.wg.Wait()
}
