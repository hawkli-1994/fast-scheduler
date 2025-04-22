package fastscheduler

import (
	"context"
	"sync"
	"sync/atomic"
)

// TaskResult 表示任务执行结果
type TaskResult struct {
	HTTPCode     int
	BusinessCode int
	Err          error
	Data         interface{}
}

// Task 表示要执行的任务
type Task struct {
	// ID 用于标识任务
	ID string

	// Execute 是任务执行函数
	Execute func(ctx context.Context) (TaskResult, error)

	// ResultChan 用于接收结果(可选)
	ResultChan chan<- TaskResult

	// 内部使用的字段
	group      *taskGroup
	cancelFunc context.CancelFunc
}

// Batch 表示一批任务
type Batch struct {
	Tasks []*Task
	group *taskGroup
}

// Scheduler 任务调度器
type Scheduler struct {
	taskQueue  chan *Task
	workerPool chan struct{}
	wg         sync.WaitGroup
	stopChan   chan struct{}
}

// taskGroup 用于管理一批任务
type taskGroup struct {
	ctx     context.Context
	cancel  context.CancelFunc
	success *atomic.Bool
	wg      sync.WaitGroup
}

// NewScheduler 创建一个新的调度器
// poolSize: goroutine池大小
// queueSize: 任务队列大小
func NewScheduler(poolSize, queueSize int) *Scheduler {
	s := &Scheduler{
		taskQueue:  make(chan *Task, queueSize),
		workerPool: make(chan struct{}, poolSize),
		stopChan:   make(chan struct{}),
	}

	// 启动调度器
	s.start()
	return s
}

// start 启动调度器
func (s *Scheduler) start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case task := <-s.taskQueue:
				// 获取worker
				s.workerPool <- struct{}{}
				s.wg.Add(1)
				go s.executeTask(task)
			case <-s.stopChan:
				return
			}
		}
	}()
}

// executeTask 执行单个任务
func (s *Scheduler) executeTask(task *Task) {
	defer func() {
		<-s.workerPool // 释放worker
		s.wg.Done()
		task.group.wg.Done()
	}()

	var result TaskResult

	// 执行任务
	var err error
	result, err = task.Execute(task.group.ctx)
	if err != nil {
		result.Err = err
		// 确保在错误情况下也设置适当的状态码
		if result.HTTPCode == 0 {
			result.HTTPCode = 500
		}
		if result.BusinessCode == 0 {
			result.BusinessCode = 1
		}
	}

	// 检查是否成功(HTTP 200且业务码0)
	if result.HTTPCode == 200 && result.BusinessCode == 0 {
		if task.group.success.CompareAndSwap(false, true) {
			// 第一个成功的任务，取消同组其他任务
			task.group.cancel()
		}
	}

	// 发送结果(如果有接收channel)
	if task.ResultChan != nil {
		task.ResultChan <- result
	}
}

// SubmitBatch 提交一批任务
func (s *Scheduler) SubmitBatch(tasks []*Task) *Batch {
	ctx, cancel := context.WithCancel(context.Background())
	group := &taskGroup{
		ctx:     ctx,
		cancel:  cancel,
		success: &atomic.Bool{},
	}

	batch := &Batch{
		Tasks: tasks,
		group: group,
	}

	group.wg.Add(len(tasks))
	for _, task := range tasks {
		task.group = group
		task.cancelFunc = cancel
		s.taskQueue <- task
	}

	return batch
}

// Wait 等待所有任务完成
func (s *Scheduler) Wait() {
	s.wg.Wait()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
	close(s.taskQueue)
	close(s.workerPool)
}

// Wait 等待批次中的所有任务完成
func (b *Batch) Wait() {
	b.group.wg.Wait()
}

// IsSuccess 返回批次中是否有任务成功
func (b *Batch) IsSuccess() bool {
	return b.group.success.Load()
}
