package perf

import (
	"testing"
)

func TestParseRegistryValid(t *testing.T) {
	doc := `{
		"version": 1,
		"workloads": [
			{"id": "backend-healthcheck", "kind": "http", "target": "http://127.0.0.1:8080/healthcheck", "rate": 50, "duration": "10s"},
			{"id": "frontend-root", "kind": "browser", "target": "/", "repetitions": 30},
			{"id": "frontend-local", "kind": "browser", "target": "http://localhost:3000/", "repetitions": 10}
		]
	}`
	r, err := ParseRegistry([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Workloads) != 3 {
		t.Fatalf("got %d workloads", len(r.Workloads))
	}
}

func TestParseRegistryRejects(t *testing.T) {
	cases := map[string]string{
		"duplicate id": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","rate":1,"duration":"1s"},
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/y","rate":1,"duration":"1s"}]}`,
		"non-local target": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"https://example.com/x","rate":1,"duration":"1s"}]}`,
		"non-http scheme": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"file:///etc/passwd","rate":1,"duration":"1s"}]}`,
		"missing rate": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","duration":"1s"}]}`,
		"zero rate": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","rate":0,"duration":"1s"}]}`,
		"bad duration": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","rate":1,"duration":"later"}]}`,
		"http with repetitions": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","rate":1,"duration":"1s","repetitions":3}]}`,
		"browser missing repetitions": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"/"}]}`,
		"browser with rate": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"/","repetitions":3,"rate":1}]}`,
		"browser remote target": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"https://example.com/","repetitions":3}]}`,
		"browser protocol-relative target": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"//example.com/","repetitions":3}]}`,
		"browser backslash target": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"/\\example.com","repetitions":3}]}`,
		"browser tab target": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"/\t/example.com","repetitions":3}]}`,
		"browser newline target": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"/\n/example.com","repetitions":3}]}`,
		"browser space target": `{"version":1,"workloads":[
			{"id":"a","kind":"browser","target":"/ foo","repetitions":3}]}`,
		"bad id": `{"version":1,"workloads":[
			{"id":"Has Space","kind":"http","target":"http://127.0.0.1:8080/x","rate":1,"duration":"1s"}]}`,
		"unknown kind": `{"version":1,"workloads":[
			{"id":"a","kind":"rpc","target":"http://127.0.0.1:8080/x","rate":1,"duration":"1s"}]}`,
		"bad version": `{"version":2,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","rate":1,"duration":"1s"}]}`,
		"unknown member": `{"version":1,"workloads":[
			{"id":"a","kind":"http","target":"http://127.0.0.1:8080/x","rate":1,"duration":"1s","weight":2}]}`,
		"empty": `{"version":1,"workloads":[]}`,
	}
	for name, doc := range cases {
		if _, err := ParseRegistry([]byte(doc)); err == nil {
			t.Errorf("%s: expected parse error", name)
		}
	}
}

func TestLocalTargetsAccepted(t *testing.T) {
	for _, target := range []string{
		"http://127.0.0.1:8080/healthcheck",
		"http://localhost:8080/healthcheck",
		"http://[::1]:8080/healthcheck",
		"https://127.0.0.1/healthcheck",
	} {
		doc := `{"version":1,"workloads":[{"id":"a","kind":"http","target":"` + target + `","rate":1,"duration":"1s"}]}`
		if _, err := ParseRegistry([]byte(doc)); err != nil {
			t.Errorf("%s: %v", target, err)
		}
	}
}

func TestRegistryRejectsNonJSON(t *testing.T) {
	if _, err := ParseRegistry([]byte("not json")); err == nil {
		t.Error("expected parse error")
	}
	if _, err := ParseRegistry([]byte(`{"version":1}`)); err == nil {
		t.Error("expected parse error for missing workloads")
	}
}
