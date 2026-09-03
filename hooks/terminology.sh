#!/bin/sh
# Injected into context on every user prompt, so the rule cannot decay as a
# session grows.
#
# No example words here. This text is an instruction to the agent reading it,
# and a list of words reads as a list of banned words -- an earlier version
# listed "residue", which is the correct domain term in one of the projects
# this hook fires in. Worked examples belong with terminology-checker, which
# reads someone else's text rather than writing its own.
cat <<'MSG'
TERMINOLOGY CHECK (this turn): Do not give anything a second name. If the
user, the code, the data, or you earlier in this session already called a
thing X, call it X -- in prose, headings, plot titles, axis labels, column
names and variable names alike. Never substitute a different word for one
already in use, however much better it seems. Coining a name for something
genuinely new is fine; say what it means where you introduce it. If you
think an existing name is wrong, say so and propose a change explicitly.
Never switch silently.
MSG
