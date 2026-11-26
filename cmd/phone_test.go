package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Encratahq/cli/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// phoneFake records the last phone call and answers with a fixed result.
type phoneFake struct {
	api.API
	op, phone, mode string
	include         []string
	result          string
}

func (f *phoneFake) PhoneLookup(_ context.Context, feature, phone string) (json.RawMessage, error) {
	f.op, f.phone = feature, phone
	return json.RawMessage(f.result), nil
}

func (f *phoneFake) PhoneSearch(_ context.Context, phone string, include []string) (json.RawMessage, error) {
	f.op, f.phone, f.include = "search", phone, include
	return json.RawMessage(f.result), nil
}

func (f *phoneFake) PhoneVoip(_ context.Context, phone, mode string) (json.RawMessage, error) {
	f.op, f.phone, f.mode = "voip", phone, mode
	return json.RawMessage(f.result), nil
}

// newPhoneTestCmd wraps a phone subcommand with its flags reset to their
// defaults, so one test's flag values never leak into the next.
func newPhoneTestCmd(t *testing.T, base *cobra.Command, flags map[string]string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: base.Use, RunE: base.RunE, Args: base.Args}
	c.Flags().AddFlagSet(base.Flags())
	c.Flags().VisitAll(func(fl *pflag.Flag) {
		if sv, ok := fl.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = fl.Value.Set(fl.DefValue)
		}
		fl.Changed = false
	})
	for k, v := range flags {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("set --%s: %v", k, err)
		}
	}
	c.SetContext(context.Background())
	return c
}

func TestPhoneFeatureCommandsNormalizeAndRoute(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		want string
	}{
		{phoneValidityCmd, "validity"},
		{phoneRiskCmd, "risk"},
		{phoneBreachesCmd, "breaches"},
		{phoneIdentityCmd, "identity"},
	}
	for _, tc := range cases {
		f := &phoneFake{result: `{"phone":"+14155552671","status":"valid","verdict":"safe","breached":false,"found":false,"credits":1}`}
		restore := withFake(t, f)
		c := newPhoneTestCmd(t, tc.cmd, nil)
		if err := c.RunE(c, []string{"+1 (415) 555-2671"}); err != nil {
			t.Fatalf("%s: %v", tc.want, err)
		}
		restore()
		if f.op != tc.want || f.phone != "+14155552671" {
			t.Fatalf("%s: called %q with %q", tc.want, f.op, f.phone)
		}
	}
}

func TestPhoneRejectsNumberWithoutCountryCode(t *testing.T) {
	f := &phoneFake{}
	defer withFake(t, f)()
	c := newPhoneTestCmd(t, phoneValidityCmd, nil)
	if err := c.RunE(c, []string{"4155552671"}); err == nil {
		t.Fatal("expected a format error")
	}
	if f.op != "" {
		t.Fatal("no request may be sent for an invalid number")
	}
}

func TestPhoneSearchIncludeAndVoipMode(t *testing.T) {
	f := &phoneFake{result: `{"phone":"+14155552671","valid":true,"credits":1}`}
	defer withFake(t, f)()

	c := newPhoneTestCmd(t, phoneSearchCmd, map[string]string{"include": " Risk ,carrier"})
	if err := c.RunE(c, []string{"+14155552671"}); err != nil {
		t.Fatal(err)
	}
	if f.op != "search" || len(f.include) != 2 || f.include[0] != "risk" || f.include[1] != "carrier" {
		t.Fatalf("search include = %v", f.include)
	}

	c = newPhoneTestCmd(t, phoneVoipCmd, nil)
	if err := c.RunE(c, []string{"+14155552671"}); err != nil {
		t.Fatal(err)
	}
	if f.mode != "static" {
		t.Fatalf("default voip mode = %q", f.mode)
	}
	c = newPhoneTestCmd(t, phoneVoipCmd, map[string]string{"current": "true"})
	if err := c.RunE(c, []string{"+14155552671"}); err != nil {
		t.Fatal(err)
	}
	if f.mode != "current" {
		t.Fatalf("--current voip mode = %q", f.mode)
	}
}

func TestPhoneFailOnFinding(t *testing.T) {
	cases := []struct {
		name    string
		cmd     *cobra.Command
		result  string
		finding bool
	}{
		{"risk high", phoneRiskCmd, `{"verdict":"high_risk"}`, true},
		{"risk suspicious", phoneRiskCmd, `{"verdict":"suspicious"}`, true},
		{"risk safe", phoneRiskCmd, `{"verdict":"safe"}`, false},
		{"breached", phoneBreachesCmd, `{"breached":true,"total_breaches":2}`, true},
		{"not breached", phoneBreachesCmd, `{"breached":false}`, false},
		{"voip", phoneVoipCmd, `{"is_voip":true}`, true},
		{"not voip", phoneVoipCmd, `{"is_voip":false}`, false},
	}
	for _, tc := range cases {
		for _, flag := range []bool{false, true} {
			f := &phoneFake{result: tc.result}
			restore := withFake(t, f)
			flags := map[string]string{}
			if flag {
				flags["fail-on-finding"] = "true"
			}
			c := newPhoneTestCmd(t, tc.cmd, flags)
			err := c.RunE(c, []string{"+14155552671"})
			restore()
			want := flag && tc.finding
			if got := errors.Is(err, errBreachDetected); got != want {
				t.Errorf("%s (--fail-on-finding=%v): finding exit = %v, want %v (err %v)", tc.name, flag, got, want, err)
			}
		}
	}
}
