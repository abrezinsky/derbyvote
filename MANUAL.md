# DerbyVote - Administrator Manual

Guide for event organizers and administrators.

## Table of Contents

1. [Introduction](#introduction)
2. [Initial Setup](#initial-setup)
3. [Event Preparation](#event-preparation)
4. [Running an Event](#running-an-event)
5. [Results and Reporting](#results-and-reporting)
6. [Configuration](#configuration)
7. [Troubleshooting](#troubleshooting)

---

## Introduction

DerbyVote is a web-based voting system for Pinewood Derby recognition awards. This manual covers event setup, operation, and result management.

### System Access

- Admin interface: `http://[server]:8081/admin`
- Voter interface: `http://[server]:8081/` or `http://[server]:8081/vote?qr=[CODE]`
- Authentication: Password-based (generated at startup or set via `-adminpw` flag)

---

## Initial Setup

### Starting the Server

```bash
./derbyvote -port 8081 -db voting.db -adminpw [password]
```

The server displays the admin password and network address on startup. The admin interface is accessible at the displayed URL.

### Importing Data from DerbyNet

If using DerbyNet:

1. Navigate to Admin → Settings
2. Enter DerbyNet URL and credentials
3. Click "Sync Cars from DerbyNet" to import racer roster
4. Click "Sync Categories from DerbyNet" to import existing awards (optional)

Cars are matched by car number. Re-syncing updates existing entries.

### Adding Data Manually

Without DerbyNet:

1. Navigate to Admin → Cars → Add Car
2. Enter car number, racer name, and car name
3. Repeat for all entries

---

## Event Preparation

### Configuring Categories

Categories represent the awards voters will select winners for.

**Creating Categories**:
1. Navigate to Admin → Categories
2. Click "Create Category"
3. Enter category name and display order
4. Optionally assign to a category group

**Category Groups**:

Groups organize related categories and can enforce exclusivity rules. Create a group to prevent voters from selecting the same car for multiple awards within that group.

1. Navigate to Admin → Category Groups
2. Click "Create Group"
3. Set an Exclusivity Pool ID to link related categories
4. Assign categories to the group

When a voter selects a car in one category within an exclusive group, the system prompts for confirmation if they attempt to select the same car in another category in that group.

### Voter Registration

Two modes are available:

**Registered Mode** (default):
- Pre-generate specific QR codes for controlled access
- Navigate to Admin → Voters → Generate QR Codes
- Enter the number of codes needed
- Print the generated page
- Distribute codes to voters

**Open Mode**:
- Navigate to Admin → Settings
- Disable "Require Registered QR Codes"
- A universal QR code becomes available
- Voters can use any code at the URL: `http://[server]:8081/vote?qr=[ANY-CODE]`

---

## Running an Event

### Opening Voting

1. Navigate to Admin → Settings or Dashboard
2. Enable "Voting Open"
3. Optionally set a countdown timer (1-60 minutes)
4. Distribute QR codes or voting URL to participants

### Monitoring Activity

**Dashboard**: Real-time statistics
- Total registered voters
- Active voters (those who have cast votes)
- Total votes cast
- Votes per category

**Results Page**: Live vote counts per category

### Voter Experience

Voters access their ballot by scanning a QR code or entering a code at the landing page. The interface presents all categories as horizontal tabs. Selecting a car saves the vote immediately. Voters can change their selection at any time before voting closes.

The system provides visual feedback:
- Selected cars are highlighted
- Category tabs show completion status
- Progress bar indicates overall completion
- Warnings appear for exclusivity conflicts

### Closing Voting

1. Navigate to Admin → Settings or Dashboard
2. Click "Close Voting"

All active voter sessions receive immediate notification via WebSocket. Votes are locked and the system transitions to result mode.

---

## Results and Reporting

### Viewing Results

Navigate to Admin → Results to view vote tallies. Results display:
- Vote counts per car per category
- Leading car(s) for each category
- Tie notifications
- Conflict indicators

### Resolving Ties

Ties are highlighted in the results view. To resolve:

1. Locate the tied category
2. Click "Set Manual Winner"
3. Select the winning car
4. Enter a resolution reason (required)
5. Confirm the override

Manual overrides are marked with a timestamp and display the reason entered.

### Exporting to DerbyNet

Prerequisites:
- Categories must be linked to DerbyNet award IDs
- DerbyNet connection must be configured

To export results:

1. Navigate to Admin → Results
2. Click "Push Results to DerbyNet"
3. Review the status report

The system reports success, errors, and skipped categories (those without DerbyNet mappings).

---

## Configuration

### Voting Control

**Timer Settings**:
- Quick presets: 1, 5, 10, 15 minutes
- Custom duration: 1-60 minutes
- Timer appears in voter interface with countdown
- Voting closes automatically when timer expires

**Voting Status**:
- Toggle between open and closed states
- Status synchronizes to all active voter sessions in real-time

### Security Settings

**Require Registered QR Codes**:
- Enabled: Only pre-generated codes are accepted
- Disabled: Any code auto-creates a voter session

### DerbyNet Configuration

**Connection Settings**:
- DerbyNet URL
- Role and password credentials
- Connection test function

**Synchronization**:
- Cars: Import racer roster (one-way sync)
- Categories: Import award definitions
- Results: Export winners (push only)

### Data Management

**Clear All Data**: Removes all votes while preserving configuration. Use between events to reset the system.

---

## Troubleshooting

### Access Issues

If voters cannot reach the voting interface:
- Verify the server is running and accessible on the network
- Check firewall settings allow connections on the configured port
- Confirm the URL matches the server's network address
- Test access from the admin computer first

### QR Code Problems

Codes not working:
- Check "Require Registered QR Codes" setting in Admin → Settings
- If enabled, only pre-generated codes are valid
- Disable the setting to allow any code (open mode)
- Voters can manually enter codes at the landing page

### DerbyNet Connection Failures

Synchronization errors:
- Verify DerbyNet URL is correct and accessible
- Test credentials by logging into DerbyNet directly
- Ensure both servers are on the same network
- Check DerbyNet server is running

### Vote Conflicts

If exclusivity rules aren't enforcing:
- Verify category group has an Exclusivity Pool ID set
- Confirm categories are assigned to the group
- Check that categories share the same pool ID

### Performance Issues

Slow response times:
- Check database size
- Verify adequate system resources
- Review number of concurrent connections
- Consider restarting the server

### Data Loss Prevention

The SQLite database file contains all data:
- Back up the database file regularly
- Database location specified by `-db` flag
- Copy the file to preserve event data

---

## Best Practices

### Before the Event

- Test the complete workflow with sample data
- Generate QR codes for each voter (if requiring pre-registered voters). We provide a QR code with each Scout's Pit Pass.
- Verify DerbyNet integration if using
- Prepare backup paper ballots
- Ensure sufficient device charging

### During the Event

- Monitor the statistics dashboard regularly
- Respond promptly to voter questions
- Watch for unusual patterns in results
- Keep voting window appropriate for event size

### After the Event

- Close voting before announcing results
- Review all results for anomalies
- Resolve ties systematically
- Verify DerbyNet export completed successfully
- Back up the database file

### Security Considerations

- Use a strong admin password in production
- Restrict network access to trusted connections if possible
- Monitor the voter list for unexpected entries
- Clear data between events to prevent confusion

---

## Command-Line Reference

```bash
derbyvote [options]

Options:
  -port int         HTTP server port (default 8081)
  -db string        SQLite database path (default "voting.db")
  -adminpw string   Admin password (generated if not specified)
  -loglevel string  Log level: debug, info, warn, error (default "info")
  -noanimate        Disable startup animation
  -nokeyboard       Disable keyboard shortcuts
  -version          Show version
  -help             Show help
```

### Keyboard Shortcuts

While server is running (if keyboard mode enabled):
- `h` - Toggle HTTP request logging
- `l` - Cycle log levels
- `?` - Show shortcuts help
- `Ctrl+C` - Graceful shutdown

---

## Additional Resources

Technical documentation: [DEVELOPING.md](DEVELOPING.md)

Project overview: [README.md](README.md)

## License

Licensed under the MIT License. See [LICENSE](LICENSE) file for details.
