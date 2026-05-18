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
	isSync   bool // 是否为同步模式（主要用于测试）
}

// NewBackgroundWorker 初始化工作池
func NewBackgroundWorker(queueSize int, workers int) *BackgroundWorker {
	return newWorker(queueSize, workers, false)
}

// NewSyncWorker 初始化一个同步执行的工作池（仅用于测试）
func NewSyncWorker() *BackgroundWorker {
	return newWorker(0, 0, true)
}

func newWorker(queueSize int, workers int, isSync bool) *BackgroundWorker {
	ctx, cancel := context.WithCancel(context.Background())
	bw := &BackgroundWorker{
		taskChan: make(chan Task, queueSize),
		ctx:      ctx,
		cancel:   cancel,
		isSync:   isSync,
	}

	if !isSync {
		// 异步模式下启动工作协程
		for i := 0; i < workers; i++ {
			bw.wg.Add(1)
			go bw.worker()
		}
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
	if bw.isSync {
		// 同步模式直接运行
		bw.runTask(task)
		return
	}

	select {
	case bw.taskChan <- task:
		// 成功入队
	default:
		// 队列已满
		log.Printf("[Worker] 警告: 异步任务队列已满，任务被丢弃")
	}
}

// Stop 优雅关闭工作池
func (bw *BackgroundWorker) Stop() {
	if bw.isSync {
		bw.cancel()
		return
	}
	close(bw.taskChan)
	bw.cancel()
	bw.wg.Wait()
}
