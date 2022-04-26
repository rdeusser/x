// Inspiration came from a project known as zap-pretty: https://github.com/maoueh/zap-pretty
// Instead of a cli tool however, this is a native encoder implementing the zapcore.Encoder interface.
// A lot of this code came directly from the json encoder present in zap: https://github.com/uber-go/zap/blob/master/zapcore/json_encoder.go

package zappretty

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"

	"github.com/rdeusser/x/safepool"
)

const (
	// For JSON-escaping; see jsonEncoder.safeAddString below.
	hex        = "0123456789abcdef"
	timeFormat = "2006-01-02 15:04:05 MST"
)

var (
	nullLiteralBytes = []byte("null")
	cliPool          = safepool.NewPool(&cliEncoder{})
	bufPool          = buffer.NewPool()
	levelColor       = map[zapcore.Level]color.Attribute{
		zapcore.DebugLevel:  color.FgBlue,
		zapcore.InfoLevel:   color.FgGreen,
		zapcore.WarnLevel:   color.FgYellow,
		zapcore.ErrorLevel:  color.FgRed,
		zapcore.DPanicLevel: color.FgRed,
		zapcore.PanicLevel:  color.FgRed,
		zapcore.FatalLevel:  color.FgRed,
	}
)

func Register(cfg zapcore.EncoderConfig) {
	zap.RegisterEncoder("cli", func(_ zapcore.EncoderConfig) (zapcore.Encoder, error) {
		return NewCLIEncoder(cfg), nil
	})
}

type cliEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	openNamespaces int

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc zapcore.ReflectedEncoder
}

func NewCLIEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}

	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = defaultReflectedEncoder
	}

	return &cliEncoder{
		EncoderConfig: &cfg,
		buf:           bufPool.Get(),
	}
}

func (enc *cliEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *cliEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.clone()

	if final.TimeKey != "" {
		final.encodeTimestamp(entry.Time)
	}

	if final.LevelKey != "" && final.EncodeLevel != nil {
		final.encodeLevel(entry.Level)
	}

	if entry.LoggerName != "" && final.NameKey != "" {
		final.encodeLoggerName(entry.LoggerName)
	}

	if entry.Caller.Defined && final.CallerKey != "" {
		final.encodeCaller(entry.Caller)
	}

	if final.MessageKey != "" {
		final.encodeMessage(entry.Message)
	}

	if enc.buf.Len() > 0 {
		final.addElementSeparator()
		final.buf.Write(enc.buf.Bytes())
	}

	if len(fields) > 0 {
		final.buf.AppendString(colorize('{', color.FgWhite, color.Bold))
		final.buf.AppendByte(' ')
	}

	// Add fields.
	for i := range fields {
		fields[i].AddTo(final)
	}

	final.closeOpenNamespaces()

	if len(fields) > 0 {
		final.buf.AppendByte(' ')
		final.buf.AppendString(colorize('}', color.FgWhite, color.Bold))
	}

	final.buf.AppendString(final.LineEnding)

	buf := final.buf

	if final.reflectBuf != nil {
		final.reflectBuf.Free()
	}

	final.EncoderConfig = nil
	final.buf = nil
	final.openNamespaces = 0
	final.reflectBuf = nil
	final.reflectEnc = nil
	cliPool.Put(final)

	return buf, nil
}

// Logging-specific marshalers.
func (enc *cliEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	enc.addKey(key)
	return enc.AppendArray(marshaler)
}

func (enc *cliEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	enc.addKey(key)
	return enc.AppendObject(marshaler)
}

// Built-in types.
func (enc *cliEncoder) AddBinary(key string, value []byte) {
	enc.AddString(key, base64.StdEncoding.EncodeToString(value))
}

func (enc *cliEncoder) AddByteString(key string, value []byte) {
	enc.addKey(key)
	enc.AppendByteString(value)
}

func (enc *cliEncoder) AddBool(key string, value bool) {
	enc.addKey(key)
	enc.AppendBool(value)
}

func (enc *cliEncoder) AddComplex128(key string, value complex128) {
	enc.addKey(key)
	enc.appendComplex(value, 64)
}

func (enc *cliEncoder) AddComplex64(key string, value complex64) {
	enc.addKey(key)
	enc.appendComplex(complex128(value), 32)
}

func (enc *cliEncoder) AddDuration(key string, value time.Duration) {
	enc.addKey(key)
	enc.AppendDuration(value)
}

func (enc *cliEncoder) AddFloat64(key string, value float64) {
	enc.addKey(key)
	enc.AppendFloat64(value)
}

func (enc *cliEncoder) AddFloat32(key string, value float32) {
	enc.addKey(key)
	enc.AppendFloat32(value)
}

func (enc *cliEncoder) AddInt(key string, value int) {
	enc.addKey(key)
	enc.AddInt64(key, int64(value))
}

