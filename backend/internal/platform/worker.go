package platform

import (
	"context"
	"log"
	"sync"
	"time"
)

// Task 定义了一个后台任务函数
type Task func(ctx context.Context)

// BackgroundWorker 处理非阻塞的后台任务池
type BackgroundWorker struct {
	taskChan chan Task
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewBackgroundWorker 初始化工作池，queueSize 为任务队列长度，workers 为并发协程数
func NewBackgroundWorker(queueSize int, workers int) *BackgroundWorker {
	ctx, cancel := context.WithCancel(context.Background())
	bw := &BackgroundWorker{
		taskChan: make(chan Task, queueSize),
		ctx:      ctx,
		cancel:   cancel,
	}

	// 启动工作协程
	for i := 0; i < workers; i++ {
		bw.wg.Add(1)
		go bw.worker()
	}

	return bw
}

func (bw *BackgroundWorker) worker() {
	defer bw.wg.Done()
	for {
		select {
		case task, ok := <-bw.taskChan:
			if !ok {
				return
			}
			// 执行任务，捕获 panic 防止整个进程崩溃
			bw.runTask(task)
		case <-bw.ctx.Done():
			return
		}
	}
}

func (bw *BackgroundWorker) runTask(task Task) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Worker] 任务执行异常崩溃: %v", r)
		}
	}()
	
	// 为任务提供 30 秒的独立执行时间
	taskCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task(taskCtx)
}

// Submit 提交一个异步任务到队列
func (bw *BackgroundWorker) Submit(task Task) {
	select {
	case bw.taskChan <- task:
		// 成功入队
	default:
		// 队列已满，为了不阻塞主链路，可以选择丢弃或打印警告
		log.Printf("[Worker] 警告: 异步任务队列已满，任务被丢弃")
	}
}

// Stop 优雅关闭工作池，等待进行中的任务完成
func (bw *BackgroundWorker) Stop() {
	close(bw.taskChan)
	bw.cancel()
	bw.wg.Wait()
}
