package fastscheduler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestScheduler_BasicFunctionality(t *testing.T) {
	scheduler := NewScheduler(5, 10)
	defer scheduler.Stop()

	// 创建一个会成功的任务
	successTask := &Task{
		ID: "success-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			return TaskResult{
				HTTPCode:     200,
				BusinessCode: 0,
				Data:         "success",
			}, nil
		},
	}

	// 创建一个会失败的任务
	failTask := &Task{
		ID: "fail-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			return TaskResult{
				HTTPCode:     200,
				BusinessCode: 1,
				Data:         "fail",
			}, nil
		},
	}

	// 提交批次
	batch := scheduler.SubmitBatch([]*Task{successTask, failTask})
	batch.Wait()

	if !batch.IsSuccess() {
		t.Error("Expected batch to have a successful task")
	}
}

func TestScheduler_EarlyCancellation(t *testing.T) {
	scheduler := NewScheduler(5, 10)
	defer scheduler.Stop()

	// 使用通道来检测任务是否被执行
	executed := make(chan string, 2)

	// 快速成功的任务
	fastTask := &Task{
		ID: "fast-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			executed <- "fast"
			return TaskResult{
				HTTPCode:     200,
				BusinessCode: 0,
				Data:         "fast success",
			}, nil
		},
	}

	// 慢速任务，应该被取消
	slowTask := &Task{
		ID: "slow-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			select {
			case <-time.After(500 * time.Millisecond):
				executed <- "slow"
				return TaskResult{
					HTTPCode:     200,
					BusinessCode: 0,
					Data:         "slow success",
				}, nil
			case <-ctx.Done():
				return TaskResult{}, ctx.Err()
			}
		},
	}

	// 提交批次
	batch := scheduler.SubmitBatch([]*Task{fastTask, slowTask})
	batch.Wait()

	// 检查执行顺序和结果
	first := <-executed
	if first != "fast" {
		t.Errorf("Expected fast task to execute first, got %s", first)
	}

	select {
	case second := <-executed:
		t.Errorf("Slow task should have been cancelled, but got %s", second)
	case <-time.After(600 * time.Millisecond):
		// 正确情况，慢任务被取消了
	}

	if !batch.IsSuccess() {
		t.Error("Expected batch to have a successful task")
	}
}

func TestScheduler_AllTasksFail(t *testing.T) {
	scheduler := NewScheduler(5, 10)
	defer scheduler.Stop()

	// 创建两个失败任务
	task1 := &Task{
		ID: "fail-1",
		Execute: func(ctx context.Context) (TaskResult, error) {
			return TaskResult{
				HTTPCode:     200,
				BusinessCode: 1,
				Data:         "fail 1",
			}, nil
		},
	}

	task2 := &Task{
		ID: "fail-2",
		Execute: func(ctx context.Context) (TaskResult, error) {
			return TaskResult{
				HTTPCode:     500,
				BusinessCode: 0,
				Data:         "fail 2",
			}, nil
		},
	}

	// 提交批次
	batch := scheduler.SubmitBatch([]*Task{task1, task2})
	batch.Wait()

	if batch.IsSuccess() {
		t.Error("Expected batch to have no successful tasks")
	}
}

func TestScheduler_ErrorHandling(t *testing.T) {
	scheduler := NewScheduler(5, 10)
	defer scheduler.Stop()

	// 创建一个双向通道用于接收结果
	resultChan := make(chan TaskResult, 1)

	// 创建一个返回错误的函数
	errTask := &Task{
		ID: "error-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			return TaskResult{}, errors.New("simulated error")
		},
		ResultChan: resultChan,
	}

	// 创建一个成功的任务
	successTask := &Task{
		ID: "success-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			return TaskResult{
				HTTPCode:     200,
				BusinessCode: 0,
				Data:         "success",
			}, nil
		},
	}

	// 提交批次
	batch := scheduler.SubmitBatch([]*Task{errTask, successTask})
	batch.Wait()

	// 检查错误任务的结果，添加超时机制
	select {
	case res := <-resultChan:
		if res.Err == nil || res.Err.Error() != "simulated error" {
			t.Errorf("Expected 'simulated error', got %v", res.Err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive error result, but timed out")
	}

	if !batch.IsSuccess() {
		t.Error("Expected batch to have a successful task")
	}
}

func TestScheduler_ConcurrentBatches(t *testing.T) {
	scheduler := NewScheduler(5, 20)
	defer scheduler.Stop()

	// 创建两个独立的批次
	batch1Success := make(chan bool, 1)
	batch2Success := make(chan bool, 1)

	// 批次1 - 包含一个成功任务
	batch1Tasks := []*Task{
		{
			ID: "batch1-task",
			Execute: func(ctx context.Context) (TaskResult, error) {
				return TaskResult{
					HTTPCode:     200,
					BusinessCode: 0,
					Data:         "batch1 success",
				}, nil
			},
		},
	}

	// 批次2 - 包含一个慢速成功任务
	batch2Tasks := []*Task{
		{
			ID: "batch2-task",
			Execute: func(ctx context.Context) (TaskResult, error) {
				time.Sleep(200 * time.Millisecond)
				return TaskResult{
					HTTPCode:     200,
					BusinessCode: 0,
					Data:         "batch2 success",
				}, nil
			},
		},
	}

	// 并发提交两个批次
	go func() {
		batch := scheduler.SubmitBatch(batch1Tasks)
		batch.Wait()
		batch1Success <- batch.IsSuccess()
	}()

	go func() {
		batch := scheduler.SubmitBatch(batch2Tasks)
		batch.Wait()
		batch2Success <- batch.IsSuccess()
	}()

	// 等待结果
	b1Success := <-batch1Success
	b2Success := <-batch2Success

	if !b1Success {
		t.Error("Expected batch1 to succeed")
	}
	if !b2Success {
		t.Error("Expected batch2 to succeed")
	}
}

func TestScheduler_StopBehavior(t *testing.T) {
	scheduler := NewScheduler(1, 1) // 小池更容易测试停止行为

	// 添加一个会阻塞的任务来测试Stop
	blockingTask := &Task{
		ID: "blocking-task",
		Execute: func(ctx context.Context) (TaskResult, error) {
			time.Sleep(2 * time.Second)
			return TaskResult{
				HTTPCode:     200,
				BusinessCode: 0,
				Data:         "blocking success",
			}, nil
		},
	}

	// 提交任务但不等待
	_ = scheduler.SubmitBatch([]*Task{blockingTask})

	// 立即停止
	stopDone := make(chan struct{})
	go func() {
		scheduler.Stop()
		close(stopDone)
	}()

	// 验证Stop不会永久阻塞
	select {
	case <-stopDone:
		// 正常情况
	case <-time.After(3 * time.Second):
		t.Error("Stop() took too long to complete")
	}
}
