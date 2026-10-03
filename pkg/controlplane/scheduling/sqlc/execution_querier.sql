-- name: GetExecutionTransitions :many
-- Control instructions the runner must act on: pause, resume, abort, cancel.
--
-- Cheap by construction, which is why it needs no bound: three columns from one
-- table, no join and no hydration. The number of executions mid-transition is
-- bounded by what a user can click, not by the dispatch backlog.
SELECT
    r.execution_id,
    r.status,
    r.predicted_status
FROM
    test_workflow_results r
WHERE r.status = ANY(@statuses::text[]);

-- name: GetExecutionsToStart :many
-- One bounded page of executions to hand to the runner, oldest scheduled first.
--
-- The bound is the point. This query's ancestor returned every pending row and
-- joined each one to six more tables, so a burst of workflows sharing a cron
-- minute made a single poll exceed the runner's call deadline; the runner then
-- backed off, the backlog stopped draining, and the query only got slower.
--
-- Reads test_workflow_executions alone: status and workflow_name are
-- denormalised onto it by trigger (see the denormalize_execution_status_name
-- migration), and every field the runner is sent lives on this row. The
-- workflow spec deliberately is not fetched - the runner asks for it separately
-- with GetExecutionWorkflow.
--
-- STARTING rows come back only once their dispatch lease has expired. The
-- handler stamps status_at when it hands a row out, so a row the runner never
-- acknowledged is retried after @redispatch_before rather than occupying a slot
-- in every poll.
SELECT
    sqlc.embed(e)
FROM
    test_workflow_executions e
WHERE e.status = 'assigned'
   OR (e.status = 'starting' AND e.status_at < @redispatch_before::timestamptz)
ORDER BY e.scheduled_at
LIMIT @batch_size::int;

-- name: GetStaleStartingExecutions :many
-- Executions handed to a runner that never reported back, neither accepting nor
-- declining them. They are failed explicitly rather than left to sit in STARTING
-- forever while the control plane reports healthy.
--
-- Bounded so one reaper tick cannot stall on a large backlog.
SELECT
    e.id
FROM
    test_workflow_executions e
WHERE e.status = 'starting'
  AND e.status_at < @stale_before::timestamptz
ORDER BY e.scheduled_at
LIMIT @batch_size::int;
