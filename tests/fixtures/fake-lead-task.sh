#!/usr/bin/env bash
# Stand-in for lead-task in tests: the same commands, input fields and exit codes (0/1/3/4/5) as the contract.
# The journal lives in $LEAD_TASKS_DIR; every journal operation runs under flock on .lock beside it.
# FAKE_LEAD_TASK_MAIL: ok (default), fail (no panel: answer exits 3), lost (the letter is sent, then the
# process dies before it marks the delivery: answer exits 1). FAKE_LEAD_TASK_EXPORT=fail makes export exit 1.
# Accepted letters are appended to $LEAD_TASKS_DIR/mailbox.log, one JSON line each.
set -euo pipefail

DIR="${LEAD_TASKS_DIR:?LEAD_TASKS_DIR is required}"
JOURNAL="$DIR/events.jsonl"
SOURCE="lead-task"
mkdir -p "$DIR"
exec 9>"$DIR/.lock"

lock() { flock -x 9; }
unlock() { flock -u 9; }
fail() {
	local code=$1
	shift
	echo "fake-lead-task: $*" >&2
	exit "$code"
}

JQ_ISO='def iso: (. * 1000 | floor) as $ms
	| ($ms / 1000 | floor | strftime("%Y-%m-%dT%H:%M:%S")) + "." + ((1000 + ($ms % 1000)) | tostring | .[1:]) + "Z";'

journal_jq() { jq -c -s "$@" "$JOURNAL"; }

record() {
	jq -cn --argjson ts "$(date +%s.%3N)" --argjson e "$1" '{ts: $ts} + $e' >> "$JOURNAL"
}

epoch() { head -n1 "$JOURNAL" | sha256sum | cut -c1-12; }

fingerprint() { jq -cS '{body, options, scope}' <<<"$1" | tr -d '\n' | sha256sum | cut -d' ' -f1; }

find_ask() { journal_jq --arg q "$1" 'map(select(.ev == "ask" and .qid == $q)) | .[0] // empty'; }

cmd_new() {
	local title=${1:-}
	[[ -n $title ]] || fail 1 "usage: new <title>"
	lock
	local n=0
	[[ -f $JOURNAL ]] && n=$(journal_jq 'map(select(.ev == "new")) | length')
	local task
	task=$(printf 'T-%04d' $((n + 1)))
	record "$(jq -cn --arg task "$task" --arg title "$title" '{ev: "new", task: $task, title: $title}')"
	jq -cn --arg task "$task" '{task: $task}'
}

cmd_ask() {
	local task=${1:-} input
	input=$(cat)
	jq -e '(.body | type == "string") and (.scope | type == "string" and length > 0)
		and ([.options[].id] | length == (unique | length))
		and (.recommendation == null or (.recommendation.option_id as $o | [.options[].id] | index($o) != null))' \
		<<<"$input" >/dev/null 2>&1 || fail 1 "invalid ask input"
	lock
	[[ -f $JOURNAL ]] || fail 1 "$task: no such task"
	[[ $(journal_jq --arg t "$task" 'map(select(.ev == "new" and .task == $t)) | length') -gt 0 ]] ||
		fail 1 "$task: no such task"
	local n qid
	n=$(journal_jq 'map(select(.ev == "ask")) | length')
	qid=$(printf 'Q-%04d' $((n + 1)))
	record "$(jq -c --arg task "$task" --arg qid "$qid" \
		'{ev: "ask", task: $task, qid: $qid, body, options, recommendation, scope}' <<<"$input")"
	jq -cn --arg qid "$qid" '{qid: $qid}'
}

delivered() { journal_jq --arg id "$1" 'any(.[]; .ev == "delivered" and .answer_id == $id)'; }

send_mail() {
	[[ ${FAKE_LEAD_TASK_MAIL:-ok} == fail ]] && return 1
	jq -cn --arg qid "$1" --arg id "$2" --argjson late "$3" '{qid: $qid, answer_id: $id, late: $late}' >> "$DIR/mailbox.log"
}

