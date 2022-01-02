package routing

import (
	"testing"
)

var tests = []struct {
	name    string
	subject string
	wantErr bool
}{
	{
		name:    "Single token",
		subject: "time",
		wantErr: false,
	},
	{
		name:    "Multiple tokens",
		subject: "time.us.atlanta",
		wantErr: false,
	},
	{
		name:    "Match single token",
		subject: "time.*",
		wantErr: false,
	},
	{
		name:    "Match multiple tokens",
		subject: "time.*.*",
		wantErr: false,
	},
	{
		name:    "Match token starting with",
		subject: "time.>",
		wantErr: false,
	},
	{
		name:    "Match single token and starting with",
		subject: "time.*.>",
		wantErr: false,
	},
	{
		name:    "Match all",
		subject: ">",
		wantErr: false,
	},
	{
		name:    "Match starting with and then error",
		subject: "time.>.*",
		wantErr: true,
	},
}

var matchTests = []struct {
	name         string
	subject      string
	subscription string
	want         bool
}{
	{
		name:         "Exact match single hierarchy",
		subject:      "time",
		subscription: "time",
		want:         true,
	},
	{
		name:         "Exact match multiple hierarchies",
		subject:      "time.us",
		subscription: "time.us",
		want:         true,
	},
	{
		name:         "Single Token",
		subject:      "time.us",
		subscription: "time.*",
		want:         true,
	},
	{
		name:         "Single Token multiple times",
		subject:      "time.us.atlanta",
		subscription: "time.*.*",
		want:         true,
	},
	{
		name:         "Single Token error",
		subject:      "time.us.atlanta",
		subscription: "time.*.california",
		want:         false,
	},
	{
		name:         "Single Token error levels mismatch",
		subject:      "time.us.atlanta",
		subscription: "time.*",
		want:         false,
	},
	{
		name:         "Multiple Token match",
		subject:      "time.us.atlanta",
		subscription: "time.>",
		want:         false,
	},
}

func TestValidateSubject(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateSubject(tt.subject); (err != nil) != tt.wantErr {
				t.Errorf("ValidateSubject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

var validateErr error

func BenchmarkValidateSubjectTest(b *testing.B) {
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := ValidateSubject(tt.subject)
				validateErr = err
			}
		})
	}
}

func TestMatchSubject(t *testing.T) {

	for _, tt := range matchTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchSubject(tt.subject, tt.subscription); got != tt.want {
				t.Errorf("MatchSubject() = %v, want %v", got, tt.want)
			}
		})
	}
}

var matchOk bool

func BenchmarkMatchSubject(b *testing.B) {
	for _, tt := range matchTests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ok := MatchSubject(tt.subject, tt.subscription)
				matchOk = ok
			}
		})
	}
}