func (enc *cliEncoder) AddInt64(key string, value int64) {
	enc.addKey(key)
	enc.AppendInt64(value)
}

func (enc *cliEncoder) AddInt32(key string, value int32) {
	enc.addKey(key)
	enc.AddInt64(key, int64(value))
}

func (enc *cliEncoder) AddInt16(key string, value int16) {
	enc.addKey(key)
	enc.AddInt64(key, int64(value))
}

func (enc *cliEncoder) AddInt8(key string, value int8) {
	enc.addKey(key)
	enc.AddInt64(key, int64(value))
}

func (enc *cliEncoder) AddString(key, value string) {
	enc.addKey(key)
	enc.AppendString(value)
}

func (enc *cliEncoder) AddTime(key string, value time.Time) {
	enc.addKey(key)
	enc.AppendTime(value)
}

func (enc *cliEncoder) AddUint(key string, value uint) {
	enc.addKey(key)
	enc.AddUint64(key, uint64(value))
}

func (enc *cliEncoder) AddUint64(key string, value uint64) {
	enc.addKey(key)
	enc.AppendUint64(value)
}

func (enc *cliEncoder) AddUint32(key string, value uint32) {
	enc.addKey(key)
	enc.AddUint64(key, uint64(value))
}

func (enc *cliEncoder) AddUint16(key string, value uint16) {
	enc.addKey(key)
	enc.AddUint64(key, uint64(value))
}

func (enc *cliEncoder) AddUint8(key string, value uint8) {
	enc.addKey(key)
	enc.AddUint64(key, uint64(value))
}

func (enc *cliEncoder) AddUintptr(key string, value uintptr) {
	enc.addKey(key)
	enc.AddUint64(key, uint64(value))
}

// AddReflected uses reflection to serialize arbitrary objects, so it can be
// slow and allocation-heavy.
func (enc *cliEncoder) AddReflected(key string, value interface{}) error {
	valueBytes, err := enc.encodeReflected(value)
	if err != nil {
		return err
	}
	enc.addKey(key)
	_, err = enc.buf.Write(valueBytes)
	return err
}

// OpenNamespace opens an isolated namespace where all subsequent fields will
// be added. Applications can use namespaces to prevent key collisions when
// injecting loggers into sub-components or third-party libraries.
func (enc *cliEncoder) OpenNamespace(key string) {
	enc.addKey(key)
	enc.buf.AppendByte('{')
	enc.openNamespaces++
}

// The following implements the PrimitiveArrayEncoder and ArrayEncoder interfaces.
func (enc *cliEncoder) AppendBool(value bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(value)
}

func (enc *cliEncoder) AppendByteString(value []byte) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddByteString(value)
	enc.buf.AppendByte('"')
}

func (enc *cliEncoder) AppendComplex128(value complex128) { enc.appendComplex(complex128(value), 64) }
func (enc *cliEncoder) AppendComplex64(value complex64)   { enc.appendComplex(complex128(value), 32) }
func (enc *cliEncoder) AppendFloat64(value float64)       { enc.appendFloat(value, 64) }
func (enc *cliEncoder) AppendFloat32(value float32)       { enc.appendFloat(float64(value), 32) }
func (enc *cliEncoder) AppendInt(value int)               { enc.AppendInt64(int64(value)) }
func (enc *cliEncoder) AppendInt64(value int64)           { enc.AppendInt64(int64(value)) }
func (enc *cliEncoder) AppendInt32(value int32)           { enc.AppendInt64(int64(value)) }
func (enc *cliEncoder) AppendInt16(value int16)           { enc.AppendInt64(int64(value)) }
func (enc *cliEncoder) AppendInt8(value int8)             { enc.AppendInt64(int64(value)) }

func (enc *cliEncoder) AppendString(value string) {
	enc.addElementSeparator()
	enc.buf.AppendString(colorize('"', color.FgGreen))
	enc.buf.AppendString(colorize(value, color.FgGreen))
	enc.buf.AppendString(colorize('"', color.FgGreen))
}

func (enc *cliEncoder) AppendUint(value uint)       { enc.AppendUint64(uint64(value)) }
func (enc *cliEncoder) AppendUint64(value uint64)   { enc.AppendUint64(uint64(value)) }
func (enc *cliEncoder) AppendUint32(value uint32)   { enc.AppendUint64(uint64(value)) }
func (enc *cliEncoder) AppendUint16(value uint16)   { enc.AppendUint64(uint64(value)) }
func (enc *cliEncoder) AppendUint8(value uint8)     { enc.AppendUint64(uint64(value)) }
func (enc *cliEncoder) AppendUintptr(value uintptr) { enc.AppendUint64(uint64(value)) }

