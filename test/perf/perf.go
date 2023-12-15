package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

const (
	lineCount = 5000
	lineSep   = '\n'
)

func main() {
	var wg sync.WaitGroup
	var lineChan = make(chan string, lineCount)

	cmd := exec.Command("python3", "perf.py")
	cmd.Stderr = cmd.Stdout

	pipe, _ := cmd.StdoutPipe()

	reader := bufio.NewReader(pipe)
	if err := cmd.Start(); err != nil {
		fmt.Printf("failed to start: %s", err.Error())
		os.Exit(1)
	}

	wg.Add(1)

	go func(reader *bufio.Reader, lineChan chan string) {
		for {
			line, err := reader.ReadBytes(lineSep)
			if err != nil {
				fmt.Printf("failed to read: %s", err.Error())
				break
			}
			lineChan <- string(line)
		}
		close(lineChan)
		wg.Done()
	}(reader, lineChan)

	wg.Add(1)

	go func(cmd *exec.Cmd) {
		_ = cmd.Wait()
		wg.Done()
	}(cmd)

	for line := range lineChan {
		fmt.Println(line)
	}

	wg.Wait()

	os.Exit(0)
}
