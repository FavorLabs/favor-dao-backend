package comet_test

import (
	"favor-dao-backend/pkg/comet"
	"testing"
)

const (
	AppId     = "235461af0053efb9"
	AppRegion = "us"
	ApiKey    = "e50156f5ab1294f3f1f67bf685d1a08e9245eea7"
)

func TestNew(t *testing.T) {
	client := comet.New(AppId, AppRegion, ApiKey)

	// Create user
	//u, err := client.Scoped().Users().Create("1", "test", nil)
	//if err != nil {
	//	t.Fatalf("create user: %v", err)
	//}
	//
	//t.Logf("created user: %v", u)

	//g, err := client.Scoped().Groups().Create("1", "test", comet.PublicGroup, &comet.GroupCreateOption{
	//	Owner: "1",
	//})
	//if err != nil {
	//	t.Fatalf("create group: %v", err)
	//}
	//
	//t.Logf("created group: %v", g)

	// Create another user
	//u, err := client.Scoped().Users().Create("2", "foo", nil)
	//if err != nil {
	//	t.Fatalf("create user: %v", err)
	//}
	//
	//t.Logf("created user: %v", u)

	// Add member
	g, err := client.Scoped().Perform("2").Groups().Members("1").Add(comet.GroupMemberOption{
		Participants: []string{"2"},
	})
	if err != nil {
		t.Fatalf("add group member: %v", err)
	}

	t.Logf("after add member: %v", g)
}
