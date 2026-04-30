package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

//go:generate mockgen --build_flags=--mod=readonly -source $GOFILE -destination ./mock/jiramock.go -package jiramock

type Client interface {
	GetOpenOHSSTickets(ctx context.Context, clusterID, externalID string) ([]OHSSTicket, error)
}

type OHSSTicket struct {
	Key     string
	Summary string
	Status  string
}

type sdkClient struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

func New(token string) Client {
	return &sdkClient{
		httpClient: &http.Client{},
		baseURL:    "https://redhat.atlassian.net",
		token:      token,
	}
}

func (c *sdkClient) GetOpenOHSSTickets(ctx context.Context, clusterID, externalID string) ([]OHSSTicket, error) {
	jql := fmt.Sprintf(
		`project = "OpenShift Hosted SRE Support" AND statusCategory != Done AND (`+
			`"Cluster ID" ~ "%s" OR "Cluster ID" ~ "%s" OR description ~ "%s" OR description ~ "%s")`,
		clusterID, externalID, clusterID, externalID,
	)

	reqURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&maxResults=20&fields=key,summary,status",
		c.baseURL, url.QueryEscape(jql))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jira request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result jiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode Jira response: %w", err)
	}

	tickets := make([]OHSSTicket, 0, len(result.Issues))
	for _, issue := range result.Issues {
		tickets = append(tickets, OHSSTicket{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
		})
	}

	return tickets, nil
}

type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
}

type jiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
		Status  struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}