cmd_answer() {
	local qid=${1:-} input
	input=$(cat)
	jq -e --arg q "$qid" 'all(.source, .epoch, .qid, .answer_id, .question_fp, .answered_at; type == "string" and length > 0)
		and .qid == $q' <<<"$input" >/dev/null 2>&1 || fail 1 "invalid answer input"

	lock
	[[ -f $JOURNAL ]] || fail 1 "$qid: no such question"
	local ask
	ask=$(find_ask "$qid")
	[[ -n $ask ]] || fail 1 "$qid: no such question"

	local source epoch fp answer_id
	source=$(jq -r .source <<<"$input")
	epoch=$(jq -r .epoch <<<"$input")
	fp=$(jq -r .question_fp <<<"$input")
	answer_id=$(jq -r .answer_id <<<"$input")
	[[ $source == "$SOURCE" && $epoch == "$(epoch)" && $fp == "$(fingerprint "$ask")" ]] ||
		fail 5 "$qid: another journal or another question"

	local stored late
	stored=$(journal_jq --arg q "$qid" 'map(select(.ev == "answer" and .qid == $q)) | .[0].answer_id // empty' | tr -d '"')
	if [[ -z $stored ]]; then
		late=$(journal_jq --arg q "$qid" 'any(.[]; .ev == "resolved" and .qid == $q)')
		record "$(jq -c --argjson late "$late" '{ev: "answer", qid, answer_id, option_id, option_label, text: (.text // ""),
			answered_at, late: $late}' <<<"$input")"
	elif [[ $stored != "$answer_id" ]]; then
		fail 4 "$qid: already answered with another answer_id"
	fi

	if [[ $(delivered "$answer_id") == false ]]; then
		unlock
		late=$(journal_jq --arg q "$qid" 'map(select(.ev == "answer" and .qid == $q)) | .[0].late')
		send_mail "$qid" "$answer_id" "$late" || fail 3 "$qid: the letter was not accepted"
		[[ ${FAKE_LEAD_TASK_MAIL:-ok} == lost ]] && fail 1 "$qid: died after the letter"
		lock
		if [[ $(delivered "$answer_id") == false ]]; then
			record "$(jq -cn --arg qid "$qid" --arg id "$answer_id" '{ev: "delivered", qid: $qid, answer_id: $id}')"
		fi
	fi
}

cmd_resolve() {
	local qid=${1:-} note="" elsewhere=false
	shift || true
	while [[ $# -gt 0 ]]; do
		case $1 in
		-m) note=${2:-}; shift 2 ;;
		--elsewhere) elsewhere=true; shift ;;
		*) fail 1 "unknown argument $1" ;;
		esac
	done
	lock
	[[ -f $JOURNAL ]] || fail 1 "$qid: no such question"
	[[ -n $(find_ask "$qid") ]] || fail 1 "$qid: no such question"
	if [[ $(journal_jq --arg q "$qid" 'any(.[]; .ev == "resolved" and .qid == $q)') == false ]]; then
		record "$(jq -cn --arg qid "$qid" --arg note "$note" --argjson el "$elsewhere" \
			'{ev: "resolved", qid: $qid, note: $note, elsewhere: $el}')"
	fi
}

