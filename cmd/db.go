package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gappylul/deployit/internal/deploy"
	"github.com/gappylul/deployit/internal/provision"
	"github.com/spf13/cobra"
)

var restoreFile string
var dbType string

var backupCmd = &cobra.Command{
	Use:   "backup <app-name>",
	Short: "Backup database (Postgres or Redis) to a local file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		ctx := context.Background()

		config, err := deploy.GetConfig()
		if err != nil {
			return err
		}
		clientset, err := deploy.GetClientset()
		if err != nil {
			return err
		}

		if dbType == "redis" {
			filename := fmt.Sprintf("%s_backup.rdb", appName)
			f, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer f.Close()

			fmt.Printf("-> Pulling Redis RDB snapshot to %s...\n", filename)
			return provision.BackupRedis(ctx, config, clientset, appName, f)
		}

		filename := fmt.Sprintf("%s_backup.sql", appName)
		f, err := os.Create(filename)
		if err != nil {
			return err
		}
		defer f.Close()

		fmt.Printf("-> Pulling Postgres backup to %s...\n", filename)
		return provision.BackupPostgres(ctx, config, clientset, appName, f)
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <app-name>",
	Short: "Restore a local file into a database",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		if restoreFile == "" {
			return fmt.Errorf("usage: deployit restore %s --file <path>", appName)
		}

		ctx := context.Background()
		config, err := deploy.GetConfig()
		if err != nil {
			return err
		}
		clientset, err := deploy.GetClientset()
		if err != nil {
			return err
		}

		f, err := os.Open(restoreFile)
		if err != nil {
			return err
		}
		defer f.Close()

		if dbType == "redis" {
			deploymentName := fmt.Sprintf("redis-%s", appName)

			fmt.Println("-> Scaling down Redis to release volume...")
			if err := deploy.ScaleDeployment(ctx, clientset, deploymentName, 0); err != nil {
				return fmt.Errorf("failed to scale down: %w", err)
			}

			fmt.Println("-> Waiting for pods to terminate...")
			time.Sleep(5 * time.Second)

			err = provision.RestoreRedis(ctx, config, clientset, appName, f)

			fmt.Println("-> Scaling Redis back up...")
			scaleErr := deploy.ScaleDeployment(ctx, clientset, deploymentName, 1)

			if err != nil {
				return err
			}
			return scaleErr
		}

		fmt.Printf("-> Pushing %s into postgres-%s...\n", restoreFile, appName)
		return provision.RestorePostgres(ctx, config, clientset, appName, f)
	},
}

func init() {
	backupCmd.Flags().StringVarP(&dbType, "type", "t", "postgres", "Database type (postgres or redis)")
	restoreCmd.Flags().StringVarP(&dbType, "type", "t", "postgres", "Database type (postgres or redis)")

	restoreCmd.Flags().StringVarP(&restoreFile, "file", "f", "", "The file to upload")
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
