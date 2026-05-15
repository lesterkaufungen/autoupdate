package autoupdate

import (
	"io"
)

type Pipe struct {
	writer io.Writer
	reader io.Reader
}

func (p *Pipe) Write(b []byte) (n int, err error) {
	if p.writer == nil {
		return 0, io.ErrClosedPipe
	}
	return p.writer.Write(b)
}

func (p *Pipe) Read(b []byte) (n int, err error) {
	if p.reader == nil {
		return 0, io.ErrClosedPipe
	}
	return p.reader.Read(b)
}

func (p *Pipe) setWriter(w io.Writer) {
	p.writer = w
}

func (p *Pipe) setReader(r io.Reader) {
	p.reader = r
}
