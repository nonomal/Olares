package os

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Masterminds/semver/v3"
	"github.com/beclab/Olares/cli/cmd/ctl/options"
	"github.com/beclab/Olares/cli/pkg/phase"
	"github.com/beclab/Olares/cli/pkg/pipelines"
	"github.com/beclab/Olares/cli/pkg/upgrade"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type UpgradeOsOptions struct {
	UpgradeOptions *options.UpgradeOptions
}

func NewUpgradeOsOptions() *UpgradeOsOptions {
	return &UpgradeOsOptions{
		UpgradeOptions: options.NewUpgradeOptions(),
	}
}

func NewCmdUpgradeOs() *cobra.Command {
	o := NewUpgradeOsOptions()
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Olares to a newer version",
		Run: func(cmd *cobra.Command, args []string) {
			if err := pipelines.UpgradeOlaresPipeline(o.UpgradeOptions); err != nil {
				log.Fatalf("error: %v", err)
			}
		},
	}
	o.UpgradeOptions.AddFlags(cmd)
	cmd.AddCommand(NewCmdUpgradePrecheck())
	cmd.AddCommand(NewCmdGetUpgradePath())
	return cmd
}

func NewCmdGetUpgradePath() *cobra.Command {
	var baseVersionStr string
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Get the upgrade path (required intermediate versions) from base version to the latest upgradable version (as known to this release of olares-cli)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var baseVersion *semver.Version
			var err error
			if baseVersionStr == "" {
				baseVersionStr, err = phase.GetOlaresVersion()
				if err != nil {
					return errors.New("failed to get current Olares version, please specify the base version explicitly")
				}
			}
			baseVersion, err = semver.NewVersion(baseVersionStr)
			if err != nil {
				return fmt.Errorf("invalid base version: %v", err)
			}

			path, err := upgrade.GetUpgradePathFor(baseVersion, nil)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(path)
		},
	}

	cmd.Flags().StringVarP(&baseVersionStr, "base-version", "b", baseVersionStr, "base version to be upgraded, defaults to the current Olares version if inside Olares cluster")

	return cmd
}

func NewCmdUpgradePrecheck() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "precheck",
		Short: "Precheck Olares for Upgrade",
		Run: func(cmd *cobra.Command, args []string) {
			if err := pipelines.UpgradePreCheckPipeline(); err != nil {
				log.Fatalf("error: %v", err)
			}
		},
	}
	return cmd
}

func NewCmdMinVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "min-version",
		Short: "Get the minimum version of Olares that this CLI can upgrade with",
		RunE: func(cmd *cobra.Command, args []string) error {
			minVersion, err := upgrade.GetMinVersion()
			if err != nil {
				return err
			}

			// output version.hint
			versionHint := map[string]interface{}{
				"upgrade": map[string]interface{}{
					"minVersion": minVersion,
				},
			}

			encoder := yaml.NewEncoder(cmd.OutOrStdout())
			return encoder.Encode(versionHint)
		},
	}
	return cmd
}
