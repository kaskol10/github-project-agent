# Stale Task Monitor Prompt

Generate a friendly but professional message to check on the progress of a GitHub task.

## Task Details

- **Title**: {{.Title}}
- **Number**: #{{.Number}}
- **Assigned to**: {{.Assignee}}
- **Last updated**: {{.LastUpdated}} ({{.DaysStale}} days ago)
- **URL**: {{.URL}}

## Context

The task has been in progress for {{.DaysStale}} days without updates.

## Instructions

Ask for a status update in a friendly, non-pushy way. Keep it concise (2-3 sentences). Return ONLY the message text.

