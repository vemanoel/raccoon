package progress

import (
	"fmt"
	"io"
)

type Bar struct {
	reader io.Reader
	total  int64
	done   int64
	desc   string
}

func New(reader io.Reader, total int64) *Bar {
	return &Bar{
		reader: reader,
		total:  total,
	}
}

func (self *Bar) SetDescription(desc string) {
	self.desc = desc
}

func (self *Bar) Read(b []byte) (int, error) {
	n, err := self.reader.Read(b)
	self.done += int64(n)

	if self.total == 0 {
		fmt.Printf("\r%s %s", self.desc, formatBytes(self.done))
	} else {
		percent := float64(self.done) / float64(self.total) * 100

		fmt.Printf(
			"\r%s %.2f%% (%s / %s)",
			self.desc, percent, formatBytes(self.done), formatBytes(self.total),
		)
	}

	return n, err
}

func (self *Bar) Finish(msg string) {
	fmt.Printf("\r\033[K%s %s\n", self.desc, msg)
}
