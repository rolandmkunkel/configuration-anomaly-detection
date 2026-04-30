package controller

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/configuration-anomaly-detection/pkg/aiconfig"
	"github.com/openshift/configuration-anomaly-detection/pkg/investigations"
	"github.com/openshift/configuration-anomaly-detection/pkg/investigations/aiassisted"
	"github.com/openshift/configuration-anomaly-detection/pkg/investigations/investigation"
	"github.com/openshift/configuration-anomaly-detection/pkg/jira"
	"github.com/openshift/configuration-anomaly-detection/pkg/logging"
	"github.com/openshift/configuration-anomaly-detection/pkg/ocm"
	"github.com/openshift/configuration-anomaly-detection/pkg/pagerduty"
)

type PagerDutyController struct {
	config     CommonConfig
	pd         PagerDutyConfig
	pdClient   *pagerduty.SdkClient
	jiraClient jira.Client
	investigationRunner
}

func (c *PagerDutyController) Investigate(ctx context.Context) error {
	experimentalEnabledVar := os.Getenv("CAD_EXPERIMENTAL_ENABLED")
	experimentalEnabled, _ := strconv.ParseBool(experimentalEnabledVar)
	alertInvestigation := investigations.GetInvestigation(c.pdClient.GetTitle(), experimentalEnabled)

	clusterID, err := c.pdClient.RetrieveClusterID()
	if err != nil {
		return err
	}

	// Update logger with cluster ID now that we have it
	c.logger = logging.InitLogger(c.config.LogLevel, c.config.Identifier, clusterID)
	c.logger.Infof("Investigating incident '%s' for service '%s (%s)'", c.pdClient.GetIncidentRef(), c.pdClient.GetServiceID(), c.pdClient.GetServiceName())

	// Check if we should escalate to AI or not
	if experimentalEnabled {
		alertInvestigation = handleUnsupportedAlertWithAI(alertInvestigation, c.pdClient)
		if alertInvestigation == nil {
			err := c.pdClient.EscalateIncident()
			if err != nil {
				return fmt.Errorf("could not escalate unsupported alert: %w", err)
			}
			return nil
		}
	}

	// Phase 1: post cluster context note (appears below investigation note in PD)
	c.enrichWithContext(ctx, clusterID)

	// Phase 2: run investigation
	return c.runInvestigation(ctx, clusterID, alertInvestigation, c.pdClient)
}

const maxServiceLogs = 5

func (c *PagerDutyController) enrichWithContext(ctx context.Context, clusterID string) {
	cluster, err := c.ocmClient.GetClusterInfo(clusterID)
	if err != nil {
		logging.Warnf("Context enrichment: failed to get cluster info: %v", err)
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 Cluster Context\n")
	sb.WriteString("===========================\n")

	c.appendServiceLogs(&sb, cluster)
	c.appendOHSSTickets(ctx, &sb, cluster)

	if err := c.pdClient.AddNote(sb.String()); err != nil {
		logging.Warnf("Context enrichment: failed to post context note to PagerDuty: %v", err)
	}
}

func (c *PagerDutyController) appendServiceLogs(sb *strings.Builder, cluster *cmv1.Cluster) {
	since := time.Now().AddDate(0, 0, -30)
	filter := fmt.Sprintf("created_at >= '%s'", since.Format("2006-01-02T15:04:05Z"))

	resp, err := c.ocmClient.GetServiceLog(cluster, filter)
	if err != nil {
		fmt.Fprintf(sb, "⚠️ Could not fetch service logs: %v\n", err)
		return
	}

	items := resp.Items().Slice()
	if len(items) == 0 {
		sb.WriteString("Service Logs (past 30 days): None\n")
		return
	}

	fmt.Fprintf(sb, "Service Logs (past 30 days): %d total\n", len(items))
	displayed := min(len(items), maxServiceLogs)
	for _, entry := range items[:displayed] {
		fmt.Fprintf(sb, "  [%s] [%s] %s\n",
			entry.Timestamp().Format("2006-01-02"),
			entry.Severity(),
			entry.Summary(),
		)
	}
	if len(items) > maxServiceLogs {
		fmt.Fprintf(sb, "  ... and %d more\n", len(items)-maxServiceLogs)
	}
}

func (c *PagerDutyController) appendOHSSTickets(ctx context.Context, sb *strings.Builder, cluster *cmv1.Cluster) {
	if c.jiraClient == nil {
		return
	}

	tickets, err := c.jiraClient.GetOpenOHSSTickets(ctx, cluster.ID(), cluster.ExternalID())
	if err != nil {
		fmt.Fprintf(sb, "⚠️ Could not fetch OHSS tickets: %v\n", err)
		return
	}

	if len(tickets) == 0 {
		sb.WriteString("Open OHSS Tickets: None\n")
		return
	}

	fmt.Fprintf(sb, "Open OHSS Tickets: %d\n", len(tickets))
	for _, t := range tickets {
		fmt.Fprintf(sb, "  [%s] %s — %s\n", t.Key, t.Summary, t.Status)
	}
}

func escalateDocumentationMismatch(docErr *ocm.DocumentationMismatchError, resources *investigation.Resources, pdClient *pagerduty.SdkClient) {
	message := docErr.EscalationMessage()

	if resources != nil && resources.Notes != nil {
		resources.Notes.AppendWarning("%s", message)
		message = resources.Notes.String()
	}

	if pdClient == nil {
		logging.Errorf("Failed to obtain PagerDuty client, unable to escalate documentation mismatch to PagerDuty notes.")
		return
	}

	if err := pdClient.EscalateIncidentWithNote(message); err != nil {
		logging.Errorf("Failed to escalate documentation mismatch notes to PagerDuty: %v", err)
		return
	}

	logging.Info("Escalated documentation mismatch to PagerDuty")
}

// handleUnsupportedAlertWithAI checks if AI is enabled for unsupported alerts.
// If AI is enabled, returns an AI investigation. If disabled, escalates the alert.
// Returns errAlertEscalated if the alert was escalated.
func handleUnsupportedAlertWithAI(alertInvestigation investigation.Investigation, pdClient *pagerduty.SdkClient) investigation.Investigation {
	if alertInvestigation != nil {
		return alertInvestigation
	}

	// Parse AI config
	aiConfig, err := aiconfig.ParseAIAgentConfig()
	if err != nil {
		aiConfig = &aiconfig.AIAgentConfig{Enabled: false}
		logging.Warnf("Failed to parse AI agent configuration, disabling AI investigation: %v", err)
	}

	// Escalate if AI is disabled
	if !aiConfig.Enabled {
		return nil
	}

	// Use AI investigation for unsupported alerts
	return &aiassisted.Investigation{}
}
