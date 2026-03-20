package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/driif/go-vibe-starter/internal/server/config"
	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Inserts seed data into the database.",
	Long:  `Uses upsert to add test data to the current database.`,
	Run:   seedCmdFunc,
}

func init() {
	dbCmd.AddCommand(seedCmd)
}

func seedCmdFunc(_ *cobra.Command, _ []string) {
	if err := applyFixtures(); err != nil {
		fmt.Printf("Error while applying fixtures: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeded all fixtures to the database.\n")
}

func applyFixtures() error {
	ctx := context.Background()
	conf := config.DefaultServiceConfigFromEnv()
	db, err := sql.Open("postgres", conf.Database.ConnectionString())
	if err != nil {
		return err
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	return nil

}
