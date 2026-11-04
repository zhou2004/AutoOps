package config

import "testing"

func TestDatabaseWorkerStartupMode(t *testing.T) {
	for _, tc := range []struct {
		setting, server string
		want            bool
	}{{"", "debug", true}, {"auto", "debug", true}, {"auto", "release", false}, {"auto", "test", false}, {"external", "debug", false}, {"embedded", "release", true}} {
		got, e := EmbeddedDatabaseWorker(tc.setting, tc.server)
		if e != nil || got != tc.want {
			t.Fatalf("%+v got=%v error=%v", tc, got, e)
		}
	}
	if _, e := EmbeddedDatabaseWorker("typo", "debug"); e == nil {
		t.Fatal("invalid startup configuration accepted")
	}
}
