package stern

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/fatih/color"
)

func TestIsIncludeTestOptions(t *testing.T) {
	msg := "this is a log message"

	tests := []struct {
		include  []*regexp.Regexp
		expected bool
	}{
		{
			include:  []*regexp.Regexp{},
			expected: true,
		},
		{
			include: []*regexp.Regexp{
				regexp.MustCompile(`this is not`),
			},
			expected: false,
		},
		{
			include: []*regexp.Regexp{
				regexp.MustCompile(`this is`),
			},
			expected: true,
		},
	}

	for i, tt := range tests {
		o := &TailOptions{Include: tt.include}
		if o.IsInclude(msg) != tt.expected {
			t.Errorf("%d: expected %s, but actual %s", i, fmt.Sprint(tt.expected), fmt.Sprint(!tt.expected))
		}
	}
}

func TestUpdateTimezoneAndFormat(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Tokyo")

	tests := []struct {
		name     string
		format   string
		message  string
		expected string
		err      string
	}{
		{
			"normal case",
			"", // default format is used if empty
			"2021-04-18T03:54:44.764981564Z",
			"2021-04-18T12:54:44.764981564+09:00",
			"",
		},
		{
			"padding",
			"",
			"2021-04-18T03:54:44.764981500Z",
			"2021-04-18T12:54:44.764981500+09:00",
			"",
		},
		{
			"timestamp required on non timestamp message",
			"",
			"",
			"",
			"missing timestamp",
		},
		{
			"not UTC",
			"",
			"2021-08-03T01:26:29.953994922+02:00",
			"2021-08-03T08:26:29.953994922+09:00",
			"",
		},
		{
			"RFC3339Nano format removed trailing zeros",
			"",
			"2021-06-20T08:20:30.331385Z",
			"2021-06-20T17:20:30.331385000+09:00",
			"",
		},
		{
			"Specified the short format",
			TimestampFormatShort,
			"2021-06-20T08:20:30.331385Z",
			"06-20 17:20:30",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tailOptions := &TailOptions{
				Location:        location,
				TimestampFormat: tt.format,
			}

			message, err := tailOptions.UpdateTimezoneAndFormat(tt.message)
			if tt.expected != message {
				t.Errorf("expected %q, but actual %q", tt.expected, message)
			}

			if err != nil && tt.err != err.Error() {
				t.Errorf("expected %q, but actual %q", tt.err, err)
			}
		})
	}
}

func TestHighlighIncludedString(t *testing.T) {
	tests := []struct {
		msg      string
		include  []*regexp.Regexp
		expected string
	}{
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`test`),
			},
			"\x1b[31;1mtest\x1b[0;22m matched",
		},
		{
			"test not-matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`hoge`),
			},
			"test not-matched",
		},
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`not-matched`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmatched\x1b[0;22m",
		},
		{
			"test multiple matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`multiple`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmultiple\x1b[0;22m \x1b[31;1mmatched\x1b[0;22m",
		},
		{
			"test match on the longer one",
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			"test \x1b[31;1mmatch on the longer one\x1b[0;22m",
		},
	}

	orig := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = orig
	}()

	for i, tt := range tests {
		o := &TailOptions{Include: tt.include}
		actual := o.HighlightMatchedString(tt.msg)
		if actual != tt.expected {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, actual)
		}
	}
}

func TestIncludeAndHighlightMatchedString(t *testing.T) {
	tests := []struct {
		msg       string
		include   []*regexp.Regexp
		highlight []*regexp.Regexp
		expected  string
	}{
		{
			"test matched with highlight",
			[]*regexp.Regexp{
				regexp.MustCompile(`test`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`highlight`),
			},
			"\x1b[31;1mtest\x1b[0;22m matched with \x1b[31;1mhighlight\x1b[0;22m",
		},
		{
			"test not-matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`hoge`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`highlight`),
			},
			"test not-matched",
		},
		{
			"test matched with highlight",
			[]*regexp.Regexp{
				regexp.MustCompile(`not-matched`),
				regexp.MustCompile(`matched`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`no-with-highlight`),
				regexp.MustCompile(`with highlight`),
			},
			"test \x1b[31;1mmatched\x1b[0;22m \x1b[31;1mwith highlight\x1b[0;22m",
		},
		{
			"test multiple matched with many highlight",
			[]*regexp.Regexp{
				regexp.MustCompile(`multiple`),
				regexp.MustCompile(`matched`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`many`),
				regexp.MustCompile(`highlight`),
			},
			"test \x1b[31;1mmultiple\x1b[0;22m \x1b[31;1mmatched\x1b[0;22m with \x1b[31;1mmany\x1b[0;22m \x1b[31;1mhighlight\x1b[0;22m",
		},
		{
			"test match on the longer one",
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			"test \x1b[31;1mmatch on the longer one\x1b[0;22m",
		},
	}

	orig := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = orig
	}()

	for i, tt := range tests {
		o := &TailOptions{Include: tt.include, Highlight: tt.highlight}
		actual := o.HighlightMatchedString(tt.msg)
		if actual != tt.expected {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, actual)
		}
	}
}

func TestHighlightMatchedString(t *testing.T) {
	tests := []struct {
		msg       string
		highlight []*regexp.Regexp
		expected  string
	}{
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`test`),
			},
			"\x1b[31;1mtest\x1b[0;22m matched",
		},
		{
			"test not-matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`hoge`),
			},
			"test not-matched",
		},
		{
			"test matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`not-matched`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmatched\x1b[0;22m",
		},
		{
			"test multiple matched",
			[]*regexp.Regexp{
				regexp.MustCompile(`multiple`),
				regexp.MustCompile(`matched`),
			},
			"test \x1b[31;1mmultiple\x1b[0;22m \x1b[31;1mmatched\x1b[0;22m",
		},
		{
			"test match on the longer one",
			[]*regexp.Regexp{
				regexp.MustCompile(`match`),
				regexp.MustCompile(`match on the longer one`),
			},
			"test \x1b[31;1mmatch on the longer one\x1b[0;22m",
		},
	}

	orig := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = orig
	}()

	for i, tt := range tests {
		o := &TailOptions{Highlight: tt.highlight}
		actual := o.HighlightMatchedString(tt.msg)
		if actual != tt.expected {
			t.Errorf("%d: expected %q, but actual %q", i, tt.expected, actual)
		}
	}
}
