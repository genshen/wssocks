// +build !windows

package term_view

import (
	"fmt"
	"strings"
)

// ESC is the ASCII code for escape character
const ESC = 27

// clear the line and move the cursor up
var clear = fmt.Sprintf("%c[%dA%c[2K", ESC, 1, ESC)

func (w *Writer) clearLines() {
	_, _ = fmt.Fprint(w.OutDev, strings.Repeat(clear, w.lineCount))
}
