package goisilon

import (
	"context"
	"fmt"
	log "github.com/akutz/gournal"
	"github.com/dell/goisilon/api/common/utils"
	apiv11 "github.com/dell/goisilon/api/v11"
	"time"
)

const defaultPoll = 5 * time.Second
const defaultTimeout = 10 * time.Minute // set high timeout, we expect to be canceled via context before

const (
	RESYNC_PREP            apiv11.JOB_ACTION              = "resync_prep"
	ALLOW_WRITE            apiv11.JOB_ACTION              = "allow_write"
	ALLOW_WRITE_REVERT     apiv11.JOB_ACTION              = "allow_write_revert"
	TEST                   apiv11.JOB_ACTION              = "test"
	SCHEDULED              apiv11.JOB_STATE               = "scheduled"
	RUNNING                apiv11.JOB_STATE               = "running"
	PAUSED                 apiv11.JOB_STATE               = "paused"
	FINISHED               apiv11.JOB_STATE               = "finished"
	FAILED                 apiv11.JOB_STATE               = "failed"
	CANCELED               apiv11.JOB_STATE               = "canceled"
	NEEDS_ATTENTION        apiv11.JOB_STATE               = "needs_attention"
	SKIPPED                apiv11.JOB_STATE               = "skipped"
	PENDING                apiv11.JOB_STATE               = "pending"
	UNKNOWN                apiv11.JOB_STATE               = "unknown"
	WRITES_DISABLED        apiv11.FAILOVER_FAILBACK_STATE = "writes_disabled"
	ENABLING_WRITES        apiv11.FAILOVER_FAILBACK_STATE = "enabling_writes"
	WRITES_ENABLED         apiv11.FAILOVER_FAILBACK_STATE = "writes_enabled"
	DISABLING_WRITES       apiv11.FAILOVER_FAILBACK_STATE = "disabling_writes"
	CREATING_RESYNC_POLICY apiv11.FAILOVER_FAILBACK_STATE = "creating_resync_policy"
	RESYNC_POLICY_CREATED  apiv11.FAILOVER_FAILBACK_STATE = "resync_policy_created"
)

// Policy is an Isilon Policy
type Policy *apiv11.Policy

type TargetPolicy *apiv11.TargetPolicy


// GetPolicyByName returns a policy with the provided ID.
func (c *Client) GetPolicyByName(ctx context.Context, id string) (Policy, error) {
	return apiv11.GetPolicyByName(ctx, c.API, id)
}

func (c *Client) GetTargetPolicyByName(ctx context.Context, id string) (TargetPolicy, error) {
	return apiv11.GetTargetPolicyByName(ctx, c.API, id)
}

func (c *Client) CreatePolicy(ctx context.Context, name string, rpo int, sourcePath string, targetPath string, targetHost string, enabled bool) error {
	return apiv11.CreatePolicy(ctx, c.API, name, sourcePath, targetPath, targetHost, rpo, enabled)
}

func (c *Client) DeletePolicy(ctx context.Context, name string) error {
	return apiv11.DeletePolicy(ctx, c.API, name)
}

func (c *Client) DeleteTargetPolicy(ctx context.Context, id string) error {
	return apiv11.DeleteTargetPolicy(ctx, c.API, id)
}

func (c *Client) BreakAssociation(ctx context.Context, targetPolicyName string) error {
	tp, err := apiv11.GetTargetPolicyByName(ctx, c.API, targetPolicyName)
	if err != nil {
		return err
	}

	return c.DeleteTargetPolicy(ctx, tp.Id)
}

func (c *Client) ResetPolicy(ctx context.Context, name string) error {
	return apiv11.ResetPolicy(ctx, c.API, name)
}

func (c *Client) EnablePolicy(ctx context.Context, name string) error {
	return c.SetPolicyEnabledField(ctx, name, true)
}

func (c *Client) DisablePolicy(ctx context.Context, name string) error {
	return c.SetPolicyEnabledField(ctx, name, false)
}

func (c *Client) SetPolicyEnabledField(ctx context.Context, name string, value bool) error {
	pp, err := c.GetPolicyByName(ctx, name)
	if err != nil {
		return err
	}
	if pp == nil {
		return nil
	}

	if pp.Enabled == value {
		return nil
	}

	p := &apiv11.Policy{
		Id:      pp.Id,
		Enabled: value,
	}

	return apiv11.UpdatePolicy(ctx, c.API, p)
}

