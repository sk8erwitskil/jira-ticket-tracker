### Jira Ticket Tracker ###

Continuously searches for new tickets created in a certain jira project
by a certain person and acts upon them.

Currently, it searches for the "reporter" field but you can change that to
"assignee" if you want to track tickets assigned TO a certain user. Just change
the constant ```trackingMethod```.

See example_config.yaml for how to setup your yaml config file.

# Build
```
go build src/jira-ticket-tracker/jira-ticket-tracker.go
```

# Run
```
./jira-ticket-tracker --config=./config.yaml --project=MyTeam --user=jsmith
```
