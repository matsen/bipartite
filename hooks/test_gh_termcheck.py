#!/usr/bin/env python3
"""Table-driven tests for gh-termcheck.py.

Run with `python3 hooks/test_gh_termcheck.py` or `make test-hooks`.

Every case asserts one of two outcomes:

  ALLOW   exit 0 -- the command runs
  REFUSE  exit 2 -- the command is blocked

A wrong ALLOW lets an unchecked body reach GitHub, so cases that could fail
open are the ones worth having. Each runs the real script in a subprocess
against a temporary stamp directory, so the caller's stamps are untouched.
"""

import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

HOOK = os.path.join(os.path.dirname(os.path.abspath(__file__)), "gh-termcheck.py")

ALLOW, REFUSE = "allow", "refuse"


class GhTermcheck(unittest.TestCase):
    def setUp(self) -> None:
        self.directory = tempfile.mkdtemp(prefix="termcheck-test-")
        self.stamps = os.path.join(self.directory, "stamps")
        os.makedirs(self.stamps)
        self.stamped = self.write("stamped.md", "a body that has been checked\n")
        self.unstamped = self.write("unstamped.md", "a body that has not\n")
        self.stamp(self.stamped)

    def tearDown(self) -> None:
        shutil.rmtree(self.directory)

    def write(self, name: str, text: str) -> str:
        path = os.path.join(self.directory, name)
        with open(path, "w") as handle:
            handle.write(text)
        return path

    def stamp(self, path: str) -> None:
        with open(path, "rb") as handle:
            digest = hashlib.sha256(handle.read()).hexdigest()
        open(os.path.join(self.stamps, digest), "w").close()

    def run_hook(self, command: str) -> int:
        environment = dict(os.environ, HOME=self.directory)
        os.makedirs(os.path.join(self.directory, ".claude"), exist_ok=True)
        target = os.path.join(self.directory, ".claude", "termcheck")
        if not os.path.exists(target):
            os.symlink(self.stamps, target)
        payload = json.dumps({"tool_name": "Bash", "tool_input": {"command": command}})
        finished = subprocess.run(
            [sys.executable, HOOK], input=payload, capture_output=True,
            text=True, env=environment,
        )
        return finished.returncode

    def check(self, expected: str, command: str) -> None:
        code = self.run_hook(command)
        got = ALLOW if code == 0 else REFUSE
        self.assertEqual(expected, got, f"{command!r} gave exit {code}")

    # Commands that carry no prose to GitHub.

    def test_unrelated_commands_pass(self) -> None:
        self.check(ALLOW, "ls -l")
        self.check(ALLOW, "git commit -m 'fix the thing'")

    def test_reads_pass(self) -> None:
        self.check(ALLOW, "gh api repos/o/r/issues/1 --jq .title")
        self.check(ALLOW, "gh issue view 42 --comments")
        self.check(ALLOW, "gh pr edit 12 --add-label bug")

    # The stamp itself.

    def test_stamped_body_passes(self) -> None:
        self.check(ALLOW, f"gh issue comment 42 --body-file {self.stamped}")

    def test_unstamped_body_is_refused(self) -> None:
        self.check(REFUSE, f"gh issue comment 42 --body-file {self.unstamped}")

    def test_editing_after_stamping_invalidates(self) -> None:
        with open(self.stamped, "a") as handle:
            handle.write("one more line\n")
        self.check(REFUSE, f"gh issue comment 42 --body-file {self.stamped}")

    def test_stamp_follows_content_not_path(self) -> None:
        copy = self.write("copy.md", open(self.stamped).read())
        self.check(ALLOW, f"gh issue comment 42 --body-file {copy}")

    # Inline bodies.

    def test_inline_bodies_are_refused(self) -> None:
        self.check(REFUSE, "gh issue comment 42 --body 'hello'")
        self.check(REFUSE, "gh pr create --title t -b 'hello'")
        self.check(REFUSE, "gh api repos/o/r/issues/1/comments -f body=hello")

    # Writing *about* these commands is not running them.

    def test_heredoc_contents_are_not_commands(self) -> None:
        self.check(ALLOW, "git commit -F - <<'EOF'\n"
                          "covers gh pr merge and gh api graphql now\n"
                          "EOF")

    def test_heredoc_writing_a_file_about_the_hook(self) -> None:
        self.check(ALLOW, "cat > /tmp/notes.md <<'EOF'\n"
                          "run `gh issue comment 42 --body-file FILE` after stamping\n"
                          "EOF")

    def test_a_real_command_after_a_heredoc_still_checks(self) -> None:
        self.check(REFUSE, "cat > /tmp/x.md <<'EOF'\n"
                           "some notes\n"
                           "EOF\n"
                           f"gh issue comment 42 --body-file {self.unstamped}")

    # Unparseable commands must fail closed, not open.

    def test_apostrophe_in_a_comment_refuses(self) -> None:
        self.check(REFUSE, f"gh issue comment 42 --body-file {self.stamped}  # don't")

    def test_unbalanced_quote_refuses(self) -> None:
        self.check(REFUSE, f'gh issue comment 42 --body-file {self.stamped} "oops')

    # Every body in the command, not just the first.

    def test_second_body_is_also_checked(self) -> None:
        self.check(REFUSE, f"gh issue comment 1 --body-file {self.stamped} && "
                           f"gh api repos/o/r/issues/2/comments -F body=@{self.unstamped}")

    def test_api_body_after_a_bodyless_subcommand(self) -> None:
        self.check(REFUSE, "gh pr create --title t --draft && "
                           f"gh api repos/o/r/issues/1/comments -F body=@{self.unstamped}")

    # Flag spellings.

    def test_equals_form_of_field_flag(self) -> None:
        self.check(REFUSE, f"gh api repos/o/r/issues/1/comments --field=body=@{self.unstamped}")
        self.check(ALLOW, f"gh api repos/o/r/issues/1/comments --field=body=@{self.stamped}")

    def test_equals_form_of_body_file(self) -> None:
        self.check(REFUSE, f"gh issue comment 42 --body-file={self.unstamped}")

    def test_body_flag_as_last_token(self) -> None:
        self.check(REFUSE, "echo x | xargs gh issue comment 42 --body-file")

    # Piped bodies.

    def test_piped_body(self) -> None:
        self.check(ALLOW, f"cat {self.stamped} | gh api repos/o/r/issues/1/comments -F body=@-")
        self.check(REFUSE, f"cat {self.unstamped} | gh api repos/o/r/issues/1/comments -F body=@-")

    def test_ambiguous_pipe_source_refuses(self) -> None:
        self.check(REFUSE, f"cat {self.stamped}; cat {self.unstamped} | "
                           "gh api repos/o/r/issues/1/comments -F body=@-")

    # GraphQL.

    def test_graphql_inline_body_refuses(self) -> None:
        self.check(REFUSE, 'gh api graphql -f query="mutation { addComment(input:{body:\\"hi\\"}) }"')

    def test_graphql_with_stamped_variable_passes(self) -> None:
        self.check(ALLOW, f'gh api graphql -F body=@{self.stamped} '
                          '-f query="mutation { addComment(input:{body: $body}) }"')

    def test_graphql_read_passes(self) -> None:
        self.check(ALLOW, 'gh api graphql -f query="{ repository(owner:\\"o\\", name:\\"r\\") { id } }"')

    # Other body-bearing subcommands.

    def test_release_notes_file(self) -> None:
        self.check(REFUSE, f"gh release create v1 --notes-file {self.unstamped}")
        self.check(ALLOW, f"gh release create v1 --notes-file {self.stamped}")

    def test_pr_merge_body(self) -> None:
        self.check(REFUSE, f"gh pr merge 5 --squash --body-file {self.unstamped}")

    # Paths.

    def test_relative_path_refuses(self) -> None:
        self.check(REFUSE, "cd /tmp && gh issue comment 42 --body-file body.md")

    def test_missing_file_refuses(self) -> None:
        self.check(REFUSE, "gh issue comment 42 --body-file /nonexistent/body.md")

    def test_shell_variable_in_path_is_expanded(self) -> None:
        os.environ["TERMCHECK_TEST_DIR"] = self.directory
        try:
            self.check(ALLOW, "gh issue comment 42 --body-file $TERMCHECK_TEST_DIR/stamped.md")
        finally:
            del os.environ["TERMCHECK_TEST_DIR"]

    # --input.

    def test_input_json(self) -> None:
        payload = self.write("body.json", json.dumps({"body": "checked"}))
        self.check(REFUSE, f"gh api repos/o/r/issues/1 --input {payload}")
        self.stamp(payload)
        self.check(ALLOW, f"gh api repos/o/r/issues/1 --input {payload}")

    def test_input_stdin_refuses(self) -> None:
        self.check(REFUSE, "echo '{}' | gh api repos/o/r/issues/1 --input -")


if __name__ == "__main__":
    unittest.main(verbosity=2)
