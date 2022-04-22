package fjira

import (
	"github.com/mk5/fjira/internal/app"
	"github.com/mk5/fjira/internal/jira"
	"log"
)

func goIntoProjectsSearch() {
	projectsView := NewProjectsSearchView()
	app.GetApp().SetView(projectsView)
}

func goIntoIssuesSearch(project *jira.JiraProject) {
	issuesSearchView := NewIssuesSearchView(project)
	app.GetApp().SetView(issuesSearchView)
}

// TODO - migrate to static one
func (f *Fjira) goIntoIssuesSearch(project *jira.JiraProject) {
	goIntoIssuesSearch(project)
}

func goIntoIssueView(issue *jira.JiraIssue) {
	issueView := NewIssueView(issue)
	app.GetApp().SetView(issueView)
}

func goIntoIssueViewFetchIssue(issueKey string) {
	app.GetApp().Loading(true)
	api, _ := GetApi()
	issue, err := api.GetIssueDetailed(issueKey)
	if err != nil {
		log.Fatalln(err)
	}
	app.GetApp().Loading(false)
	issueView := NewIssueView(issue)
	app.GetApp().SetView(issueView)
}

// TODO - migrate to static one
func (f *Fjira) goIntoIssueView(issue *jira.JiraIssue) {
	goIntoIssueView(issue)
}

func goIntoChangeStatus(issue *jira.JiraIssue) {
	statusChangeView := NewStatusChangeView(issue)
	app.GetApp().SetView(statusChangeView)
}

func goIntoChangeAssignment(issue *jira.JiraIssue) {
	assignChangeView := NewAssignChangeView(issue)
	app.GetApp().SetView(assignChangeView)
}
