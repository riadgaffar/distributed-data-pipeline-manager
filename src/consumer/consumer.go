package consumer

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

/*
	TODO: Currently unused. Remove/Update.

Might Still Need for the followings:

1.	Custom Logic Not Supported by the Manager:
  - If the application requires logic that can’t be achieved through the YAML configuration (e.g., complex error handling, dynamic topic selection, non-standard message transformations).

2.	Standalone Consumers:
  - If the plan is to use consumer.go outside the pipeline manager, as a standalone Kafka consumer (e.g., for debugging or prototyping).

3.	Transition Period:
  - If migrating from custom consumers to configuration-driven pipelines, you may temporarily keep consumer.go.

4. Others, TBD
*/

// ConsumeMessages connects to Kafka, transforms messages, and writes them to Postgres
func ConsumeMessages(
	ctx context.Context,
	consumer KafkaConsumer,
	db *sql.DB,
	table string,
	columns map[string]string,
) error {
	defer consumer.Close()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Consumer shutting down...")
			return nil
		default:
			msg, err := consumer.ReadMessage(-1)
			if err != nil {
				// Log the error but exit the loop if no more messages are available
				if err.Error() == "no more messages" {
					fmt.Println("No more messages. Stopping consumer.")
					return nil
				}
				fmt.Printf("Consumer error: %v\n", err)
				continue
			}

			fmt.Printf("Received message: %s\n", string(msg.Value))

			// Transform the message
			transformed := map[string]interface{}{
				"id":        uuid.New().String(),
				"timestamp": time.Now().Format(time.RFC3339),
				"data":      strings.ToUpper(string(msg.Value)),
			}

			// Construct the query dynamically
			columnsList := []string{}
			placeholders := []string{}
			values := []interface{}{}
			i := 1
			for field, column := range columns {
				columnsList = append(columnsList, column)
				placeholders = append(placeholders, fmt.Sprintf("$%d", i))
				values = append(values, transformed[field])
				i++
			}

			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columnsList, ", "), strings.Join(placeholders, ", "))
			_, err = db.Exec(query, values...)
			if err != nil {
				fmt.Printf("ERROR: Failed to write to Postgres: %v\n", err)
				continue
			}
			fmt.Println("DEBUG: Message written to Postgres.")
		}
	}
}
