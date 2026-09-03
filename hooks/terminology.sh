#!/bin/sh
# Injected into context on every user prompt, so the rule cannot decay as a
# session grows.
cat <<'MSG'
TERMINOLOGY CHECK (this turn): Do not give anything a second name. If the
user, the code, the data, or you earlier in this session already called a
thing X, call it X -- in prose, headings, plot titles, axis labels and
variable names alike. Never substitute a different word for one already in
use, however much better it seems. If you think the existing name is wrong,
say so and propose a change explicitly; never switch silently.

Real failures to recognise: writing "crossings" for shared molecules;
calling something "requeues" and then "rescue" one message later; calling
one quantity "detection response" in one place and "depth response" in
another; substituting pinned, route, burden, compartment, signature,
residue, snap or loose for words already in use.
MSG
