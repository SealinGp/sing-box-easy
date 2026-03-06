package process

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"syscall"
)

type ProcessManager struct {
	cmd     *exec.Cmd
	logChan chan string
	cancel  context.CancelFunc
	ctx     context.Context
	wg      sync.WaitGroup
}

func NewProcessManager() *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessManager{
		logChan: make(chan string, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (pm *ProcessManager) Start(name string, args ...string) error {
	cmd := exec.CommandContext(pm.ctx, name, args...)
	pm.cmd = cmd

	// 获取 stderr 管道
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("获取stderr管道失败: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取stdout管道失败: %w", err)
	}

	// 启动进程
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("启动进程失败: %w", err)
	}

	// 启动读取协程
	pm.wg.Go(func() {
		pm.readPipe(stdout, "stdout")
	})
	pm.wg.Go(func() {
		pm.readPipe(stderr, "stderr")
	})
	pm.wg.Go(pm.monitorProcess)

	// 启动资源回收协程
	logChan := pm.logChan
	go func() {
		pm.wg.Wait()
		close(logChan)
	}()

	return nil
}

func (pm *ProcessManager) monitorProcess() {
	logMsg := "进程正常退出"
	err := pm.cmd.Wait()
	if err != nil {
		// 检查是否是信号终止
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					signal := status.Signal()
					pm.logChan <- fmt.Sprintf("进程被信号 %v 终止", signal)
					return
				}
			}
		}
		logMsg = fmt.Sprintf("进程异常退出: %v", err)
	}

	select {
	case pm.logChan <- logMsg:
	default:
	}
}

func (pm *ProcessManager) GetLogChanel() <-chan string {
	return pm.logChan
}

func (pm *ProcessManager) Stop() {
	pm.cancel()
}

func isProcessKilledError(err error) bool {
	if err == nil {
		return false
	}

	// 常见的进程被杀死错误
	errStr := err.Error()
	return errStr == "read |0: file already closed" ||
		errStr == "read |0: bad file descriptor" ||
		errStr == "read |0: input/output error" ||
		errStr == "read |0: broken pipe"
}

func (pm *ProcessManager) readPipe(pipe io.ReadCloser, name string) {
	reader := bufio.NewReader(pipe)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// 处理不同类型的错误
			switch {
			case err == io.EOF:
				return

			case isProcessKilledError(err):
				pm.logChan <- fmt.Sprintf("进程被终止: %v", err)
				return

			default:
				pm.logChan <- fmt.Sprintf("读取错误: %v", err)
				return
			}
		}

		// 正常输出日志
		select {
		case pm.logChan <- fmt.Sprintf("[%s] %s", name, line):
		case <-pm.ctx.Done():
			return
		default:
			log.Printf("日志通道已满，丢弃: %s", line)
		}
	}
}
