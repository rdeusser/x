package zappretty

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	gofuzz "github.com/google/gofuzz"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var epoch = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	color.NoColor = false
	goleak.VerifyTestMain(m)
}

func TestPrettyOutput(t *testing.T) {
	testcases := []struct {
		name  string
		entry zapcore.Entry
		want  string
	}{
		{
			name: "info with caller",
			entry: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       epoch,
				LoggerName: "main",
				Message:    "hello world",
				Caller: zapcore.EntryCaller{
					Defined:  true,
					File:     "foo.go",
					Line:     42,
					Function: "foo.Bar",
				},
				Stack: "foo",
			},
			want: fmt.Sprintf("\x1b[37m[%s]\x1b[0m \x1b[32mINFO \x1b[0m \x1b[90mmain\x1b[0m \x1b[90m(foo.go:42)\x1b[0m \x1b[97mhello world\x1b[0m \n", epoch.Format(timeFormat)),
		},
	}

	for _, tc := range testcases {
		encoder := NewCLIEncoder(EncoderTestEncoderConfig())

		t.Run(tc.name, func(t *testing.T) {
			out, err := encoder.EncodeEntry(tc.entry, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, out.String(), "Unexpected output")
		})
	}
}

func TestFuzzLog(t *testing.T) {
	atom := zap.NewAtomicLevel()
	cfg := zap.NewProductionEncoderConfig()

	Register(cfg)

	cliEncoder := NewCLIEncoder(cfg)
	jsonEncoder := zapcore.NewJSONEncoder(cfg)

	tmp, err := os.CreateTemp("", "test.*.log")
	assert.NoError(t, err)

	writer, closer, err := zap.Open(tmp.Name())
	assert.NoError(t, err)
	defer closer()

	leveler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
		return level >= atom.Level()
	})

	core := zapcore.NewTee(
		zapcore.NewCore(cliEncoder, zapcore.AddSync(ioutil.Discard), leveler),
		zapcore.NewCore(jsonEncoder, writer, leveler),
	)

	logger := zap.New(core).Named("zappretty")
	defer logger.Sync()

	corpus := make([]string, 0)

	f := gofuzz.New()

	for i := 0; i < 1000; i++ {
		var s string

		f.Fuzz(&s)
		corpus = append(corpus, s)
	}

	go func() {
		for i := 0; i < 1000; i++ {
			logger.Info(corpus[i])
		}
	}()

	for i := 0; i < 1000; i++ {
		logger.Info(corpus[i])
	}
}
