// Copyright 2023 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logexperimental

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
)

// JSONWriter writes into a JSON terminal
type JSONWriter struct {
	out   *logrus.Logger
	file  *logrus.Entry
	stage string
	buf   *bytes.Buffer
}

type jsonMessage struct {
	Level     string `json:"level"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// JSONLogFormat formats the messages into json struct
type JSONLogFormat struct {
	Level     string `json:"level"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// Format formats the message
func (f *JSONLogFormat) Format(entry *logrus.Entry) ([]byte, error) {
	level := strings.ToLower(entry.Level.String())
	if entry.Level == logrus.WarnLevel {
		level = "info"
	}
	outputJSON := &jsonMessage{
		Level:     level,
		Timestamp: time.Now().Unix(),
		Stage:     f.Stage,
		Message:   entry.Message,
	}
	messageJSON, err := json.Marshal(outputJSON)
	if err != nil {
		return nil, err
	}
	messageJSON = []byte(string(messageJSON) + "\n")
	return messageJSON, nil
}

// newJSONWriter creates a new JSONWriter
func newJSONWriter(out *logrus.Logger, file *logrus.Entry) *JSONWriter {
	return &JSONWriter{
		out:  out,
		file: file,
	}
}

func (w *JSONWriter) SetStage(stage string) {
	w.stage = stage
}

// Debug writes a debug-level log
func (w *JSONWriter) Debug(args ...interface{}) {
	w.out.Debug(args...)
	if w.file != nil {
		w.file.Debug(args...)
	}
}

// Debugf writes a debug-level log with a format
func (w *JSONWriter) Debugf(format string, args ...interface{}) {
	w.out.Debugf(format, args...)
	if w.file != nil {
		w.file.Debugf(format, args...)
	}
}

// Info writes a info-level log
func (w *JSONWriter) Info(args ...interface{}) {
	w.out.Info(args...)
	if w.file != nil {
		w.file.Info(args...)
	}
}

// Infof writes a info-level log with a format
func (w *JSONWriter) Infof(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	if w.file != nil {
		w.file.Infof(format, args...)
	}
}

// Error writes a error-level log
func (w *JSONWriter) Error(args ...interface{}) {
	w.out.Error(args...)
	if w.file != nil {
		w.file.Error(args...)
	}
}

// Errorf writes a error-level log with a format
func (w *JSONWriter) Errorf(format string, args ...interface{}) {
	w.out.Errorf(format, args...)
	if w.file != nil {
		w.file.Errorf(format, args...)
	}
}

// Fatalf writes a error-level log with a format
func (w *JSONWriter) Fatalf(format string, args ...interface{}) {
	if w.file != nil {
		w.file.Errorf(format, args...)
	}

	w.out.Fatalf(format, args...)
}

// Green writes a line in green
func (w *JSONWriter) Green(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	w.FPrintln(w.out.Out, fmt.Sprintf(format, args...))
}

// Yellow writes a line in yellow
func (w *JSONWriter) Yellow(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	w.FPrintln(w.out.Out, fmt.Sprintf(format, args...))
}

// Success prints a message with the success symbol first, and the text in green
func (w *JSONWriter) Success(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	w.FPrintln(w.out.Out, fmt.Sprintf("%s %s", successSymbol, fmt.Sprintf(format, args...)))
}

// Information prints a message with the information symbol first, and the text in blue
func (w *JSONWriter) Information(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	w.FPrintln(w.out.Out, fmt.Sprintf("%s %s", informationSymbol, fmt.Sprintf(format, args...)))
}

// Question prints a message with the question symbol first, and the text in magenta
func (*JSONWriter) Question(_ string, _ ...interface{}) error {
	return fmt.Errorf("can't ask questions on json mode")
}

// Warning prints a message with the warning symbol first, and the text in yellow
func (w *JSONWriter) Warning(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	msg := fmt.Sprintf("%s %s", warningSymbol, fmt.Sprintf(format, args...))
	if msg != "" {
		msg := w.convertToJSON("warn", w.stage, msg)
		if msg != "" {
			fmt.Fprintln(w.out.Out, msg)
		}
	}
}

// FWarning prints a message with the warning symbol first, and the text in yellow
func (w *JSONWriter) FWarning(writer io.Writer, format string, args ...interface{}) {
	w.out.Infof(format, args...)
	msg := fmt.Sprintf("%s %s", warningSymbol, fmt.Sprintf(format, args...))
	if msg != "" {
		msg := w.convertToJSON("warn", w.stage, msg)
		if msg != "" {
			fmt.Fprintln(writer, msg)
		}
	}
}

// Hint prints a message with the text in blue
func (w *JSONWriter) Hint(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	w.FPrintln(w.out.Out, fmt.Sprintf(format, args...))
}

// Fail prints a message with the error symbol first, and the text in red
func (w *JSONWriter) Fail(format string, args ...interface{}) {
	w.out.Infof(format, args...)
	msg := fmt.Sprintf("%s %s", errorSymbol, fmt.Sprintf(format, args...))
	if msg != "" {
		if w.stage == "" {
			w.stage = "Internal server error"
		}
		msg = w.convertToJSON(ErrorLevel, w.stage, msg)
		if msg != "" {
			w.buf.WriteString(msg)
			w.buf.WriteString("\n")
			fmt.Fprintln(w.out.Out, msg)
		}
	}
}

// Println writes a line with colors
func (w *JSONWriter) Println(args ...interface{}) {
	w.FPrintln(w.out.Out, args...)
}

// Fprintf prints a line with format
func (w *JSONWriter) Fprintf(writer io.Writer, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	if strings.HasSuffix(format, "\n") {
		w.FPrintln(writer, msg)
		return
	}
	if msg != "" && writer == w.out.Out {
		msg = w.convertToJSON(InfoLevel, w.stage, msg)
		if msg != "" {
			w.buf.WriteString(msg)
			w.buf.WriteString("\n")
		}
		fmt.Fprint(writer, msg)
	}

}

// FPrintln prints a line with format
func (w *JSONWriter) FPrintln(writer io.Writer, args ...interface{}) {
	msg := fmt.Sprint(args...)
	if msg != "" && writer == w.out.Out {
		msg = w.convertToJSON(InfoLevel, w.stage, msg)
		if msg != "" {
			w.buf.WriteString(msg)
			w.buf.WriteString("\n")
			fmt.Fprintln(writer, msg)
		}

	}
}

// Print writes a line with colors
func (w *JSONWriter) Print(args ...interface{}) {
	msg := w.convertToJSON(InfoLevel, w.stage, fmt.Sprint(args...))
	if msg != "" {
		w.buf.WriteString(msg)
		w.buf.WriteString("\n")
		fmt.Fprint(w.out.Out, msg)
	}

}

// Printf writes a line with format
func (w *JSONWriter) Printf(format string, a ...interface{}) {
	w.Fprintf(w.out.Out, format, a...)
}

// IsInteractive checks if the writer is interactive
func (*JSONWriter) IsInteractive() bool {
	return false
}

func (w *JSONWriter) convertToJSON(level, stage, message string) string {
	message = strings.TrimRightFunc(message, unicode.IsSpace)
	if stage == "" || message == "" {
		return ""
	}
	messageStruct := jsonMessage{
		Level:     level,
		Message:   ansiRegex.ReplaceAllString(message, ""),
		Stage:     stage,
		Timestamp: time.Now().Unix(),
	}
	messageJSON, err := json.Marshal(messageStruct)
	if err != nil {
		w.Infof("error marshalling message: %s", err)
		return ""
	}
	return string(messageJSON)
}

// AddToBuffer logs into the buffer and writes to stdout if its a json writer
func (w *JSONWriter) AddToBuffer(level, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	msg = w.convertToJSON(level, w.stage, msg)
	if msg != "" {
		w.buf.WriteString(msg)
		w.buf.WriteString("\n")
		fmt.Fprintln(w.out.Out, msg)
	}
}

// Write logs into the buffer but does not print anything
func (w *JSONWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	msg = w.convertToJSON(InfoLevel, w.stage, msg)
	if msg != "" {
		if _, err := w.out.Out.Write([]byte("")); err != nil {
			return 0, err
		}
	}
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	return w.out.Out.Write([]byte(msg))
}
