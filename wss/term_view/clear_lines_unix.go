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

func clearLines(outDev FdWriter, lines int) {
    _, _ = fmt.Fprint(outDev, strings.Repeat(clear, lines))
}
