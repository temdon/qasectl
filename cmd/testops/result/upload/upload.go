package upload

import (
	"fmt"
	"github.com/qase-tms/qasectl/cmd/flags"
	"github.com/qase-tms/qasectl/internal/client"
	"github.com/qase-tms/qasectl/internal/parsers/allure"
	"github.com/qase-tms/qasectl/internal/parsers/junit"
	"github.com/qase-tms/qasectl/internal/parsers/qase"
	"github.com/qase-tms/qasectl/internal/parsers/xctest"
	"github.com/qase-tms/qasectl/internal/service/result"
	"github.com/qase-tms/qasectl/internal/service/run"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log/slog"
)

const (
	pathFlag        = "path"
	formatFlag      = "format"
	runIDFlag       = "id"
	titleFlag       = "title"
	descriptionFlag = "description"
)

// Command returns a new cobra command for upload
func Command() *cobra.Command {
	var (
		path        string
		format      string
		runID       int64
		title       string
		description string
		steps       string
		batch       int64
		suite       string
	)

	cmd := &cobra.Command{
		Use:     "upload",
		Short:   "Upload test results",
		Example: "qli testops result upload --path 'path' --format 'junit' --id 123 --project 'PRJ' --token 'TOKEN'",
		RunE: func(cmd *cobra.Command, args []string) error {
			const op = "upload"
			logger := slog.With("op", op)

			token := viper.GetString(flags.TokenFlag)
			project := viper.GetString(flags.ProjectFlag)

			var p result.Parser
			switch format {
			case "junit":
				p = junit.NewParser(path)
			case "qase":
				p = qase.NewParser(path)
			case "allure":
				p = allure.NewParser(path)
			case "xctest":
				prs, err := xctest.NewParser(path, steps)
				if err != nil {
					return err
				}

				p = prs
			default:
				return fmt.Errorf("unknown format: %s. allowed formats: junit, qase, allure, xctest", format)
			}

			cv1 := client.NewClientV1(token)
			cv2 := client.NewClientV2(token, cv1)
			rs := run.NewService(cv1)
			s := result.NewService(cv2, p, rs)

			param := result.UploadParams{
				RunID:       runID,
				Title:       title,
				Description: description,
				Batch:       batch,
				Project:     project,
				Suite:       suite,
			}

			err := s.Upload(cmd.Context(), param)
			if err != nil {
				return err
			}

			logger.Info("Results uploaded successfully")

			return nil
		},
	}

	cmd.Flags().StringVar(&path, pathFlag, "", "path to the results file")
	err := cmd.MarkFlagRequired(pathFlag)
	if err != nil {
		slog.Error("Error while marking flag as required", "error", err)
	}

	cmd.Flags().StringVar(&format, formatFlag, "", "format of the results file: junit, qase, allure, xctest")
	err = cmd.MarkFlagRequired(formatFlag)
	if err != nil {
		slog.Error("Error while marking flag as required", "error", err)
	}

	cmd.Flags().Int64Var(&runID, runIDFlag, 0, "ID of the test run")
	cmd.Flags().StringVar(&title, titleFlag, "", "Title of the test run")
	cmd.Flags().StringVarP(&description, descriptionFlag, "d", "", "Description of the test run")
	cmd.MarkFlagsOneRequired(runIDFlag, titleFlag)
	cmd.MarkFlagsMutuallyExclusive(runIDFlag, titleFlag)

	cmd.Flags().StringVar(&steps, "steps", "", "Steps show mode in XCTest. Allowed values: all, user")
	cmd.Flags().Int64VarP(&batch, "batch", "b", 200, "Batch size for uploading results")
	cmd.Flags().StringVarP(&suite, "suite", "s", "", "Root suite for the results")

	return cmd
}
