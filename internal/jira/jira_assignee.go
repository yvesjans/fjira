package jira

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	DoAssigneePath = "/rest/api/2/issue/%s/assignee"
)

type assigneeRequestBody struct {
	AccountId *string `json:"accountId"`
}

func (api httpJiraApi) DoAssignee(issueId string, accountId *string) error {
	url := fmt.Sprintf(DoAssigneePath, issueId)
	body := &assigneeRequestBody{AccountId: accountId}
	jsonBody, _ := json.Marshal(body)
	_, err := api.jiraRequest("PUT", url, nil, strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	return nil
}