cmd_export() {
	[[ ${FAKE_LEAD_TASK_EXPORT:-ok} == fail ]] && fail 1 "export failed on request"
	lock
	[[ -f $JOURNAL ]] || fail 1 "no journal"
	local fps="{}" qid
	while IFS= read -r ask; do
		qid=$(jq -r .qid <<<"$ask")
		fps=$(jq -c --arg q "$qid" --arg fp "$(fingerprint "$ask")" '. + {($q): $fp}' <<<"$fps")
	done < <(journal_jq '.[] | select(.ev == "ask")')

	local now rows
	now=$(date +%s.%3N)
	mapfile -t rows < <(journal_jq --argjson fps "$fps" "$JQ_ISO"'
		def qview($ev):
			. as $a
			| ([$ev[] | select(.ev == "answer" and .qid == $a.qid)] | .[0]) as $ans
			| ([$ev[] | select(.ev == "resolved" and .qid == $a.qid)] | .[0]) as $res
			| {qid: $a.qid, fp: $fps[$a.qid], options: $a.options, recommendation: $a.recommendation, scope: $a.scope,
			   answer: (if $ans == null then null else
			     {answer_id: $ans.answer_id, option_id: $ans.option_id, text: $ans.text, answered_at: $ans.answered_at,
			      late: $ans.late, delivered: ([$ev[] | select(.ev == "delivered" and .answer_id == $ans.answer_id)] | length > 0)}
			   end),
			   resolved: (if $res == null then null else {at: ($res.ts | iso), note: $res.note, elsewhere: $res.elsewhere} end)};
		[range(0; length) as $i | .[$i] + {ref: ($i + 1 | tostring)}] as $ev
		| ([$ev[] | select(.ev == "new")] | map(
			. as $t
			| [$ev[] | select(.ev == "ask" and .task == $t.task)] as $asks
			| [$asks[] | select(. as $a | [$ev[] | select(.ev == "resolved" and .qid == $a.qid)] | length == 0)] as $open
			| {type: "task", id: $t.task, title: $t.title, project: "fake", agent: "claude", created_at: ($t.ts | iso),
			   status: (if ($open | length) > 0 then "wait_user" else "work" end),
			   waiting_on: (if ($open | length) > 0 then "owner" else "agent" end),
			   status_since: (($open[0].ts // $t.ts) | iso), note: "", returns: 0,
			   spent: {queued: 0, work: 0, lead: 0, user: 0},
			   transitions: ([{at: ($t.ts | iso), status: "work", cause: ("j:" + $t.ref), derived: false}]
			     + (if ($open | length) > 0
			        then [{at: ($open[0].ts | iso), status: "wait_user", cause: ("j:" + $open[0].ref), derived: false}]
			        else [] end))})) as $tasks
		| ([$ev[] | select(.ev == "new" or .ev == "ask") | . as $e
			| {type: "record", task: $e.task, at: ($e.ts | iso), actor: "lead", actor_derived: false,
			   source: {kind: "journal", ref: $e.ref}, body: (if $e.ev == "new" then "Task " + $e.task + ": " + $e.title else $e.body end),
			   body_missing: null, format: "text", id: ("j:" + $e.ref), kind: (if $e.ev == "new" then "letter" else "question" end)}
			  + (if $e.ev == "ask" then {question: ($e | qview($ev))} else {} end)]) as $recs
		| $tasks[], $recs[]')

	local body revision tasks schema=1
	body=$(printf '%s\n' "${rows[@]}")
	body=${body%$'\n'}
	revision=$(printf '%s' "$body" | sha256sum | cut -c1-16)
	tasks=$(printf '%s\n' "${rows[@]}" | grep -c '^{"type":"task"' || true)
	[[ $body == *'"kind":"question"'* ]] && schema=2
	jq -cn --argjson now "$now" --argjson schema "$schema" --arg epoch "$(epoch)" --arg rev "$revision" "$JQ_ISO"'
		{type: "header", schema: $schema, source: "lead-task", epoch: $epoch, revision: $rev, as_of: ($now | iso),
		 derivation: 1, complete: true, errors: []}'
	printf '%s\n' "${rows[@]}"
	jq -cn --arg rev "$revision" --argjson tasks "$tasks" --argjson records "$((${#rows[@]} - tasks))" \
		'{type: "footer", revision: $rev, tasks: $tasks, records: $records}'
}

command=${1:-}
shift || true
case $command in
new) cmd_new "$@" ;;
ask) cmd_ask "$@" ;;
answer) cmd_answer "$@" ;;
resolve) cmd_resolve "$@" ;;
export) cmd_export ;;
*) fail 1 "unknown command ${command:-<none>}" ;;
esac
