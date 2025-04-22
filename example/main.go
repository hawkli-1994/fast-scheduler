package main

import (
	"context"
	"fmt"
	"time"

	fastscheduler "github.com/hawkli-1994/fast-scheduler"
)

func main() {
	// 创建调度器，10个worker，任务队列大小100
	scheduler := fastscheduler.NewScheduler(10, 100)
	defer scheduler.Stop()

	// 模拟HTTP请求函数
	mockHTTPRequest := func(id string, sleepTime time.Duration, success bool) (fastscheduler.TaskResult, error) {
		time.Sleep(sleepTime)
		if success {
			return fastscheduler.TaskResult{
				HTTPCode:     200,
				BusinessCode: 0,
				Data:         "success from " + id,
			}, nil
		}
		return fastscheduler.TaskResult{
			HTTPCode:     200,
			BusinessCode: 1,
			Data:         "fail from " + id,
		}, nil
	}

	// 创建一批任务
	var tasks []*fastscheduler.Task
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("task-%d", i)
		success := i == 3 // 只有task-3会成功
		task := &fastscheduler.Task{
			ID: id,
			Execute: func(ctx context.Context) (fastscheduler.TaskResult, error) {
				return mockHTTPRequest(id, time.Duration(i+1)*time.Second, success)
			},
		}
		tasks = append(tasks, task)
	}

	// 提交任务批次
	batch := scheduler.SubmitBatch(tasks)

	// 等待批次完成
	batch.Wait()

	// 检查是否有任务成功
	if batch.IsSuccess() {
		fmt.Println("有一项任务成功完成，其他任务已取消")
	} else {
		fmt.Println("所有任务都失败了")
	}

	// 等待所有调度器任务完成
	scheduler.Wait()
}
