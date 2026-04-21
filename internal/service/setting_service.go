package service

import (
	"context"
	"encoding/json"

	"lumina/internal/database"
)

// GetAllSettings returns all settings for userID as a key→value map.
// Missing users return an empty map (frontend applies its own defaults).
func GetAllSettings(ctx context.Context, userID int) (map[string]any, error) {
	rows, err := database.Pool.Query(ctx,
		`SELECT key, value FROM user_settings WHERE user_id = $1 ORDER BY key`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]any)
	for rows.Next() {
		var key string
		var valueJSON []byte
		if err := rows.Scan(&key, &valueJSON); err != nil {
			return nil, err
		}

		var value any
		if err := json.Unmarshal(valueJSON, &value); err != nil {
			settings[key] = string(valueJSON)
		} else {
			settings[key] = value
		}
	}

	return settings, nil
}

// UpdateSettings batch upserts multiple settings for userID.
func UpdateSettings(ctx context.Context, userID int, settings map[string]any) error {
	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for key, value := range settings {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		// idx_user_settings_user_key enforces (user_id, key) uniqueness.
		_, err = tx.Exec(ctx,
			`INSERT INTO user_settings (user_id, key, value, updated_at)
			 VALUES ($1, $2, $3, NOW())
			 ON CONFLICT (user_id, key) DO UPDATE SET
			   value      = EXCLUDED.value,
			   updated_at = NOW()`,
			userID, key, valueJSON,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
