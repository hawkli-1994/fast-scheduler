# Fast Scheduler

Fast Scheduler 是一个高性能的 Go 语言任务调度框架，专注于快速执行和高效资源利用。它提供了简单而强大的 API 来管理并发任务的执行。

## 特性

- 🚀 高性能：使用 goroutine 池和任务队列实现高效的任务调度
- 🔄 并发控制：支持自定义 worker 池大小和任务队列容量
- 🎯 任务批处理：支持批量提交任务，自动管理任务生命周期
- ⚡ 快速失败：当批次中有一个任务成功时，自动取消其他任务
- 🔍 结果追踪：支持通过 channel 接收任务执行结果
- 🛡️ 错误处理：完善的错误处理机制，支持 HTTP 状态码和业务状态码
- ⏱️ 超时控制：支持通过 context 控制任务超时

## 安装

```bash
go get github.com/hawkli-1994/fast-scheduler
```

## 快速开始

```go
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

    // 创建任务
    task := &fastscheduler.Task{
        ID: "my-task",
        Execute: func(ctx context.Context) (fastscheduler.TaskResult, error) {
            // 执行任务逻辑
            return fastscheduler.TaskResult{
                HTTPCode:     200,
                BusinessCode: 0,
                Data:        "success",
            }, nil
        },
    }

    // 提交任务批次
    batch := scheduler.SubmitBatch([]*fastscheduler.Task{task})
    
    // 等待批次完成
    batch.Wait()

    // 检查结果
    if batch.IsSuccess() {
        fmt.Println("任务执行成功")
    }
}
```

## 高级用法

### 批量任务处理

```go
// 创建多个任务
tasks := []*fastscheduler.Task{
    {
        ID: "task-1",
        Execute: func(ctx context.Context) (fastscheduler.TaskResult, error) {
            // 任务1逻辑
            return fastscheduler.TaskResult{...}, nil
        },
    },
    {
        ID: "task-2",
        Execute: func(ctx context.Context) (fastscheduler.TaskResult, error) {
            // 任务2逻辑
            return fastscheduler.TaskResult{...}, nil
        },
    },
}

// 提交批次
batch := scheduler.SubmitBatch(tasks)
batch.Wait()
```

### 接收任务结果

```go
// 创建结果通道
resultChan := make(chan fastscheduler.TaskResult, 1)

task := &fastscheduler.Task{
    ID: "result-task",
    Execute: func(ctx context.Context) (fastscheduler.TaskResult, error) {
        return fastscheduler.TaskResult{...}, nil
    },
    ResultChan: resultChan,
}

// 提交任务
batch := scheduler.SubmitBatch([]*fastscheduler.Task{task})

// 接收结果
select {
case result := <-resultChan:
    fmt.Printf("任务结果: %+v\n", result)
case <-time.After(time.Second):
    fmt.Println("接收结果超时")
}
```

### 错误处理

```go
task := &fastscheduler.Task{
    ID: "error-task",
    Execute: func(ctx context.Context) (fastscheduler.TaskResult, error) {
        // 返回错误
        return fastscheduler.TaskResult{
            HTTPCode:     500,
            BusinessCode: 1,
            Err:         errors.New("任务执行失败"),
        }, nil
    },
}
```

## API 文档

### Task

```go
type Task struct {
    ID         string
    Execute    func(ctx context.Context) (TaskResult, error)
    ResultChan chan<- TaskResult
}
```

### TaskResult

```go
type TaskResult struct {
    HTTPCode     int
    BusinessCode int
    Err          error
    Data         interface{}
}
```

### Scheduler

```go
// 创建调度器
func NewScheduler(poolSize, queueSize int) *Scheduler

// 提交任务批次
func (s *Scheduler) SubmitBatch(tasks []*Task) *Batch

// 等待所有任务完成
func (s *Scheduler) Wait()

// 停止调度器
func (s *Scheduler) Stop()
```

### Batch

```go
// 等待批次中的所有任务完成
func (b *Batch) Wait()

// 检查批次中是否有任务成功
func (b *Batch) IsSuccess() bool
```

## 最佳实践

1. 合理设置 worker 池大小和任务队列容量
2. 使用 context 控制任务超时
3. 正确处理任务返回的错误
4. 在不需要时及时调用 `Stop()` 释放资源
5. 使用 `ResultChan` 接收关键任务的结果

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
