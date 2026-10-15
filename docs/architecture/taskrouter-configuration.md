# Task Router configuration

What must exist in Twilio for the four flows in this directory to work. Every requirement
below is one the code actually imposes; file references are given so each can be checked
against the source rather than taken on trust.

Flows: [direct answer](direct-answer-call.d2) · [warm transfer](warm-transfer-call.d2) ·
[accept a transfer](accept-warm-transfer.d2) · [chat](chat-conversation.d2)

## At a glance

- 2 task queues — one holding the virtual agent, one named exactly `Everyone`
- 3 workflows — inbound voice, chat, and Flex's own; **only the voice one may point at the
  assignment callback**
- Workspace event callback enabled for `reservation.created`
- `contact_uri` on every worker that can be dialled, `isVirtualAgent` on the assistant
- `isVirtualAgentTask` on inbound voice and chat tasks, set upstream

## Queues

Two are needed.

### The virtual agent queue

Any friendly name. Holds the worker representing the assistant. The code never refers to
this queue by name — it is reached purely through workflow routing, so its name is yours to
choose.

Voice and chat can both be served from this one queue. Nothing in the code distinguishes
them at the queue level; the split happens by task channel (`handleReservationCreated`,
`controllers/event-taskrouter-post.go`).

### `Everyone`

Holds the human agents the assistant hands callers to. The name is fixed:

- `models.TransferQueueEveryone` hardcodes the string `Everyone`
- `FindTaskQueueByName` (`services/twilio/twilio.go`) resolves it by exact friendly-name
  match against the Task Router API
- it is the only value in the `enum` of the `transfer_call` tool schema
  (`services/agent-tools/queue-transfer.go`), so it is the only queue the assistant can ask
  for

To support more transfer targets, all three have to change together.

### Keep them disjoint

The virtual agent worker must not be reachable from `Everyone`. If it is, a warm transfer
can be routed straight back to the assistant. Either leave it out of that queue or exclude
it in the queue's target worker expression.

## Workflows

Three, and the difference between them is which one carries the assignment callback.

### 1. Inbound voice

Assignment Callback URL:

```
{TWILIO_WEBHOOK_BASE_URL}/twilio/taskrouter/queue-assignment-callback
```

Routes to the virtual agent queue. This is the only workflow that may point at that URL —
see the constraint below.

### 2. Chat

Must **not** carry an assignment callback pointing at this application. Leave it unset, so
Task Router creates the reservation and waits for it to be accepted over the API.
`handleChatReservationCreated` accepts it off the workspace event stream and wires the
conversation up.

### 3. Flex Task Assignment

Flex's own workflow, unchanged. It likewise must not point at this application's assignment
callback. When a Flex agent transfers a call to the virtual agent worker, the reservation is
picked up from the workspace event stream by `handleVoiceReservationCreated`.

### Warm transfer needs no workflow of its own

`startWarmTransfer` (`controllers/event-openai-post.go`) creates the transfer task on the
inbound task's own `WorkflowSid` and sets `routing_target` to the `Everyone` queue sid.
Routing target bypasses filter evaluation, so no filter matching `target_queue` is required
on the inbound workflow.

### The constraint that matters most

`controllers/taskrouter-queue-assignment-callback-post.go` has exactly two branches:

- task attributes say `is_warm_transfer` and name a conference, so answer `accept` and dial
  the worker into the conference the caller is already on
- anything else, so answer with a `conference` instruction

That second branch is unconditional. Any task reaching this endpoint that is not one of our
own warm transfers is told to start a conference named after its own task sid:

- a **Flex transfer** would get a brand new empty conference instead of joining the
  customer's, and the transfer breaks with no obvious error
- a **chat task** would get a voice instruction that means nothing for chat

So exactly one workflow may point at the assignment callback, and chat and voice cannot
share a workflow.

## Workspace settings

Event Callback URL:

```
{TWILIO_WEBHOOK_BASE_URL}/twilio/taskrouter/event
```

with **`reservation.created`** enabled — the handler switches on the CloudEvents type
`com.twilio.taskrouter.reservation.created`.

Both the chat flow and Flex transfers into the assistant depend on this; neither works
without it. It must be the same workspace as `TWILIO_WORKSPACE_SID`, since that is the
workspace the application accepts reservations in.

## Worker attributes

### The virtual agent worker

```json
{
  "isVirtualAgent": true,
  "contact_uri": "sip:<your-openai-sip-address>"
}
```

`isVirtualAgent` gates every reservation handler (`IsAssignedToVirtualAgent`). `contact_uri`
is the address Task Router dials on an inbound call, and the address this application dials
when joining the assistant to an existing conference.

### Human agents in `Everyone`

```json
{
  "contact_uri": "client:agent_name"
}
```

An E.164 number works too. This is required: `joinWorkerToConference` gives up if it is
missing, so a warm transfer to an agent without one accepts the reservation and then never
connects anybody.

## Task attributes

These are set by whatever creates the task — a Studio flow or TwiML application — not by
this repository.

### Inbound voice tasks

```json
{ "isVirtualAgentTask": true }
```

Load bearing. It is what makes the event stream handler skip the reservation that the
assignment callback already owns (`IsTransferToVirtualAgent`,
`models/virtual-agent-transfer.go`). Without it the assistant is dialled onto every inbound
call twice.

### Chat tasks

```json
{ "isVirtualAgentTask": true, "conversationSid": "CHxxxxxxxx" }
```

`conversationSid` is a hard requirement — `handleChatReservationCreated` fails the
reservation without it, because the conversation is where the webhook and the task sid have
to be recorded.

## Hardcoded values to reconcile

`controllers/taskrouter-queue-assignment-callback-post.go` pins two account-specific values:

| Constant | Value | Used for |
|---|---|---|
| `assignmentPostWorkActivitySid` | `WA1a65e9e0f4f271d8c0d8ea5cc831ef9d` | The post-work activity on the conference instruction |
| `assignmentCallerId` | `+17039975700` | The `From` on the assistant's leg, and on every agent dialled into a conference |

Both must exist in the target workspace and account. These are the first things to break if
this is deployed against a second Twilio project, and the obvious candidates to move into
`config.TwilioConfig` alongside the other Twilio settings.
