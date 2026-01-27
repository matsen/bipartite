package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/flow"
	"github.com/matsen/bipartite/internal/store"
	"github.com/spf13/cobra"
)

var (
	slackIngestDays        int
	slackIngestSince       string
	slackIngestLimit       int
	slackIngestStore       string
	slackIngestCreateStore bool
)

var slackIngestCmd = &cobra.Command{
	Use:   "ingest <channel>",
	Short: "Ingest Slack messages into a store",
	Long: `Ingest messages from a Slack channel directly into a store.

Combines slack history fetching with store append in a single command.
Messages are stored with a composite id (channel:timestamp) to ensure
uniqueness across channels. Duplicate messages are skipped (idempotent).

The target store must exist unless --create-store is specified.

Examples:
  # Ingest last 30 days of fortnight-goals into the slack_msgs store
  bip slack ingest fortnight-goals --store slack_msgs --days 30

  # Create store if it doesn't exist
  bip slack ingest fortnight-goals --store slack_msgs --create-store

  # Ingest since a specific date
  bip slack ingest fortnight-feats --store slack_msgs --since 2026-01-01`,
	Args: cobra.ExactArgs(1),
	RunE: runSlackIngest,
}

func init() {
	slackCmd.AddCommand(slackIngestCmd)
	slackIngestCmd.Flags().IntVar(&slackIngestDays, "days", 14, "Number of days to fetch")
	slackIngestCmd.Flags().StringVar(&slackIngestSince, "since", "", "Start date (YYYY-MM-DD), overrides --days")
	slackIngestCmd.Flags().IntVar(&slackIngestLimit, "limit", 100, "Maximum messages to fetch")
	slackIngestCmd.Flags().StringVar(&slackIngestStore, "store", "", "Target store name (required)")
	slackIngestCmd.Flags().BoolVar(&slackIngestCreateStore, "create-store", false, "Create the store if it doesn't exist")
	slackIngestCmd.MarkFlagRequired("store")
}

// SlackIngestResult is the JSON output for the ingest command.
type SlackIngestResult struct {
	Channel      string `json:"channel"`
	Store        string `json:"store"`
	Ingested     int    `json:"ingested"`
	Skipped      int    `json:"skipped"`
	StoreCreated bool   `json:"store_created,omitempty"`
}

func runSlackIngest(cmd *cobra.Command, args []string) error {
	channelName := args[0]
	repoRoot := mustFindRepository()

	// Get channel configuration
	channelConfig, err := flow.GetSlackChannel(channelName)
	if err != nil {
		return outputSlackError(ExitSlackChannelNotFound, "channel_not_found", err.Error())
	}

	// Create Slack client
	client, err := flow.NewSlackClient()
	if err != nil {
		return outputSlackError(ExitSlackMissingToken, "missing_token", err.Error())
	}

	// Load user cache
	if _, err := client.GetUsers(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load users: %v\n", err)
	}

	// Open or create store
	s, storeCreated, err := openOrCreateSlackStore(repoRoot, slackIngestStore)
	if err != nil {
		exitWithError(ExitError, "%v", err)
	}

	// Calculate time range
	var oldest time.Time
	if slackIngestSince != "" {
		t, err := time.Parse("2006-01-02", slackIngestSince)
		if err != nil {
			return outputSlackError(1, "invalid_date", fmt.Sprintf("invalid date format '%s'; use YYYY-MM-DD", slackIngestSince))
		}
		oldest = t
	} else {
		oldest = time.Now().AddDate(0, 0, -slackIngestDays)
	}

	// Fetch messages
	messages, err := client.GetChannelHistory(channelConfig.ID, oldest, slackIngestLimit)
	if err != nil {
		if strings.Contains(err.Error(), "not_in_channel") {
			return outputSlackError(ExitSlackNotMember, "not_member",
				fmt.Sprintf("Bot is not a member of channel '%s'. Invite the bot with /invite @bot-name", channelName))
		}
		return outputSlackError(1, "api_error", err.Error())
	}

	// Ingest messages into store
	ingested := 0
	skipped := 0

	for _, msg := range messages {
		record := messageToRecord(channelName, msg)
		err := s.Append(record)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate primary key") {
				skipped++
				continue
			}
			exitWithError(ExitError, "appending record: %v", err)
		}
		ingested++
	}

	// Output result
	result := SlackIngestResult{
		Channel:      channelName,
		Store:        slackIngestStore,
		Ingested:     ingested,
		Skipped:      skipped,
		StoreCreated: storeCreated,
	}

	if humanOutput {
		if storeCreated {
			fmt.Printf("Created store '%s'\n", slackIngestStore)
		}
		if skipped > 0 {
			fmt.Printf("Ingested %d messages into '%s' (%d duplicates skipped)\n", ingested, slackIngestStore, skipped)
		} else {
			fmt.Printf("Ingested %d messages into '%s'\n", ingested, slackIngestStore)
		}
	} else {
		outputJSON(result)
	}

	return nil
}

