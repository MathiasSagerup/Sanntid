# Exercise 4: Process Pairs - Go Implementation

## Overview

This is a Go implementation of the **Process Pairs** exercise from TTK4145. The program demonstrates fault tolerance through a self-healing process pair mechanism that continuously prints sequential numbers without skipping.

## How It Works

### Architecture

- **Primary Process**: Counts continuously and prints numbers (1, 2, 3, ...) while sending heartbeat messages
- **Backup Process**: Listens silently for heartbeats from the primary and takes over if the primary dies
- **Communication**: UDP on localhost for heartbeat signals
- **State Persistence**: Counter state is saved to a file (`/tmp/process_pairs_state`) so the backup can resume from where the primary left off

### Process Flow

1. Main program starts as the **Primary**
2. Primary spawns a backup process
3. Primary enters its counting loop:
   - Increments counter every 100ms
   - Prints the counter value
   - Sends the counter value to the backup via UDP heartbeat
   - Saves counter to persistent file
4. Backup enters listening mode:
   - Listens on UDP port 12346
   - Resets timeout counter when receiving heartbeats
   - If it misses 5 consecutive heartbeats (2.5 seconds), it assumes primary is dead
5. When backup detects primary death:
   - Closes its listening socket
   - Calls `runAsPrimary()` to become the new primary
   - Reads the last counter value from the state file
   - Spawns its own new backup process
   - Continues counting from where the previous primary left off

## Building

```bash
cd /home/student/Documents/Gruppe82/Sanntid/assignment4
go build -o process-pairs main.go
```

## Running

```bash
./process-pairs
```

Or with `go run`:

```bash
go run main.go
```

### Running the Backup Mode (Automatic)

The backup mode is started automatically by the primary process. However, you can manually start a backup for testing:

```bash
./process-pairs backup
```

## Configuration

The following constants can be adjusted in `main.go`:

- `PRIMARY_PORT = "12345"` - UDP port for primary communication
- `BACKUP_PORT = "12346"` - UDP port for backup communication
- `HEARTBEAT_INTERVAL = 100 * time.Millisecond` - How often primary sends updates
- `TIMEOUT_DURATION = 500 * time.Millisecond` - Max time to wait for each heartbeat
- `TIMEOUT_THRESHOLD = 5` - Number of missed heartbeats before takeover

## Testing the Process Pair

To test the mechanism:

1. Start the program: `./process-pairs`
2. Watch both windows (primary and backup) - numbers should keep incrementing
3. Kill the primary window (close it or Ctrl+C)
4. The backup should take over within ~2.5 seconds and continue counting without skipping

## State Persistence

The counter is saved to `/tmp/process_pairs_state` after each increment. This ensures:
- The backup knows what number to continue from
- If both processes die, the counter can be recovered (though subsequent runs will start from the last saved state)

To reset the state, delete the file:

```bash
rm /tmp/process_pairs_state
```

## Terminal Spawning

The program attempts to spawn terminal windows in the following order:
1. `gnome-terminal` (GNOME Desktop)
2. `xterm` (X11 fallback)

If neither is available, the backup process will not spawn in a separate window. You can modify the `spawnBackup()` function to support other terminal emulators.

## Key Design Decisions

1. **UDP Communication**: Simple, connection-less protocol ideal for heartbeat monitoring
2. **File-based State**: Reliable persistence without complex serialization
3. **Timeout-based Detection**: No complex health checks needed; missing heartbeats indicate failure
4. **Recursive Process Creation**: New backup automatically takes over when current primary dies
5. **Non-blocking Architecture**: No threads needed; single-threaded event loop using `time.Ticker` and signal handling

## Possible Improvements

1. Add graceful shutdown to prevent orphaned processes
2. Implement exponential backoff for backup takeover
3. Add command-line arguments for configuration
4. Support for multiple backups or geographic distribution
5. Metrics/monitoring for process pair health

## References

- Original Exercise: https://github.com/TTK4145/Exercise4
- Process Pairs Concept: Used in fault-tolerant systems (Ericsson AXE, etc.)
- Go UDP Documentation: https://golang.org/pkg/net/
