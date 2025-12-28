package vt

import (
	"io"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// KeyMod represents a key modifier.
type KeyMod = uv.KeyMod

// Modifier keys.
const (
	ModShift = uv.ModShift
	ModAlt   = uv.ModAlt
	ModCtrl  = uv.ModCtrl
	ModMeta  = uv.ModMeta
)

// KeyPressEvent represents a key press event.
type KeyPressEvent = uv.KeyPressEvent

// SendKey returns the default key map.
func (e *Emulator) SendKey(k uv.KeyEvent) {
	key, ok := k.(KeyPressEvent)
	if !ok {
		return
	}
	ack := e.isModeSet(ansi.ModeCursorKeys)    // Application cursor keys mode
	akk := e.isModeSet(ansi.ModeNumericKeypad) // Application keypad keys mode

	seq := sequenceForKeyPress(key, ack, akk)
	if seq == "" {
		return
	}
	io.WriteString(e.pw, seq) //nolint:errcheck,gosec
}

func sequenceForKeyPress(key KeyPressEvent, ack, akk bool) string {
	seqPrefix := ""
	if key.Mod&ModAlt != 0 {
		seqPrefix = "\x1b"
		key.Mod &^= ModAlt // Remove the Alt modifier for easier matching
	}

	//nolint:godox
	// FIXME: We remove any Base and Shifted codes to properly handle
	// comparison. This is a workaround for the fact that we don't support
	// extended keys yet.
	key.BaseCode = 0
	key.ShiftedCode = 0

	seq := rawSequenceForKeyPress(key, ack, akk)
	if seq == "" {
		return seqPrefix
	}
	return seqPrefix + seq
}

func rawSequenceForKeyPress(key KeyPressEvent, ack, akk bool) string {
	if key.Mod == ModCtrl {
		if seq, ok := ctrlKeySequence(key.Code); ok {
			return seq
		}
	}
	if key.Mod == ModShift && key.Code == KeyTab {
		return "\x1b[Z"
	}
	if key.Mod != 0 {
		return ""
	}
	if seq, ok := basicKeySequence(key.Code); ok {
		return seq
	}
	if seq, ok := arrowKeySequence(key.Code, ack); ok {
		return seq
	}
	if seq, ok := navKeySequence(key.Code); ok {
		return seq
	}
	if seq, ok := functionKeySequence(key.Code); ok {
		return seq
	}
	if seq, ok := keypadSequence(key.Code, akk); ok {
		return seq
	}
	return string(key.Code)
}

func ctrlKeySequence(code rune) (string, bool) {
	seq, ok := ctrlKeyMap[code]
	return seq, ok
}

func basicKeySequence(code rune) (string, bool) {
	seq, ok := basicKeyMap[code]
	return seq, ok
}

func arrowKeySequence(code rune, app bool) (string, bool) {
	if app {
		seq, ok := arrowKeyAppMap[code]
		return seq, ok
	}
	seq, ok := arrowKeyMap[code]
	return seq, ok
}

func navKeySequence(code rune) (string, bool) {
	seq, ok := navKeyMap[code]
	return seq, ok
}

func functionKeySequence(code rune) (string, bool) {
	seq, ok := functionKeyMap[code]
	return seq, ok
}

func keypadSequence(code rune, app bool) (string, bool) {
	if app {
		seq, ok := keypadAppMap[code]
		return seq, ok
	}
	seq, ok := keypadKeyMap[code]
	return seq, ok
}

var ctrlKeyMap = map[rune]string{
	KeySpace: "\x00",
	'a':      "\x01",
	'b':      "\x02",
	'c':      "\x03",
	'd':      "\x04",
	'e':      "\x05",
	'f':      "\x06",
	'g':      "\x07",
	'h':      "\x08",
	'i':      "\x09",
	'j':      "\x0a",
	'k':      "\x0b",
	'l':      "\x0c",
	'm':      "\x0d",
	'n':      "\x0e",
	'o':      "\x0f",
	'p':      "\x10",
	'q':      "\x11",
	'r':      "\x12",
	's':      "\x13",
	't':      "\x14",
	'u':      "\x15",
	'v':      "\x16",
	'w':      "\x17",
	'x':      "\x18",
	'y':      "\x19",
	'z':      "\x1a",
	'[':      "\x1b",
	'\\':     "\x1c",
	']':      "\x1d",
	'^':      "\x1e",
	'_':      "\x1f",
}

var basicKeyMap = map[rune]string{
	KeyEnter:     "\r",
	KeyTab:       "\t",
	KeyBackspace: "\x7f",
	KeyEscape:    "\x1b",
}

var arrowKeyMap = map[rune]string{
	KeyUp:    "\x1b[A",
	KeyDown:  "\x1b[B",
	KeyRight: "\x1b[C",
	KeyLeft:  "\x1b[D",
}

var arrowKeyAppMap = map[rune]string{
	KeyUp:    "\x1bOA",
	KeyDown:  "\x1bOB",
	KeyRight: "\x1bOC",
	KeyLeft:  "\x1bOD",
}

var navKeyMap = map[rune]string{
	KeyInsert: "\x1b[2~",
	KeyDelete: "\x1b[3~",
	KeyHome:   "\x1b[H",
	KeyEnd:    "\x1b[F",
	KeyPgUp:   "\x1b[5~",
	KeyPgDown: "\x1b[6~",
}

var functionKeyMap = map[rune]string{
	KeyF1:  "\x1bOP",
	KeyF2:  "\x1bOQ",
	KeyF3:  "\x1bOR",
	KeyF4:  "\x1bOS",
	KeyF5:  "\x1b[15~",
	KeyF6:  "\x1b[17~",
	KeyF7:  "\x1b[18~",
	KeyF8:  "\x1b[19~",
	KeyF9:  "\x1b[20~",
	KeyF10: "\x1b[21~",
	KeyF11: "\x1b[23~",
	KeyF12: "\x1b[24~",
}

var keypadKeyMap = map[rune]string{
	KeyKp0:        "0",
	KeyKp1:        "1",
	KeyKp2:        "2",
	KeyKp3:        "3",
	KeyKp4:        "4",
	KeyKp5:        "5",
	KeyKp6:        "6",
	KeyKp7:        "7",
	KeyKp8:        "8",
	KeyKp9:        "9",
	KeyKpEnter:    "\r",
	KeyKpEqual:    "=",
	KeyKpMultiply: "*",
	KeyKpPlus:     "+",
	KeyKpComma:    ",",
	KeyKpMinus:    "-",
	KeyKpDecimal:  ".",
}

var keypadAppMap = map[rune]string{
	KeyKp0:        "\x1bOp",
	KeyKp1:        "\x1bOq",
	KeyKp2:        "\x1bOr",
	KeyKp3:        "\x1bOs",
	KeyKp4:        "\x1bOt",
	KeyKp5:        "\x1bOu",
	KeyKp6:        "\x1bOv",
	KeyKp7:        "\x1bOw",
	KeyKp8:        "\x1bOx",
	KeyKp9:        "\x1bOy",
	KeyKpEnter:    "\x1bOM",
	KeyKpEqual:    "\x1bOX",
	KeyKpMultiply: "\x1bOj",
	KeyKpPlus:     "\x1bOk",
	KeyKpComma:    "\x1bOl",
	KeyKpMinus:    "\x1bOm",
	KeyKpDecimal:  "\x1bOn",
}

// Key codes.
const (
	KeyExtended         = uv.KeyExtended
	KeyUp               = uv.KeyUp
	KeyDown             = uv.KeyDown
	KeyRight            = uv.KeyRight
	KeyLeft             = uv.KeyLeft
	KeyBegin            = uv.KeyBegin
	KeyFind             = uv.KeyFind
	KeyInsert           = uv.KeyInsert
	KeyDelete           = uv.KeyDelete
	KeySelect           = uv.KeySelect
	KeyPgUp             = uv.KeyPgUp
	KeyPgDown           = uv.KeyPgDown
	KeyHome             = uv.KeyHome
	KeyEnd              = uv.KeyEnd
	KeyKpEnter          = uv.KeyKpEnter
	KeyKpEqual          = uv.KeyKpEqual
	KeyKpMultiply       = uv.KeyKpMultiply
	KeyKpPlus           = uv.KeyKpPlus
	KeyKpComma          = uv.KeyKpComma
	KeyKpMinus          = uv.KeyKpMinus
	KeyKpDecimal        = uv.KeyKpDecimal
	KeyKpDivide         = uv.KeyKpDivide
	KeyKp0              = uv.KeyKp0
	KeyKp1              = uv.KeyKp1
	KeyKp2              = uv.KeyKp2
	KeyKp3              = uv.KeyKp3
	KeyKp4              = uv.KeyKp4
	KeyKp5              = uv.KeyKp5
	KeyKp6              = uv.KeyKp6
	KeyKp7              = uv.KeyKp7
	KeyKp8              = uv.KeyKp8
	KeyKp9              = uv.KeyKp9
	KeyKpSep            = uv.KeyKpSep
	KeyKpUp             = uv.KeyKpUp
	KeyKpDown           = uv.KeyKpDown
	KeyKpLeft           = uv.KeyKpLeft
	KeyKpRight          = uv.KeyKpRight
	KeyKpPgUp           = uv.KeyKpPgUp
	KeyKpPgDown         = uv.KeyKpPgDown
	KeyKpHome           = uv.KeyKpHome
	KeyKpEnd            = uv.KeyKpEnd
	KeyKpInsert         = uv.KeyKpInsert
	KeyKpDelete         = uv.KeyKpDelete
	KeyKpBegin          = uv.KeyKpBegin
	KeyF1               = uv.KeyF1
	KeyF2               = uv.KeyF2
	KeyF3               = uv.KeyF3
	KeyF4               = uv.KeyF4
	KeyF5               = uv.KeyF5
	KeyF6               = uv.KeyF6
	KeyF7               = uv.KeyF7
	KeyF8               = uv.KeyF8
	KeyF9               = uv.KeyF9
	KeyF10              = uv.KeyF10
	KeyF11              = uv.KeyF11
	KeyF12              = uv.KeyF12
	KeyF13              = uv.KeyF13
	KeyF14              = uv.KeyF14
	KeyF15              = uv.KeyF15
	KeyF16              = uv.KeyF16
	KeyF17              = uv.KeyF17
	KeyF18              = uv.KeyF18
	KeyF19              = uv.KeyF19
	KeyF20              = uv.KeyF20
	KeyF21              = uv.KeyF21
	KeyF22              = uv.KeyF22
	KeyF23              = uv.KeyF23
	KeyF24              = uv.KeyF24
	KeyF25              = uv.KeyF25
	KeyF26              = uv.KeyF26
	KeyF27              = uv.KeyF27
	KeyF28              = uv.KeyF28
	KeyF29              = uv.KeyF29
	KeyF30              = uv.KeyF30
	KeyF31              = uv.KeyF31
	KeyF32              = uv.KeyF32
	KeyF33              = uv.KeyF33
	KeyF34              = uv.KeyF34
	KeyF35              = uv.KeyF35
	KeyF36              = uv.KeyF36
	KeyF37              = uv.KeyF37
	KeyF38              = uv.KeyF38
	KeyF39              = uv.KeyF39
	KeyF40              = uv.KeyF40
	KeyF41              = uv.KeyF41
	KeyF42              = uv.KeyF42
	KeyF43              = uv.KeyF43
	KeyF44              = uv.KeyF44
	KeyF45              = uv.KeyF45
	KeyF46              = uv.KeyF46
	KeyF47              = uv.KeyF47
	KeyF48              = uv.KeyF48
	KeyF49              = uv.KeyF49
	KeyF50              = uv.KeyF50
	KeyF51              = uv.KeyF51
	KeyF52              = uv.KeyF52
	KeyF53              = uv.KeyF53
	KeyF54              = uv.KeyF54
	KeyF55              = uv.KeyF55
	KeyF56              = uv.KeyF56
	KeyF57              = uv.KeyF57
	KeyF58              = uv.KeyF58
	KeyF59              = uv.KeyF59
	KeyF60              = uv.KeyF60
	KeyF61              = uv.KeyF61
	KeyF62              = uv.KeyF62
	KeyF63              = uv.KeyF63
	KeyCapsLock         = uv.KeyCapsLock
	KeyScrollLock       = uv.KeyScrollLock
	KeyNumLock          = uv.KeyNumLock
	KeyPrintScreen      = uv.KeyPrintScreen
	KeyPause            = uv.KeyPause
	KeyMenu             = uv.KeyMenu
	KeyMediaPlay        = uv.KeyMediaPlay
	KeyMediaPause       = uv.KeyMediaPause
	KeyMediaPlayPause   = uv.KeyMediaPlayPause
	KeyMediaReverse     = uv.KeyMediaReverse
	KeyMediaStop        = uv.KeyMediaStop
	KeyMediaFastForward = uv.KeyMediaFastForward
	KeyMediaRewind      = uv.KeyMediaRewind
	KeyMediaNext        = uv.KeyMediaNext
	KeyMediaPrev        = uv.KeyMediaPrev
	KeyMediaRecord      = uv.KeyMediaRecord
	KeyLowerVol         = uv.KeyLowerVol
	KeyRaiseVol         = uv.KeyRaiseVol
	KeyMute             = uv.KeyMute
	KeyLeftShift        = uv.KeyLeftShift
	KeyLeftAlt          = uv.KeyLeftAlt
	KeyLeftCtrl         = uv.KeyLeftCtrl
	KeyLeftSuper        = uv.KeyLeftSuper
	KeyLeftHyper        = uv.KeyLeftHyper
	KeyLeftMeta         = uv.KeyLeftMeta
	KeyRightShift       = uv.KeyRightShift
	KeyRightAlt         = uv.KeyRightAlt
	KeyRightCtrl        = uv.KeyRightCtrl
	KeyRightSuper       = uv.KeyRightSuper
	KeyRightHyper       = uv.KeyRightHyper
	KeyRightMeta        = uv.KeyRightMeta
	KeyIsoLevel3Shift   = uv.KeyIsoLevel3Shift
	KeyIsoLevel5Shift   = uv.KeyIsoLevel5Shift
	KeyBackspace        = uv.KeyBackspace
	KeyTab              = uv.KeyTab
	KeyEnter            = uv.KeyEnter
	KeyReturn           = uv.KeyReturn
	KeyEscape           = uv.KeyEscape
	KeyEsc              = uv.KeyEsc
	KeySpace            = uv.KeySpace
)
