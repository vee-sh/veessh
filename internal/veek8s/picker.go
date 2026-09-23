package veek8s

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

const pickerRows = 9

type deadlineReader interface {
	Read([]byte) (int, error)
	SetReadDeadline(time.Time) error
}

// Pick draws a filterable list on /dev/tty. Stdout is left alone.
func Pick(title string, items []string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("open /dev/tty: %w", err)
	}
	defer tty.Close()
	fd := int(tty.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("/dev/tty is not a terminal")
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, old)
	return pickOn(tty, tty, title, items)
}

func pickOn(in deadlineReader, out io.Writer, title string, items []string) (choice string, err error) {
	if len(items) == 0 {
		return "", errors.New("nothing to pick")
	}
	fmt.Fprint(out, "\033[?25l")
	drawn := false
	defer func() {
		if drawn {
			fmt.Fprintf(out, "\033[%dA\033[J", pickerRows)
		}
		fmt.Fprint(out, "\033[?25h")
	}()

	filter := ""
	sel := 0
	for {
		shown := Match(items, filter)
		if sel >= len(shown) {
			sel = len(shown) - 1
		}
		if sel < 0 {
			sel = 0
		}
		drawPicker(out, drawn, title, filter, shown, sel)
		drawn = true
		key, err := readKey(in)
		if err != nil {
			return "", err
		}
		switch key {
		case "enter":
			if len(shown) == 0 {
				continue
			}
			return shown[sel], nil
		case "up":
			if sel > 0 {
				sel--
			}
		case "down":
			if sel < len(shown)-1 {
				sel++
			}
		case "cancel":
			return "", errors.New("canceled")
		case "backspace":
			r := []rune(filter)
			if len(r) > 0 {
				filter = string(r[:len(r)-1])
				sel = 0
			}
		default:
			if key != "" {
				filter += key
				sel = 0
			}
		}
	}
}

func drawPicker(out io.Writer, drawn bool, title, filter string, items []string, sel int) {
	if drawn {
		fmt.Fprintf(out, "\033[%dA", pickerRows)
	}
	start := 0
	if sel >= 8 {
		start = sel - 7
	}
	end := start + 8
	if end > len(items) {
		end = len(items)
	}
	lines := make([]string, 0, pickerRows)
	lines = append(lines, title+"  /"+filter)
	if len(items) == 0 {
		lines = append(lines, "  (no match)")
	}
	for i := start; i < end; i++ {
		prefix := "  "
		if i == sel {
			prefix = "> "
		}
		lines = append(lines, prefix+items[i])
	}
	for len(lines) < pickerRows {
		lines = append(lines, "")
	}
	for _, ln := range lines {
		fmt.Fprintf(out, "\r\033[2K%s\r\n", ln)
	}
}

func readKey(in deadlineReader) (string, error) {
	_ = in.SetReadDeadline(time.Time{})
	b, err := readByte(in)
	if err != nil {
		return "", err
	}
	switch b {
	case 3, 4:
		return "cancel", nil
	case 13, 10:
		return "enter", nil
	case 127, 8:
		return "backspace", nil
	case 0x1b:
		_ = in.SetReadDeadline(time.Now().Add(25 * time.Millisecond))
		b2, err := readByte(in)
		_ = in.SetReadDeadline(time.Time{})
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return "cancel", nil
			}
			return "", err
		}
		if b2 != '[' {
			return "", nil
		}
		b3, err := readByte(in)
		if err != nil {
			return "", err
		}
		switch b3 {
		case 'A':
			return "up", nil
		case 'B':
			return "down", nil
		default:
			return "", nil
		}
	default:
		if b >= 32 && b < 127 {
			return string(rune(b)), nil
		}
		return "", nil
	}
}

func readByte(in deadlineReader) (byte, error) {
	var buf [1]byte
	_, err := in.Read(buf[:])
	return buf[0], err
}
