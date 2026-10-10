package accountid

import (
	"context"
	"encoding/json"
	"testing"
)

type robotCaller struct{ body []byte }

func (c *robotCaller) Post(ctx context.Context, op, path string, body, result any) error {
	c.body, _ = json.Marshal(body)
	return json.Unmarshal([]byte(`{"userid_list":[{"open_userid":"o","userid":"u"}],"invalid_open_userid_list":["bad"]}`), result)
}
func TestRobotIdentityConversionUsesApplicationCaller(t *testing.T) {
	caller := new(robotCaller)
	c := NewWithCaller(caller)
	result, err := c.RobotOpenUserIDToUserID(context.Background(), []string{"o", "bad"})
	if err != nil || len(result.InvalidOpenUserIDs) != 1 || string(caller.body) != `{"open_userid_list":["o","bad"]}` {
		t.Fatalf("%s %+v %v", caller.body, result, err)
	}
	if _, err = c.RobotOpenUserIDToUserID(context.Background(), make([]string, 1001)); err == nil {
		t.Fatal("over limit accepted")
	}
}
