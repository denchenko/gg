package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/denchenko/gg/internal/config"
	"github.com/denchenko/gg/internal/core/app"
	"github.com/denchenko/gg/internal/core/domain"
	ascii "github.com/denchenko/gg/internal/format/ascii"
	"github.com/denchenko/gg/internal/log"
	"github.com/spf13/cobra"
)

func My(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "my",
		Short: "Everything related to you",
	}

	cmd.AddCommand(
		MyMR(cfg, appInstance, formatter),
		MyReview(cfg, appInstance, formatter),
		MyActivity(cfg, appInstance, formatter),
	)

	return cmd
}

func MyMR(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	return &cobra.Command{
		Use:   "mr",
		Short: "Show your merge requests status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return showMyMRStatus(cfg, appInstance, formatter)
		},
	}
}

func MyReview(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Show your review workload",
		RunE: func(_ *cobra.Command, _ []string) error {
			return showMyReviewWorkload(cfg, appInstance, formatter)
		},
	}
}

func showMyReviewWorkload(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) error {
	ctx := context.Background()

	var mrsWithStatus []*domain.MergeRequestWithStatus
	err := log.WithSpinner("Fetching your review workload...", func() error {
		mergeRequestsWithStatus, err := appInstance.GetMyReviewWorkloadWithStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get review workload: %w", err)
		}

		mrsWithStatus = mergeRequestsWithStatus

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get review workload: %w", err)
	}

	formatted, err := formatter.FormatMyReviewWorkload(cfg.BaseURL, mrsWithStatus)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	return nil
}

func showMyMRStatus(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) error {
	ctx := context.Background()

	var mrsWithStatus []*domain.MergeRequestWithStatus
	err := log.WithSpinner("Fetching your merge requests...", func() error {
		mergeRequestsWithStatus, err := appInstance.GetMergeRequestsWithStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get merge requests: %w", err)
		}

		mrsWithStatus = mergeRequestsWithStatus

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get merge requests: %w", err)
	}

	formatted, err := formatter.FormatMyMergeRequestStatus(cfg.BaseURL, mrsWithStatus)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	return nil
}

// ActivityDateRange holds the date ranges for querying activity events.
type ActivityDateRange struct {
	After  time.Time
	Before *time.Time
}

func MyActivity(cfg *config.Config, appInstance *app.App, formatter *ascii.Formatter) *cobra.Command {
	var afterStr, beforeStr string

	cmd := &cobra.Command{
		Use:   "activity",
		Short: "Show your activity",
		RunE: func(_ *cobra.Command, _ []string) error {
			return showMyActivity(cfg, appInstance, formatter, afterStr, beforeStr)
		},
	}

	cmd.Flags().StringVar(&afterStr, "after", "",
		"Activities after this date (ISO 8601 format, defaults to day before last working day)")
	cmd.Flags().StringVar(&beforeStr, "before", "", "Activities before this date (ISO 8601 format, optional)")

	return cmd
}

func showMyActivity(
	cfg *config.Config,
	appInstance *app.App,
	formatter *ascii.Formatter,
	afterStr, beforeStr string,
) error {
	ctx := context.Background()

	dateRange, err := parseActivityDates(afterStr, beforeStr, time.Now())
	if err != nil {
		return fmt.Errorf("failed to parse dates: %w", err)
	}

	var events []*domain.Event
	err = log.WithSpinner("Fetching your activity...", func() error {
		activityEvents, err := appInstance.GetMyActivity(ctx, dateRange.After, dateRange.Before)
		if err != nil {
			return fmt.Errorf("failed to get activity: %w", err)
		}

		events = activityEvents

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to get activity: %w", err)
	}

	formatted, err := formatter.FormatMyActivity(cfg.BaseURL, events)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	fmt.Print(formatted)

	return nil
}

const iso8601DateOnlyFormat = "2006-01-02"

func parseActivityDates(afterStr, beforeStr string, now time.Time) (ActivityDateRange, error) {
	var (
		after     time.Time
		before    *time.Time
		err       error
		dateRange ActivityDateRange
	)

	if afterStr == "" {
		// Default: set after to day before last working day
		after = calculateDefaultAfter(now)
		dateRange.After = after
	} else {
		after, err = time.Parse(iso8601DateOnlyFormat, afterStr)
		if err != nil {
			return dateRange, fmt.Errorf("invalid --after date: %w", err)
		}
		dateRange.After = after
	}

	if beforeStr != "" {
		parsedBefore, err := time.Parse(iso8601DateOnlyFormat, beforeStr)
		if err != nil {
			return dateRange, fmt.Errorf("invalid --before date: %w", err)
		}
		before = &parsedBefore
		dateRange.Before = before

		// Validate that after is before before
		if after.After(*before) {
			return dateRange, errors.New("--after date must be before --before date")
		}
	}

	return dateRange, nil
}

// calculateDefaultAfter calculates the default 'after' date based on working days.
// Working days are Monday (1) through Friday (5).
// If run on a working day, it returns the day before the previous working day.
// If run on Saturday/Sunday, it returns Thursday (day before Friday).
func calculateDefaultAfter(now time.Time) time.Time {
	weekday := int(now.Weekday())
	// time.Weekday: Sunday=0, Monday=1, ..., Saturday=6

	var lastWorkingDay time.Time
	if weekday >= 1 && weekday <= 5 {
		// Today is a working day (Mon-Fri)
		// Find the previous working day
		if weekday == 1 {
			// Monday: previous working day is Friday (3 days ago)
			lastWorkingDay = now.AddDate(0, 0, -3)
		} else {
			// Tuesday-Friday: previous working day is yesterday
			lastWorkingDay = now.AddDate(0, 0, -1)
		}
	} else {
		// Today is Saturday or Sunday
		// Last working day is Friday
		if now.Weekday() == time.Saturday {
			// Saturday: Friday is yesterday
			lastWorkingDay = now.AddDate(0, 0, -1)
		} else {
			// Sunday: Friday is 2 days ago
			lastWorkingDay = now.AddDate(0, 0, -2)
		}
	}

	// Return the day before the last working day (start of that day at 00:00:00)
	dayBefore := lastWorkingDay.AddDate(0, 0, -1)

	return time.Date(dayBefore.Year(), dayBefore.Month(), dayBefore.Day(), 0, 0, 0, 0, dayBefore.Location())
}
