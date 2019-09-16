package utils

import (
	"os"
	"fmt"
)

type Recorder struct {
	file	*os.File 
}

func NewRecorder(filename string) *Recorder {
	m := Recorder{}
	m.file, _ = os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	return &m
}

func (m *Recorder) AddPrivate(key string, value string) {
	line := fmt.Sprintf("[%s : %s]\n", key, value)
	m.file.WriteString(line)
}

func (m *Recorder) AddPublic(key string, value string) {
	line := fmt.Sprintf("<%s : %s>\n", key, value)
	m.file.WriteString(line)	
}

func (m *Recorder) End() {
	m.file.WriteString("%%==%%\n")
}

func (m *Recorder) CleanUp() {
	m.file.Close()
}