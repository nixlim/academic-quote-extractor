# OpenCode Beads Plugin

Integrates [Beads (bd)](https://github.com/steveyegge/beads) issue tracking with
[OpenCode](https://github.com/anomalyco/opencode). This is the OpenCode equivalent
of running `bd setup claude` for Claude Code.

## What It Does

| OpenCode Event              | Action                                  | Claude Code Equivalent |
|-----------------------------|------------------------------------------|------------------------|
| `session.created`           | Runs `bd prime`, injects into session    | `SessionStart` hook    |
| `session.compacting`        | Injects `bd prime` into compaction context | `PreCompact` hook    |
| `session.idle`              | Optional `bd sync`                       | (no equivalent)        |

### session.created (SessionStart)

When a new OpenCode session is created, the plugin runs `bd prime` and injects
the output into the session as a context-only message (using the SDK's
`client.session.prompt()` with `noReply: true`). This gives the agent awareness
of open issues, dependencies, priorities, and the beads command reference.

**Important timing note:** Unlike Claude Code's `SessionStart` hook (which fires
when the CLI process starts), OpenCode's `session.created` event fires lazily --
only when the first prompt is sent, not when the TUI launches. OpenCode's TUI is
a persistent application where sessions are created on-demand; you can sit in the
TUI, browse history, or switch sessions without a new session being created. The
session (and this event) only comes into existence when you actually send a
message.

This means `bd prime` runs **concurrently with** (not before) the LLM processing
your first prompt. The prime output is injected as a `noReply: true` message into
the session, and the LLM sees it because it appears in the message history. In
practice the agent does receive the beads context and can act on it, but the
first prompt has already been submitted by the time the event fires -- so the
sequence is:

1. User sends first message
2. OpenCode creates the session and fires `session.created`
3. The plugin's event handler runs `bd prime` and injects the output
4. The LLM reads the message stream (which now includes both the user prompt
   and the injected beads context) and generates its response

This differs from Claude Code where the `SessionStart` hook runs and completes
*before* the first prompt is processed. The practical effect is the same -- the
agent has beads awareness in its first response -- but the mechanism is
concurrent injection rather than sequential pre-loading.

### session.compacting (PreCompact)

When OpenCode compacts a long conversation to free up context window space, the
plugin injects the `bd prime` output into the compaction context. This ensures
beads knowledge survives compaction and the agent retains awareness of the issue
tracker state.

### session.idle (optional sync)

When the agent finishes and the session goes idle, the plugin can optionally run
`bd sync` to push beads changes to git. Disabled by default.

## Prerequisites

1. `bd` CLI installed and on your PATH
2. `bd init` already run in the project (`.beads/` directory must exist)

The plugin silently deactivates if no `.beads/` directory is found.

## Installation

The plugin is already in place at `.opencode/plugins/beads.js`. OpenCode
automatically loads all plugins from `.opencode/plugins/` at startup.

No additional configuration is required.

## Environment Variables

Control the plugin behaviour with environment variables:

| Variable                | Default | Effect                                    |
|-------------------------|---------|-------------------------------------------|
| `BEADS_DISABLE_PRIME`   | unset   | Set to `1` to skip `bd prime` on session start |
| `BEADS_DISABLE_COMPACT` | unset   | Set to `1` to skip injecting context on compaction |
| `BEADS_DISABLE_IDLE`    | unset   | Set to `1` to skip idle event handling    |
| `BEADS_SYNC_ON_IDLE`    | unset   | Set to `1` to auto-run `bd sync` when session idles |

Example:

```bash
# Disable auto-prime, enable auto-sync
BEADS_DISABLE_PRIME=1 BEADS_SYNC_ON_IDLE=1 opencode
```

## Customising the Plugin

The plugin is a standard OpenCode plugin (ES module exporting an async function).
Edit `.opencode/plugins/beads.js` to change behaviour.

### Common modifications

**Add more events:**
OpenCode exposes many events you can hook into. Add handlers inside the
`event` callback:

```js
event: async ({ event }) => {
  if (event.type === "session.created") {
    // ... existing prime logic
  }

  // Example: log every file edit
  if (event.type === "file.edited") {
    console.log("File edited:", event);
  }
}
```

Available events: `session.created`, `session.idle`, `session.updated`,
`session.error`, `session.compacted`, `session.deleted`, `file.edited`,
`tool.execute.before`, `tool.execute.after`, `message.updated`, and more.
See https://opencode.ai/docs/plugins for the full list.

**Change the compaction context:**
Edit the `experimental.session.compacting` handler to inject different or
additional context:

```js
"experimental.session.compacting": async (_input, output) => {
  const prime = await runBd("prime");
  const ready = await runBd("ready", "--json");
  if (prime) {
    output.context.push(`## Beads Context\n\n${prime}`);
  }
  if (ready) {
    output.context.push(`## Ready Issues\n\n${ready}`);
  }
}
```

**Replace the compaction prompt entirely:**
Set `output.prompt` to override the default compaction prompt:

```js
"experimental.session.compacting": async (_input, output) => {
  output.prompt = `You are resuming a session that uses beads (bd) for tracking.
Run bd prime to reload context, then bd ready to find work.`;
}
```

**Run bd commands before/after tool execution:**

```js
"tool.execute.before": async (input, output) => {
  // Example: auto-sync before any bash command
  if (input.tool === "bash") {
    await runBd("sync");
  }
}
```

**Send system notifications:**

```js
if (event.type === "session.idle") {
  await $`osascript -e 'display notification "Session done - run bd sync" with title "OpenCode"'`;
}
```

## Using with bd setup (custom recipe)

You can register this as a custom `bd setup` recipe so other projects can
install it with one command:

```bash
bd setup --add opencode .opencode/plugins/beads.js
```

Then in any project:

```bash
bd setup opencode
```

## Troubleshooting

**Plugin not loading:**
- Check that `.opencode/plugins/beads.js` exists
- Ensure `.beads/` directory exists in the project root (`bd init` if not)
- Run `opencode --print-logs --log-level DEBUG` to see plugin load messages on stderr

**bd prime returning empty:**
- Run `bd prime` manually in your terminal to verify output
- Check `bd doctor` for any beads installation issues

**Compaction not preserving context:**
- The `experimental.session.compacting` hook is an experimental API
- Check OpenCode release notes if behaviour changes after an update

**bd prime only runs after sending the first message, not on TUI launch:**
- This is expected. OpenCode creates sessions lazily -- the `session.created`
  event fires when you send your first prompt, not when the TUI starts. The
  prime context is injected concurrently into the session message stream, so
  the agent still has beads awareness in its first response. See the timing
  note under "session.created" above for details.

**Prime context not appearing in session:**
- The plugin uses `client.session.prompt()` with `noReply: true` to inject
  context. Verify the SDK client is available by checking OpenCode's plugin
  loading logs.
- Ensure `@opencode-ai/plugin` is installed (check `.opencode/package.json`)
