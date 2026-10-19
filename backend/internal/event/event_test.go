package event

import (
	"encoding/json/v2"
	"testing"
	"time"
)

func TestEventEncoding(t *testing.T) {
	published := Event{
		SchemaVersion: SchemaVersion, ID: "BUILD1-1", BuildID: "BUILD1",
		Time: time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC), Kind: KindArticlePublished,
		Path: "Essays/Hello-World.md", URLPath: "/essays/hello-world",
	}
	failed := published
	failed.ID, failed.Kind, failed.Path, failed.URLPath, failed.Message = "BUILD1-2", KindValidationFailed, "Bad Name.md", "", `invalid name "Bad Name"`
	for event, want := range map[*Event]string{
		&published: `{"schemaVersion":1,"id":"BUILD1-1","buildId":"BUILD1","time":"2026-10-18T03:00:00Z","kind":"article.published","path":"Essays/Hello-World.md","urlPath":"/essays/hello-world"}`,
		&failed:    `{"schemaVersion":1,"id":"BUILD1-2","buildId":"BUILD1","time":"2026-10-18T03:00:00Z","kind":"validation.failed","path":"Bad Name.md","message":"invalid name \"Bad Name\""}`,
	} {
		got, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("encoded %s\nwant    %s", got, want)
		}
		var decoded Event
		if err := json.Unmarshal(got, &decoded); err != nil || decoded != *event {
			t.Errorf("round trip = %+v (%v), want %+v", decoded, err, *event)
		}
	}
}