func (c *Client) AllowWrites(ctx context.Context, policyName string) error {
	targetPolicy, err := c.GetTargetPolicyByName(ctx, policyName)
	if err != nil {
		return err
	}
	if targetPolicy.FailoverFailbackState == WRITES_ENABLED {
		return nil
	}

	_, err = c.RunActionForPolicy(ctx, policyName, apiv11.ALLOW_WRITE)
	if err != nil {
		return err
	}

	err = c.WaitForTargetPolicyCondition(ctx, policyName, WRITES_ENABLED)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DisallowWrites(ctx context.Context, policyName string) error {
	targetPolicy, err := c.GetTargetPolicyByName(ctx, policyName)
	if err != nil {
		return err
	}
	if targetPolicy.FailoverFailbackState == WRITES_DISABLED {
		return nil
	}

	_, err = c.RunActionForPolicy(ctx, policyName, apiv11.ALLOW_WRITE_REVERT)
	if err != nil {
		return err
	}

	err = c.WaitForTargetPolicyCondition(ctx, policyName, WRITES_DISABLED)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ResyncPrep(ctx context.Context, policyName string) error {
	targetPolicy, err := c.GetTargetPolicyByName(ctx, policyName)
	if err != nil {
		return err
	}
	if targetPolicy.FailoverFailbackState == RESYNC_POLICY_CREATED {
		return nil
	}

	_, err = c.RunActionForPolicy(ctx, policyName, apiv11.RESYNC_PREP)
	if err != nil {
		return err
	}

	err = c.WaitForTargetPolicyCondition(ctx, policyName, RESYNC_POLICY_CREATED)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) RunActionForPolicy(ctx context.Context, policyName string, action apiv11.JOB_ACTION) (*apiv11.JobRequest, error) {
	job := &apiv11.JobRequest{
		Id:     policyName,
		Action: action,
	}

	return apiv11.StartSyncIQJob(ctx, c.API, job)
}

func (c *Client) StartSyncIQJob(ctx context.Context, job *apiv11.JobRequest) (*apiv11.JobRequest, error) {
	return apiv11.StartSyncIQJob(ctx, c.API, job)
}

func (c *Client) GetReport(ctx context.Context, reportName string) (*apiv11.Report, error) {
	return apiv11.GetReport(ctx, c.API, reportName)
}

func (c *Client) GetReportsByPolicyName(ctx context.Context, policyName string, reportsForPolicy int) (*apiv11.Reports, error) {
	return apiv11.GetReportsByPolicyName(ctx, c.API, policyName, reportsForPolicy)
}

func (c *Client) WaitForPolicyEnabledFieldCondition(ctx context.Context, policyName string, enabled bool) error {
	pollErr := utils.PollImmediateWithContext(ctx, defaultPoll, defaultTimeout,
		func(iCtx context.Context) (bool, error) {
			p, err := c.GetPolicyByName(iCtx, policyName)
			if err != nil {
				return false, err
			}

			if p.Enabled != enabled {
				return false, nil
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}

	return nil
}

func (c *Client) WaitForPolicyLastJobState(ctx context.Context, policyName string, state apiv11.JOB_STATE) error {
	pollErr := utils.PollImmediateWithContext(ctx, defaultPoll, defaultTimeout,
		func(iCtx context.Context) (bool, error) {
			p, err := c.GetPolicyByName(iCtx, policyName)
			if err != nil {
				return false, err
			}

			if p.LastJobState != state {
				return false, nil
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}

	return nil
}

func (c *Client) WaitForTargetPolicyCondition(ctx context.Context, policyName string, condition apiv11.FAILOVER_FAILBACK_STATE) error {
	pollErr := utils.PollImmediateWithContext(ctx, defaultPoll, defaultTimeout,
		func(iCtx context.Context) (bool, error) {
			tp, err := c.GetTargetPolicyByName(iCtx, policyName)
			if err != nil {
				return false, err
			}

			if tp.FailoverFailbackState != condition {
				return false, nil
			}

			return true, nil
		})

	if pollErr != nil {
		return pollErr
	}

	return nil
}

func (c *Client) SyncPolicy(ctx context.Context, policyName string) error {
	p, err := c.GetPolicyByName(ctx, policyName)
	if err != nil {
		return err
	}
	rpo := p.JobDelay

	reportFilter := func(r apiv11.Report) bool {
		// Check if report is for sync action and is recent enough
		isSync := r.Policy.Action == "sync"

		endTime := time.Unix(r.EndTime, 0)
		diff := time.Now().Sub(endTime).Seconds()

		fmt.Println("end time", endTime.String())

		fmt.Println("diff", diff)

		isRecent := diff < float64(rpo) / 2

		isFinished := r.State == apiv11.FINISHED

		return isSync && isRecent && isFinished
	}

	reports, err := c.GetReportsByPolicyName(ctx, policyName, 5)
	if err != nil {
		return err
	}

	filtered := FilterReports(reports.Reports, reportFilter)
	log.Debug(ctx, "filtered reports %v for policy %s", filtered, policyName)

	if len(filtered) == 0 {
		// If no reports found matching to our criteria -- run sync job
		log.Info(ctx, "No matching reports were found, starting sync job")
		jobReq := &apiv11.JobRequest{
			Id: policyName,
		}
		_, err := c.StartSyncIQJob(ctx, jobReq)
		if err != nil {
			return err
		}

		log.Info(ctx, "Waiting for SyncIQ job to complete")
		pollErr := utils.PollImmediateWithContext(ctx, defaultPoll, defaultTimeout,
			func(iCtx context.Context) (bool, error) {
				reports, err := c.GetReportsByPolicyName(ctx, policyName, 5)
				if err != nil {
					return false, err
				}

				filtered := FilterReports(reports.Reports, reportFilter)

				if len(filtered) == 0 {
					return false, nil
				}

				return true, nil
			})
		if pollErr != nil {
			return pollErr
		}
	} else {
		log.Info(ctx, "Matching reports for policy %s were already found", policyName)
	}

	return nil
}

func FilterReports(values []apiv11.Report, filterFunc func(apiv11.Report) bool) []apiv11.Report {
	filtered := make([]apiv11.Report, 0)
	for _, v := range values {
		if filterFunc(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