// messageToRecord converts a Slack message to a store record.
func messageToRecord(channel string, msg flow.Message) store.Record {
	return store.Record{
		"id":      fmt.Sprintf("%s:%s", channel, msg.Timestamp),
		"channel": channel,
		"user":    msg.UserName,
		"date":    msg.Date,
		"text":    msg.Text,
	}
}

// openOrCreateSlackStore opens an existing store or creates one if --create-store is set.
// Returns the store, whether it was created, and any error.
func openOrCreateSlackStore(repoRoot, storeName string) (*store.Store, bool, error) {
	// Try to open existing store
	s, err := store.OpenStore(repoRoot, storeName)
	if err == nil {
		return s, false, nil
	}

	// Store doesn't exist
	if !slackIngestCreateStore {
		return nil, false, fmt.Errorf("store %q not found. Create it with:\n  bip slack ingest <channel> --store %s --create-store", storeName, storeName)
	}

	// Create the store with predefined schema
	schema := slackMessageSchema(storeName)

	// Save schema to file
	schemaDir := filepath.Join(repoRoot, ".bipartite", "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		return nil, false, fmt.Errorf("creating schema directory: %w", err)
	}

	schemaPath := filepath.Join(schemaDir, storeName+".json")
	if err := writeSchemaFile(schemaPath, schema); err != nil {
		return nil, false, fmt.Errorf("writing schema file: %w", err)
	}

	// Create store directory
	storeDir := filepath.Join(repoRoot, ".bipartite")

	// Create and initialize store
	s = store.NewStore(storeName, schema, storeDir, schemaPath)
	if err := s.Init(repoRoot); err != nil {
		return nil, false, fmt.Errorf("initializing store: %w", err)
	}

	return s, true, nil
}

// slackMessageSchema returns the predefined schema for Slack messages.
func slackMessageSchema(name string) *store.Schema {
	return &store.Schema{
		Name: name,
		Fields: map[string]*store.Field{
			"id":      {Type: store.FieldTypeString, Primary: true},
			"channel": {Type: store.FieldTypeString, Index: true},
			"user":    {Type: store.FieldTypeString, Index: true},
			"date":    {Type: store.FieldTypeDate, Index: true},
			"text":    {Type: store.FieldTypeString, FTS: true},
		},
	}
}

// writeSchemaFile writes a schema to a JSON file.
func writeSchemaFile(path string, schema *store.Schema) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write JSON manually for nice formatting
	fmt.Fprintf(f, "{\n")
	fmt.Fprintf(f, "  \"name\": %q,\n", schema.Name)
	fmt.Fprintf(f, "  \"fields\": {\n")

	// Write fields in a consistent order
	fieldOrder := []string{"id", "channel", "user", "date", "text"}
	for i, name := range fieldOrder {
		field := schema.Fields[name]
		fmt.Fprintf(f, "    %q: {\"type\": %q", name, field.Type)
		if field.Primary {
			fmt.Fprintf(f, ", \"primary\": true")
		}
		if field.Index {
			fmt.Fprintf(f, ", \"index\": true")
		}
		if field.FTS {
			fmt.Fprintf(f, ", \"fts\": true")
		}
		fmt.Fprintf(f, "}")
		if i < len(fieldOrder)-1 {
			fmt.Fprintf(f, ",")
		}
		fmt.Fprintf(f, "\n")
	}

	fmt.Fprintf(f, "  }\n")
	fmt.Fprintf(f, "}\n")

	return nil
}
