package veek8s

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type fakeTTY struct {
	in  []byte
	i   int
	dl  time.Time
	err error
}

func (f *fakeTTY) SetReadDeadline(t time.Time) error {
	f.dl = t
	return nil
}

func (f *fakeTTY) Read(p []byte) (int, error) {
	if f.err != nil && f.i >= len(f.in) {
		return 0, f.err
	}
	if f.i >= len(f.in) {
		if !f.dl.IsZero() {
			return 0, os.ErrDeadlineExceeded
		}
		return 0, io.EOF
	}
	p[0] = f.in[f.i]
	f.i++
	return 1, nil
}

func TestPickOn(t *testing.T) {
	items := []string{"dev", "dev-west", "prod"}
	cases := []struct {
		name string
		keys string
		want string
		err  string
	}{
		{name: "enter first", keys: "\r", want: "dev"},
		{name: "down", keys: "\x1b[B\r", want: "dev-west"},
		{name: "down stops", keys: "\x1b[B\x1b[B\x1b[B\r", want: "prod"},
		{name: "up stays", keys: "\x1b[A\r", want: "dev"},
		{name: "filter exact", keys: "dev\r", want: "dev"},
		{name: "filter one", keys: "w\r", want: "dev-west"},
		{name: "backspace", keys: "x\x7f\r", want: "dev"},
		{name: "no match then clear", keys: "z\r\x7f\r", want: "dev"},
		{name: "ignore high byte", keys: "\xff\r", want: "dev"},
		{name: "bare esc", keys: "\x1b", err: "canceled"},
		{name: "ctrl-d", keys: "\x04", err: "canceled"},
		{name: "newline", keys: "\n", want: "dev"},
		{name: "backspace byte", keys: "x\x08\r", want: "dev"},
		{name: "other arrow", keys: "\x1b[C\r", want: "dev"},
		{name: "scroll", keys: strings.Repeat("\x1b[B", 12) + "\r", want: "n11"},
		{name: "esc then other", keys: "\x1bX\r", want: "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := items
			if tc.name == "scroll" {
				list = []string{"n0", "n1", "n2", "n3", "n4", "n5", "n6", "n7", "n8", "n9", "n10", "n11"}
			}
			var buf bytes.Buffer
			got, err := pickOn(&fakeTTY{in: []byte(tc.keys)}, &buf, "context", list)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("err %v", err)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("got %q err %v", got, err)
			}
			if !strings.Contains(buf.String(), "\033[?25h") {
				t.Fatal("cursor not restored")
			}
			if tc.err == "" && !strings.Contains(buf.String(), "> "+tc.want) {
				t.Fatalf("selection not drawn:\n%s", buf.String())
			}
		})
	}
	if _, err := pickOn(&fakeTTY{}, io.Discard, "context", nil); err == nil {
		t.Fatal("empty list")
	}
	_, err := pickOn(&fakeTTY{err: errors.New("closed")}, io.Discard, "context", items)
	if err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("io err %v", err)
	}
}
