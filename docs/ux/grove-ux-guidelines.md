# Grove UX Guidelines

This document is the source of truth for how Grove should behave across CLI, TUI, and MenuBar.

## Purpose

- Keep interactions consistent across surfaces.
- Prevent docs and behavior drift.
- Make user-facing copy actionable and predictable.

## Canonical Model

- **Worktree**: git worktree identity (name, path, branch, repo).
- **Server**: process runtime attached to a worktree.
- **Workspace**: UI-level combined view of worktree plus optional server/activity data.

Rule: do not use these terms interchangeably in user-facing copy.

## Action Semantics

- `start`: starts server for a worktree context.
  - If context is required, copy must say so explicitly (for example: `cd <path> && grove start`).
- `stop`: stops running server by name or current context.
- `restart`: canonical name-based restart action (`grove restart <name>`).
- `open`: opens running server URL.
- `logs`: opens/streams logs for selected target.

Rule: never show command signatures that do not exist.

## Feedback Contract

Every async action must provide all three states:

1. `pending` immediately after user action
2. `success` when state mutation completes
3. `error` with one concrete next step

Error copy format:
- What failed
- Why (if known)
- Exact recovery command/path

Example:
- "Failed to start api-feature. Run `cd /path/to/worktree && grove start` to retry."

## Cross-Surface Consistency Rules

### CLI
- Help and errors must include next-step guidance for common failure modes.
- Keep command examples aligned with real command signatures.

### TUI
- Keybindings shown in help/docs must match shipped keymap.
- If an action is not implemented, say so explicitly and provide a valid fallback command.
- Do not perform destructive partial behavior behind a "guidance" message.

### MenuBar
- Scope filter (`Servers Only`, `Active Worktrees`, `All Worktrees`) must actually change visible rows.
- Keep one clear primary action per row; secondary actions live in a consistent overflow location.
- Avoid duplicate error channels; one global stream plus row-local action feedback.

## IA and Discoverability Principles

- Prefer "triage first" in MenuBar; move advanced workflows to dedicated windows/CLI.
- Keep shortcut discoverability explicit in one place per surface.
- Use progressive disclosure: primary action first, advanced actions second.

## Documentation Parity Rules

Any change to one of these must update all related surfaces in the same PR:

- TUI keybindings
- README shortcut table
- Help text and tooltip copy
- Command examples in docs

Recommended PR checklist:
- [ ] Updated user-facing copy is command-accurate
- [ ] README shortcut table matches keymap
- [ ] New errors include concrete recovery action
- [ ] MenuBar scope/filter behavior covered in manual test notes

## Change Governance

When introducing a new command or action behavior:

1. Define signature and context requirements.
2. Add/update canonical copy in CLI help text.
3. Update TUI/MenuBar guidance text.
4. Update docs (`README.md` and this file if rule-level).
5. Add/adjust tests where possible.

## Current Known Gaps (Track for Follow-Up)

- MenuBar still has concentrated view/state complexity in `MenuView.swift`.
- Action feedback is partially global and partially local in MenuBar.
- Some discovery/start UX paths still need stronger completion messaging.

These are expected to be addressed during the phased UX roadmap execution.