func (enc *cliEncoder) AppendDuration(value time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(value, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(value))
	}
}

func (enc *cliEncoder) AppendTime(value time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(value, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valueid.
		enc.AppendInt64(value.UnixNano())
	}
}

func (enc *cliEncoder) AppendArray(value zapcore.ArrayMarshaler) error {
	enc.addElementSeparator()
	enc.buf.AppendString(colorize('[', color.FgWhite, color.Bold))
	err := value.MarshalLogArray(enc)
	enc.buf.AppendString(colorize(']', color.FgWhite, color.Bold))
	return err
}

func (enc *cliEncoder) AppendObject(value zapcore.ObjectMarshaler) error {
	// Close ONLY new openNamespaces that are created during
	// AppendObject().
	old := enc.openNamespaces
	enc.openNamespaces = 0
	enc.addElementSeparator()
	enc.buf.AppendString(colorize('{', color.FgWhite, color.Bold))
	err := value.MarshalLogObject(enc)
	enc.buf.AppendString(colorize('}', color.FgWhite, color.Bold))
	enc.closeOpenNamespaces()
	enc.openNamespaces = old
	return err
}

func (enc *cliEncoder) AppendReflected(value interface{}) error {
	valueBytes, err := enc.encodeReflected(value)
	if err != nil {
		return err
	}
	enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *cliEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *cliEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufPool.Get()
		enc.reflectEnc = enc.NewReflectedEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

// appendComplex appends the encoded form of the provided complex128 value.
// precision specifies the encoding precision for the real and imaginary
// components of the complex number.
func (enc *cliEncoder) appendComplex(val complex128, precision int) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, precision)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		enc.buf.AppendByte('+')
	}
	enc.buf.AppendFloat(i, precision)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *cliEncoder) appendFloat(val float64, bitSize int) {
	enc.addElementSeparator()
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

func (enc *cliEncoder) clone() *cliEncoder {
	clone := cliPool.Get()
	clone.EncoderConfig = enc.EncoderConfig
	clone.buf = bufPool.Get()
	return clone
}

func (enc *cliEncoder) encodeTimestamp(timestamp time.Time) {
	enc.buf.WriteString(color.New(color.FgWhite).Sprintf("[%s]", timestamp.Format(timeFormat)))
	enc.buf.WriteString(" ")
}

func (enc *cliEncoder) encodeLevel(level zapcore.Level) {
	if level == zapcore.InfoLevel {
		enc.buf.WriteString(color.New(levelColor[level]).Sprint(level.CapitalString() + " "))
	} else {
		enc.buf.WriteString(color.New(levelColor[level]).Sprint(level.CapitalString()))
	}
	enc.buf.WriteString(" ")
}

func (enc *cliEncoder) encodeLoggerName(logger string) {
	enc.buf.WriteString(color.New(color.FgHiBlack).Sprint(logger))
	enc.buf.WriteString(" ")
}

func (enc *cliEncoder) encodeCaller(caller zapcore.EntryCaller) {
	enc.buf.WriteString(color.New(color.FgHiBlack).Sprintf("(%s)", caller.TrimmedPath()))
	enc.buf.WriteString(" ")
}

func (enc *cliEncoder) encodeMessage(message string) {
	enc.buf.WriteString(color.New(color.FgHiWhite).Sprint(message))
	enc.buf.WriteString(" ")
}

func (enc *cliEncoder) addKey(key string) {
	enc.addElementSeparator()
	enc.buf.AppendString(colorize('"', color.FgBlue, color.Bold))
	enc.buf.AppendString(colorize(key, color.FgBlue, color.Bold))
	enc.buf.AppendString(colorize('"', color.FgBlue, color.Bold))
	enc.buf.AppendString(colorize(':', color.FgBlue, color.Bold))
	enc.buf.AppendByte(' ')
}

func (enc *cliEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		enc.buf.AppendString(colorize(',', color.FgWhite, color.Bold))
		enc.buf.AppendByte(' ')
	}
}

func (enc *cliEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendString(colorize('}', color.FgWhite, color.Bold))
	}
	enc.openNamespaces = 0
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s
// []byte.
func (enc *cliEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a
// single byte.
func (enc *cliEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if 0x20 <= b && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(hex[b>>4])
		enc.buf.AppendByte(hex[b&0xF])
	}
	return true
}

func (enc *cliEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func colorize(arg any, attributes ...color.Attribute) string {
	switch x := arg.(type) {
	case byte:
		return color.New(attributes...).Sprint(string(x))
	case rune:
		return color.New(attributes...).Sprint(string(x))
	default:
		return color.New(attributes...).Sprint(arg)
	}
}

func defaultReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	enc := json.NewEncoder(w)
	// For consistency with our custom JSON encoder.
	enc.SetEscapeHTML(false)
	return enc
}
