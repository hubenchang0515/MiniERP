package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	"minioa/internal/service"
)

// Feed the observer in proxy read order. Decoding skips unrelated JSON fields
// without storing the response or delaying a stream until its final event.
type aiUsageObserver struct {
	input *io.PipeWriter
	done  chan *service.AITokenUsage
}

func observeAIUsage(path string, stream bool) *aiUsageObserver {
	reader, writer := io.Pipe()
	observer := &aiUsageObserver{input: writer, done: make(chan *service.AITokenUsage, 1)}
	go func() {
		defer reader.Close()
		if stream {
			observer.done <- readStreamingUsage(reader, path)
			return
		}
		usage, _ := readUsageJSON(reader, path, false)
		observer.done <- usage
	}()
	return observer
}

func (o *aiUsageObserver) observe(chunk []byte) {
	// The parser can terminate early on malformed data. The original body
	// still goes to the client through ReverseProxy.
	_, _ = o.input.Write(chunk)
}

func (o *aiUsageObserver) finish() *service.AITokenUsage {
	_ = o.input.Close()
	return <-o.done
}

type upstreamAIUsage struct {
	Prompt     *int64 `json:"prompt_tokens"`
	Completion *int64 `json:"completion_tokens"`
	Input      *int64 `json:"input_tokens"`
	Output     *int64 `json:"output_tokens"`
}

func (u upstreamAIUsage) tokens(path string) *service.AITokenUsage {
	input, output := u.Prompt, u.Completion
	switch path {
	case "/responses":
		input, output = u.Input, u.Output
	case "/embeddings":
		zero := int64(0)
		output = &zero
	}
	if input == nil || output == nil || *input < 0 || *output < 0 {
		return nil
	}
	return &service.AITokenUsage{Input: *input, Output: *output}
}

func readUsageJSON(r io.Reader, path string, stream bool) (*service.AITokenUsage, error) {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	usage, kind, err := readUsageObject(decoder, stream && path == "/responses")
	if err != nil {
		return nil, err
	}
	if stream && path == "/responses" && kind != "response.completed" && kind != "response.incomplete" && kind != "response.failed" {
		return nil, nil
	}
	return usage.tokens(path), nil
}

func readUsageObject(decoder *json.Decoder, nestedResponse bool) (usage upstreamAIUsage, kind string, err error) {
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return usage, kind, err
	}
	for decoder.More() {
		var key json.Token
		key, err = decoder.Token()
		if err != nil {
			return usage, kind, err
		}
		switch {
		case key == "response" && nestedResponse:
			usage, _, err = readUsageObject(decoder, false)
		case key == "type" && nestedResponse:
			err = decoder.Decode(&kind)
		case key == "usage":
			// Null or a repeated field replaces previous values rather than
			// accidentally merging usage from different JSON objects.
			usage = upstreamAIUsage{}
			err = decoder.Decode(&usage)
		default:
			err = skipJSONValue(decoder)
		}
		if err != nil {
			return usage, kind, err
		}
	}
	_, err = decoder.Token()
	return usage, kind, err
}

func skipJSONValue(decoder *json.Decoder) error {
	depth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
		if depth == 0 {
			return nil
		}
	}
}

func readStreamingUsage(r io.Reader, path string) *service.AITokenUsage {
	reader := bufio.NewReader(&sseLineReader{reader: r})
	if prefix, _ := reader.Peek(3); bytes.Equal(prefix, []byte("\xef\xbb\xbf")) {
		_, _ = reader.Discard(3)
	}
	var latest *service.AITokenUsage
	for {
		event := sseUsageEvent{reader: reader}
		usage, _ := readUsageJSON(&event, path, true)
		_, _ = io.Copy(io.Discard, &event)
		if !event.complete {
			return latest
		}
		if usage != nil {
			latest = usage
		}
	}
}

// SSE allows LF, CRLF and CR, including CRLF split between transport reads.
type sseLineReader struct {
	reader  io.Reader
	afterCR bool
}

func (s *sseLineReader) Read(p []byte) (int, error) {
	for {
		n, err := s.reader.Read(p)
		written := 0
		for _, c := range p[:n] {
			if s.afterCR && c == '\n' {
				s.afterCR = false
				continue
			}
			s.afterCR = c == '\r'
			if s.afterCR {
				c = '\n'
			}
			p[written] = c
			written++
		}
		if written > 0 || n == 0 || err != nil {
			return written, err
		}
	}
}

// Expose one SSE event's data lines as a JSON reader. Fragmenting long lines
// keeps large response.completed events from requiring an event-sized buffer.
type sseUsageEvent struct {
	reader   *bufio.Reader
	rest     []byte
	inLine   bool
	dataLine bool
	done     bool
	complete bool
}

func (s *sseUsageEvent) Read(p []byte) (int, error) {
	for len(s.rest) == 0 {
		if s.done {
			return 0, io.EOF
		}
		line, err := s.reader.ReadSlice('\n')
		if err != nil && err != bufio.ErrBufferFull {
			s.done = true
			return 0, io.EOF
		}
		if !s.inLine {
			if len(line) == 1 && line[0] == '\n' {
				s.done, s.complete = true, true
				return 0, io.EOF
			}
			line, s.dataLine = bytes.CutPrefix(line, []byte("data:"))
			if s.dataLine {
				line, _ = bytes.CutPrefix(line, []byte(" "))
			}
		}
		s.inLine = err == bufio.ErrBufferFull
		if s.dataLine {
			s.rest = line
		}
	}
	n := copy(p, s.rest)
	s.rest = s.rest[n:]
	return n, nil
}
