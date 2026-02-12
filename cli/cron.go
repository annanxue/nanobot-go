package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List scheduled tasks",
	RunE:  runCronList,
}

var cronApp = &cobra.Command{
	Use:   "cron",
	Short: "Manage scheduled tasks",
}

func init() {
	cronApp.AddCommand(cronListCmd)
	rootCmd.AddCommand(cronApp)
}

func runCronList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cronService, err := createCronService(cfg)
	if err != nil {
		return fmt.Errorf("failed to create cron service: %w", err)
	}

	status := cronService.Status()
	jobsCount := 0
	if count, ok := status["jobs"].(int); ok {
		jobsCount = count
	}
	fmt.Printf("Scheduled jobs: %d\n", jobsCount)

	jobs := cronService.ListJobs(false)
	for _, job := range jobs {
		fmt.Printf("- %s (id: %s)\n", job.Name, job.ID)
	}

	return nil
}
