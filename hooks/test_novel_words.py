#!/usr/bin/env python3
"""Tests for novel-words.py.

Run with `python3 hooks/test_novel_words.py` or `make test-hooks`.

Each test builds a small transcript, runs the real hook against it with a
temporary state directory, and asserts whether it reports and what it names.
Exit 2 means it reported; exit 0 means it stayed silent.
"""

import datetime
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HOOK = os.path.join(os.path.dirname(os.path.abspath(__file__)), "novel-words.py")


def assistant(text: str) -> dict:
    return {"type": "assistant", "message": {"content": [{"type": "text", "text": text}]}}


def user(text: str) -> dict:
    return {"type": "user", "message": {"content": text}}


def tool_result(text: str) -> dict:
    return {"type": "user", "message": {"content": [{"type": "tool_result", "content": text}]}}


class NovelWords(unittest.TestCase):
    def setUp(self) -> None:
        self.directory = tempfile.mkdtemp(prefix="novel-words-test-")
        os.makedirs(os.path.join(self.directory, ".claude", "termcheck"))

    def tearDown(self) -> None:
        shutil.rmtree(self.directory)

    def run_hook(self, entries: list[dict], session: str = "s1") -> tuple[int, str]:
        path = os.path.join(self.directory, f"{session}.jsonl")
        with open(path, "w") as handle:
            for entry in entries:
                handle.write(json.dumps(entry) + "\n")
        payload = json.dumps({
            "session_id": session, "transcript_path": path, "stop_hook_active": False,
        })
        finished = subprocess.run(
            [sys.executable, HOOK], input=payload, capture_output=True, text=True,
            env=dict(os.environ, HOME=self.directory),
        )
        return finished.returncode, finished.stderr

    def test_silent_when_nothing_is_new(self) -> None:
        code, _ = self.run_hook([
            user("tell me about the shared molecules"),
            assistant("The shared molecules are the shared molecules we discussed."),
        ])
        self.assertEqual(0, code)

    def test_reports_a_repeated_new_word(self) -> None:
        code, message = self.run_hook([
            user("tell me about the shared molecules"),
            assistant("The crossings are worth a look. Each crossings entry pairs two donors."),
        ])
        self.assertEqual(2, code)
        self.assertIn("crossings", message)

    def test_a_word_used_once_is_not_reported(self) -> None:
        code, _ = self.run_hook([
            user("tell me about the shared molecules"),
            assistant("The shared molecules include one crossings entry."),
        ])
        self.assertEqual(0, code)

    def test_a_word_the_user_used_is_not_new(self) -> None:
        code, _ = self.run_hook([
            user("what about the crossings?"),
            assistant("The crossings are listed below. Every crossings row has two donors."),
        ])
        self.assertEqual(0, code)

    def test_a_word_repeated_in_a_tool_result_is_not_new(self) -> None:
        code, _ = self.run_hook([
            user("read the file"),
            tool_result("column: crossings\nheader: crossings\nrows: 42"),
            assistant("The crossings column has 42 rows, so crossings are common."),
        ])
        self.assertEqual(0, code)

    def test_a_word_appearing_once_in_tool_output_does_not_count(self) -> None:
        """Base64 and hex are unique per appearance; a real name recurs."""
        code, message = self.run_hook([
            user("read the file"),
            tool_result("some output mentioning crossings exactly once"),
            assistant("The crossings matter here, and the crossings are new."),
        ])
        self.assertEqual(2, code)
        self.assertIn("crossings", message)

    def test_a_firing_is_logged(self) -> None:
        self.run_hook([
            user("go"),
            assistant("The crossings and the crossings again."),
        ])
        log = os.path.join(self.directory, ".claude", "termcheck", "firings.jsonl")
        self.assertTrue(os.path.exists(log), "no log written")
        with open(log) as handle:
            record = json.loads(handle.readline())
        self.assertIn("crossings", record["words"])
        self.assertIn("when", record)

    def test_silence_is_not_logged(self) -> None:
        self.run_hook([user("go"), assistant("Nothing new to say at all here.")])
        log = os.path.join(self.directory, ".claude", "termcheck", "firings.jsonl")
        self.assertFalse(os.path.exists(log))

    def test_old_caches_are_pruned(self) -> None:
        stale = os.path.join(self.directory, ".claude", "termcheck", "vocab-gone.txt")
        os.makedirs(os.path.dirname(stale), exist_ok=True)
        with open(stale, "w") as handle:
            handle.write("0\n")
        long_ago = datetime.datetime.now().timestamp() - 30 * 86400
        os.utime(stale, (long_ago, long_ago))
        self.run_hook([user("go"), assistant("Nothing new here at all.")])
        self.assertFalse(os.path.exists(stale), "stale cache was not pruned")

    def test_a_live_cache_is_not_pruned(self) -> None:
        fresh = os.path.join(self.directory, ".claude", "termcheck", "vocab-live.txt")
        os.makedirs(os.path.dirname(fresh), exist_ok=True)
        with open(fresh, "w") as handle:
            handle.write("0\n")
        self.run_hook([user("go"), assistant("Nothing new here at all.")])
        self.assertTrue(os.path.exists(fresh))

    def test_code_spans_are_not_prose(self) -> None:
        code, _ = self.run_hook([
            user("show me the call"),
            assistant("Use `calculate_crossings()` and then `calculate_crossings(x)` again."),
        ])
        self.assertEqual(0, code)

    def test_fenced_blocks_are_not_prose(self) -> None:
        code, _ = self.run_hook([
            user("show me"),
            assistant("Here:\n```\ncrossings = compute()\nprint(crossings)\n```\n"),
        ])
        self.assertEqual(0, code)

    def test_quoted_lines_are_not_prose(self) -> None:
        code, _ = self.run_hook([
            user("what did I say"),
            assistant("> the crossings matter\n> crossings again\n\nAgreed."),
        ])
        self.assertEqual(0, code)

    def test_display_math_is_not_prose(self) -> None:
        code, message = self.run_hook([
            user("show the ratio"),
            assistant("The ratio is\n$$\\frac{a}{b} \\qquad \\qquad (1)$$\nas above."),
        ])
        self.assertEqual(0, code, f"reported: {message}")

    def test_inline_math_is_not_prose(self) -> None:
        code, _ = self.run_hook([
            user("what is it"),
            assistant("With $\\lambda_{obs}$ fixed and $\\lambda_{obs}$ known, it holds."),
        ])
        self.assertEqual(0, code)

    def test_latex_commands_outside_math_are_not_prose(self) -> None:
        code, _ = self.run_hook([
            user("how do I space it"),
            assistant("Use \\qquad there, and \\qquad again after the fraction."),
        ])
        self.assertEqual(0, code)

    def test_html_tag_names_are_not_prose(self) -> None:
        """The tag is markup; the text between tags is prose and still counts."""
        code, _ = self.run_hook([
            user("show it"),
            assistant('Here <details open="true">x</details> and <details>y</details>.'),
        ])
        self.assertEqual(0, code)

    def test_text_inside_tags_still_counts(self) -> None:
        code, message = self.run_hook([
            user("show it"),
            assistant("Here <b>crossings</b> and also <b>crossings</b> again."),
        ])
        self.assertEqual(2, code)
        self.assertIn("crossings", message)

    def test_paths_are_not_prose(self) -> None:
        code, _ = self.run_hook([
            user("where is it"),
            assistant("At /tmp/crossings/out.csv and also at /var/crossings/in.csv."),
        ])
        self.assertEqual(0, code)

    def test_a_plural_of_a_seen_word_is_not_new(self) -> None:
        code, message = self.run_hook([
            user("the skill will spawn a subagent"),
            assistant("It spawns a subagent, and spawns another after that."),
        ])
        self.assertEqual(0, code, f"reported: {message}")

    def test_a_verb_form_of_a_seen_word_is_not_new(self) -> None:
        code, _ = self.run_hook([
            user("please review the draft"),
            assistant("I am reviewing it now, and reviewing the second one too."),
        ])
        self.assertEqual(0, code)

    def test_short_words_are_ignored(self) -> None:
        code, _ = self.run_hook([
            user("hello"),
            assistant("Yes ok ok."),
        ])
        self.assertEqual(0, code)

    def test_only_the_last_turn_is_checked(self) -> None:
        code, _ = self.run_hook([
            user("first"),
            assistant("The crossings and the crossings again."),
            user("second"),
            assistant("Nothing new to say here at all."),
        ])
        self.assertEqual(0, code)

    def test_stop_hook_active_does_not_loop(self) -> None:
        path = os.path.join(self.directory, "loop.jsonl")
        with open(path, "w") as handle:
            handle.write(json.dumps(assistant("crossings and crossings")) + "\n")
        payload = json.dumps({
            "session_id": "loop", "transcript_path": path, "stop_hook_active": True,
        })
        finished = subprocess.run(
            [sys.executable, HOOK], input=payload, capture_output=True, text=True,
            env=dict(os.environ, HOME=self.directory),
        )
        self.assertEqual(0, finished.returncode)

    def test_vocabulary_carries_across_runs(self) -> None:
        entries = [user("go"), assistant("The crossings and the crossings again.")]
        first, _ = self.run_hook(entries, session="carry")
        self.assertEqual(2, first)
        entries += [user("more"), assistant("More crossings, and crossings once more.")]
        second, _ = self.run_hook(entries, session="carry")
        self.assertEqual(0, second, "a word reported once should not be reported again")

    def test_missing_transcript_is_silent(self) -> None:
        payload = json.dumps({
            "session_id": "x", "transcript_path": "/nonexistent/t.jsonl",
            "stop_hook_active": False,
        })
        finished = subprocess.run(
            [sys.executable, HOOK], input=payload, capture_output=True, text=True,
            env=dict(os.environ, HOME=self.directory),
        )
        self.assertEqual(0, finished.returncode)


if __name__ == "__main__":
    unittest.main(verbosity=2)
