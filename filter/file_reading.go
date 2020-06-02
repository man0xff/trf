package filter

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func readHead(f *os.File, n int) ([]string, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}
	r, head := bufio.NewReader(f), make([]string, 0, n)
	for len(head) < n {
		line, err := r.ReadString('\n')
		switch err {
		case nil:
		case io.EOF:
			return head, nil
		default:
			return nil, err
		}
		line = strings.TrimRight(line, "\n")
		if line != "" {
			head = append(head, line)
		}
	}
	return head, nil
}

func readTail(f *os.File, n int) ([]string, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int64(n * 4 * 1024)
	if size > info.Size() {
		size = info.Size()
	}
	if _, err := f.Seek(-size, 2); err != nil {
		return nil, err
	}
	text, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	lines := bytes.Split(text, []byte("\n"))
	tail := make([]string, 0, n)
	if len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	for i := len(lines) - 1; i >= 0 && len(tail) < n; i-- {
		if len(lines[i]) != 0 {
			tail = append(tail, string(lines[i]))
		}
	}
	return tail, nil
}

func readLines(file string, n int) ([]string, []string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	head, err := readHead(f, n)
	if err != nil {
		return nil, nil, err
	}
	tail, err := readTail(f, n)
	if err != nil {
		return nil, nil, err
	}
	return head, tail, nil
}
